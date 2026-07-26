package messaging

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/shared/infrastructure/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutboxRepositoryImpl struct {
	baseDB *gorm.DB
}

func NewOutboxRepository(baseDB *gorm.DB) OutboxRepository {
	return &OutboxRepositoryImpl{baseDB: baseDB}
}

func (r *OutboxRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.baseDB
}

func (r *OutboxRepositoryImpl) Create(ctx context.Context, record *OutboxRecord) error {
	if err := r.getDB(ctx).Create(record).Error; err != nil {
		return err
	}
	return r.SendNotification(ctx)
}

func (r *OutboxRepositoryImpl) ClaimPending(ctx context.Context, limit int) ([]OutboxRecord, error) {
	var records []OutboxRecord
	err := r.baseDB.WithContext(ctx).
		Raw("SELECT * FROM outbox_records WHERE status = ? ORDER BY created_at ASC LIMIT ? FOR UPDATE SKIP LOCKED", OutboxStatusPending, limit).
		Scan(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to claim pending outbox records: %w", err)
	}
	return records, nil
}

func (r *OutboxRepositoryImpl) MarkPublished(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return r.baseDB.WithContext(ctx).
		Model(&OutboxRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       OutboxStatusPublished,
			"published_at": now,
		}).Error
}

func (r *OutboxRepositoryImpl) SendNotification(ctx context.Context) error {
	return r.baseDB.WithContext(ctx).
		Exec("SELECT pg_notify('outbox_channel', '')").Error
}

func (r *OutboxRepositoryImpl) MarkFailed(ctx context.Context, id uuid.UUID, err error) error {
	errMsg := err.Error()
	return r.baseDB.WithContext(ctx).
		Model(&OutboxRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     OutboxStatusFailed,
			"last_error": &errMsg,
		}).Error
}

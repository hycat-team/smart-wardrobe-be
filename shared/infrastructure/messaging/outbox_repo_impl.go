package messaging

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/shared/infrastructure/db"

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

// outboxStaleProcessingSeconds là thời gian 1 record bị kẹt ở 'processing'
// (worker crash giữa claim và publish) trước khi được reclaim.
const outboxStaleProcessingSeconds = 60

// ClaimToPublish claims up to `limit` outbox records ready to be published:
// pending records, failed records past their retry backoff, and stale
// `processing` records (crashed worker). The claim atomically flips status to
// `processing` inside the same transaction that holds the row locks, so two
// replicas can never claim the same record.
func (r *OutboxRepositoryImpl) ClaimToPublish(ctx context.Context, limit int, maxAttempts int, retryAfter time.Duration) ([]OutboxRecord, error) {
	var records []OutboxRecord
	retryAfterSeconds := int(retryAfter.Seconds())
	err := r.baseDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Raw(`UPDATE outbox_records SET status = ?, updated_at = NOW()
			WHERE id IN (
				SELECT id FROM outbox_records
				WHERE status = ?
				   OR (status = ? AND attempts < ? AND updated_at < NOW() - ? * INTERVAL '1 second')
				   OR (status = ? AND updated_at < NOW() - ? * INTERVAL '1 second')
				ORDER BY created_at ASC
				LIMIT ?
				FOR UPDATE SKIP LOCKED
			)
			RETURNING *`,
			OutboxStatusProcessing,
			OutboxStatusPending,
			OutboxStatusFailed, maxAttempts, retryAfterSeconds,
			OutboxStatusProcessing, outboxStaleProcessingSeconds,
			limit,
		).Scan(&records).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to claim outbox records ready to publish: %w", err)
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
			"updated_at":   now,
		}).Error
}

func (r *OutboxRepositoryImpl) CountPending(ctx context.Context) (int64, error) {
	var count int64
	err := r.baseDB.WithContext(ctx).Model(&OutboxRecord{}).
		Where("status = ?", OutboxStatusPending).
		Count(&count).Error
	return count, err
}

func (r *OutboxRepositoryImpl) SendNotification(ctx context.Context) error {
	return r.baseDB.WithContext(ctx).
		Exec("SELECT pg_notify('outbox_channel', '')").Error
}

func (r *OutboxRepositoryImpl) MarkFailed(ctx context.Context, id uuid.UUID, err error) error {
	errMsg := err.Error()
	now := time.Now().UTC()
	return r.baseDB.WithContext(ctx).
		Model(&OutboxRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     OutboxStatusFailed,
			"last_error": &errMsg,
			"attempts":   gorm.Expr("attempts + 1"),
			"updated_at": now,
		}).Error
}

func (r *OutboxRepositoryImpl) DeletePublishedBefore(ctx context.Context, since time.Time) (int64, error) {
	result := r.baseDB.WithContext(ctx).
		Where("status = ? AND published_at < ?", OutboxStatusPublished, since).
		Delete(&OutboxRecord{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old published outbox records: %w", result.Error)
	}
	return result.RowsAffected, nil
}

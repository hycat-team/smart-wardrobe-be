package persistence

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	"time"
)

type PaymentEventRepository struct{ db *gorm.DB }

func NewPaymentEventRepository(conn *gorm.DB) repositories.IPaymentEventRepository {
	return &PaymentEventRepository{db: conn}
}
func (r *PaymentEventRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *PaymentEventRepository) Create(ctx context.Context, event *entities.ProviderPaymentEvent) error {
	return r.current(ctx).Create(event).Error
}
func (r *PaymentEventRepository) GetByReference(ctx context.Context, provider, reference string) (*entities.ProviderPaymentEvent, error) {
	var event entities.ProviderPaymentEvent
	err := r.current(ctx).Where("provider = ? AND provider_reference = ?", provider, reference).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}

type WebhookInboxRepository struct{ db *gorm.DB }

func NewWebhookInboxRepository(conn *gorm.DB) repositories.IWebhookInboxRepository {
	return &WebhookInboxRepository{db: conn}
}
func (r *WebhookInboxRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *WebhookInboxRepository) Create(ctx context.Context, event *entities.ProviderWebhookInbox) error {
	return r.current(ctx).Create(event).Error
}
func (r *WebhookInboxRepository) GetByHash(ctx context.Context, provider, hash string) (*entities.ProviderWebhookInbox, error) {
	var event entities.ProviderWebhookInbox
	err := r.current(ctx).Where("provider = ? AND canonical_payload_hash = ?", provider, hash).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}
func (r *WebhookInboxRepository) GetByReference(ctx context.Context, provider, reference, eventCode string) (*entities.ProviderWebhookInbox, error) {
	var event entities.ProviderWebhookInbox
	err := r.current(ctx).Where("provider = ? AND provider_reference = ? AND event_code = ?", provider, reference, eventCode).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}
func (r *WebhookInboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.current(ctx).Model(&entities.ProviderWebhookInbox{}).Where("id = ?", id).Updates(map[string]any{"processing_status": "PROCESSED", "processed_at": at, "processing_token": nil, "processing_lease_until": nil}).Error
}
func (r *WebhookInboxRepository) MarkRetry(ctx context.Context, id uuid.UUID, at time.Time, message string) error {
	next := at.Add(time.Minute)
	return r.current(ctx).Model(&entities.ProviderWebhookInbox{}).Where("id = ?", id).Updates(map[string]any{"processing_status": "RETRY_REQUIRED", "processing_error": message, "processing_attempts": gorm.Expr("processing_attempts + 1"), "next_processing_at": next, "processing_token": nil, "processing_lease_until": nil}).Error
}
func (r *WebhookInboxRepository) MarkInvestigation(ctx context.Context, id uuid.UUID, at time.Time, message string) error {
	return r.current(ctx).Model(&entities.ProviderWebhookInbox{}).Where("id = ?", id).Updates(map[string]any{"processing_status": "INVESTIGATION_REQUIRED", "processing_error": message, "next_processing_at": nil, "processing_token": nil, "processing_lease_until": nil}).Error
}
func (r *WebhookInboxRepository) Claim(ctx context.Context, limit int, lease time.Duration) ([]*entities.ProviderWebhookInbox, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []*entities.ProviderWebhookInbox
	err := r.current(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("(processing_status IN ? AND (next_processing_at IS NULL OR next_processing_at <= NOW())) OR (processing_status = ? AND processing_lease_until <= NOW())", []string{"RECEIVED", "RETRY_REQUIRED"}, "PROCESSING").Order("received_at ASC").Limit(limit).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			token := uuid.New()
			row.ProcessingStatus = "PROCESSING"
			row.ProcessingToken = &token
			if err := tx.Model(row).Updates(map[string]any{"processing_status": "PROCESSING", "processing_token": token, "processing_lease_until": gorm.Expr("NOW() + (? * interval '1 second')", int(lease.Seconds()))}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return rows, err
}

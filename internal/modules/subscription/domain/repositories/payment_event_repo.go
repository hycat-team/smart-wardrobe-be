package repositories

import (
	"context"
	"github.com/google/uuid"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"time"
)

type IPaymentEventRepository interface {
	Create(ctx context.Context, event *entities.ProviderPaymentEvent) error
	GetByReference(ctx context.Context, provider, reference string) (*entities.ProviderPaymentEvent, error)
}

type IWebhookInboxRepository interface {
	Create(ctx context.Context, event *entities.ProviderWebhookInbox) error
	GetByHash(ctx context.Context, provider, hash string) (*entities.ProviderWebhookInbox, error)
	GetByReference(ctx context.Context, provider, reference, eventCode string) (*entities.ProviderWebhookInbox, error)
	MarkProcessed(ctx context.Context, id uuid.UUID, at time.Time) error
	MarkRetry(ctx context.Context, id uuid.UUID, at time.Time, message string) error
	MarkInvestigation(ctx context.Context, id uuid.UUID, at time.Time, message string) error
	Claim(ctx context.Context, limit int, lease time.Duration) ([]*entities.ProviderWebhookInbox, error)
}

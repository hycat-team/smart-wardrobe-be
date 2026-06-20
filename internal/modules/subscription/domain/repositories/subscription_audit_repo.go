package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

type IUserSubscriptionEventRepository interface {
	Create(ctx context.Context, event *entities.UserSubscriptionEvent) error
	GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error)
}

type ISubscriptionRenewalAttemptRepository interface {
	Create(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error
	GetByKey(ctx context.Context, key string) (*entities.SubscriptionRenewalAttempt, error)
	Update(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error
}

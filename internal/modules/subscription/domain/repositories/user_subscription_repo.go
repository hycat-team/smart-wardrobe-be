package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

// IUserSubscriptionRepository handles database operations for user subscriptions
type IUserSubscriptionRepository interface {
	repositories.IGenericRepository[entities.UserSubscription, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error)
	GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error)
	GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error)
	GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error)
	GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error)
}

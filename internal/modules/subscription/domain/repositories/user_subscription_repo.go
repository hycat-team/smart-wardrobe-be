package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

// IUserSubscriptionRepository handles database operations for user subscriptions
type IUserSubscriptionRepository interface {
	repositories.IGenericRepository[entities.UserSubscription, uuid.UUID]
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error)
}

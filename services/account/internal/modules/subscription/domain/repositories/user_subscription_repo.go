package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type AdminUserSubscriptionFilter struct {
	PlanCode *string
	Status   *usersubscriptionstatus.UserSubscriptionStatus
	UserID   *uuid.UUID
}

// IUserSubscriptionRepository handles database operations for user subscriptions
type IUserSubscriptionRepository interface {
	repositories.IGenericRepository[entities.UserSubscription, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error)
	GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*entities.UserSubscription, error)
	GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error)
	GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error)
	GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error)
	GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error)
	GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error)
	// 024 T147: `BulkCreate` is GONE. FR-014 needs a history row for every account
	// in a bulk provision, which `ProvisionDefault` plus `PlanChangePublisher` do per
	// account inside one transaction. The plain bulk INSERT this replaced skipped
	// both - no history row and no event - so it could not have satisfied FR-014
	// even if something had called it, and nothing did.
	ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error
	GetDistributionByPlanCode(ctx context.Context) ([]SubscriptionDistributionDTO, error)
	ListAdmin(ctx context.Context, filter AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error)
}

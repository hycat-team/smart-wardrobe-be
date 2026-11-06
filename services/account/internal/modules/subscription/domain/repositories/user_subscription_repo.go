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
	// GetSubscriptionsExpiringWithinBatch pages through finite plans ending in
	// [from, to]. Cursor-paged for the same reason the expiry scan is: the
	// reminder used to load every row in the window and then look up one wallet
	// per row, which is a whole night's worth of queries on a large window.
	// Order is (expires_at, user_id), matching the cursor predicate.
	GetSubscriptionsExpiringWithinBatch(ctx context.Context, from, to time.Time, lastExpiresAt time.Time, lastUserID uuid.UUID, limit int) ([]*entities.UserSubscription, error)
	// 024 T147: `BulkCreate` is GONE. FR-014 needs a history row for every account
	// in a bulk provision, which `ProvisionDefault` plus `PlanChangePublisher` do per
	// account inside one transaction. The plain bulk INSERT this replaced skipped
	// both - no history row and no event - so it could not have satisfied FR-014
	// even if something had called it, and nothing did.
	ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error
	// 031 T019: `GetDistributionByPlanCode` is GONE. The projection can only
	// answer "what plan is this user on now", so it silently dropped users
	// who changed plan after the reported period. The distribution report now
	// reads `subscription.user_plan_periods` through
	// IUserPlanPeriodRepository.GetDistribution; keeping a second, wrong
	// counting path here would only invite someone to call it again.
	ListAdmin(ctx context.Context, filter AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error)
	// ListUserIDsWithEffectivePlan answers "which accounts are on plan X right
	// now" for the admin user list filter.
	//
	// It mirrors the effective-plan rule of `GetUserSubscriptionOverviews`: a
	// finite plan that has already expired counts as its fallback plan, or as
	// `defaultPlanID` when no fallback was set. The rule lives in two places on
	// purpose — the mapper resolves it in Go for each row it renders, this
	// query resolves it in SQL so the identity module can paginate — and the
	// consistency test in `user_subscription_repo_effective_plan_test.go`
	// pins the two together.
	//
	// The result is the WHOLE id set, not a page: callers paginate the
	// identity table themselves and cannot page a filter.
	ListUserIDsWithEffectivePlan(ctx context.Context, planID, defaultPlanID uuid.UUID) ([]uuid.UUID, error)
}

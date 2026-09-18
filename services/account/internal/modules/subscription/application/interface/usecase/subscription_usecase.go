package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/shared/observability/workerlog"

	"github.com/google/uuid"
)

type ISubscriptionUseCase interface {
	GetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewRes, error)
	GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionOverviewRes, error)
	ProcessScheduledRenewals(ctx context.Context, run *workerlog.Run) error
	ProcessUpcomingExpiryNotifications(ctx context.Context) error
	SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error)
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanBriefRes, error)
	AdminList(ctx context.Context, planCode *string, status *usersubscriptionstatus.UserSubscriptionStatus, page, limit int) ([]*dto.AdminUserSubscriptionItemDTO, int64, error)
	AdminCancel(ctx context.Context, userID uuid.UUID) error
	AdminGrant(ctx context.Context, userID uuid.UUID, planID uuid.UUID) error
}

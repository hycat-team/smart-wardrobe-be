package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
)

type ISubscriptionUseCase interface {
	GetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewDTO, error)
	GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionOverviewDTO, error)
	ProcessScheduledRenewals(ctx context.Context, run *workerlog.Run) error
	ProcessUpcomingExpiryNotifications(ctx context.Context) error
	SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error)
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error)
}

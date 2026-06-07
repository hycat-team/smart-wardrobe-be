package contract

import (
	"context"

	"github.com/google/uuid"
)

// ISubscriptionPlanContract retrieves plan configuration details
type ISubscriptionPlanContract interface {
	GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error)
	IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error)
}

// IUserSubscriptionContract manages active user subscriptions and overview queries
type IUserSubscriptionContract interface {
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*UserSubscriptionDTO, error)
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*UserSubscriptionOverviewDTO, error)
	GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*UserSubscriptionOverviewDTO, error)
}

// IUserQuotaContract manages daily quota evaluations, updates, and constraints
type IUserQuotaContract interface {
	GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*UserSubscriptionDTO, error)
	UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error
	UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error
}

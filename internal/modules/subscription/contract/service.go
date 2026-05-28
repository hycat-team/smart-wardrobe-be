package contract

import (
	"context"

	"github.com/google/uuid"
)

// ISubscriptionModuleContract exposes core functions for subscription lifecycle and quota checks
type ISubscriptionModuleContract interface {
	GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error)
	IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error)
	InitializeUserSubscription(ctx context.Context, userID uuid.UUID) error
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*UserSubscriptionDTO, error)
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*UserSubscriptionOverviewDTO, error)
	GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*UserSubscriptionDTO, error)
	UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error
	UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error
	ResetDailyQuotas(ctx context.Context, userID uuid.UUID) error
}

package contract

import (
	"context"

	"smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
)

// IUserSubscriptionContract manages active user subscriptions and overview queries
type IUserSubscriptionContract interface {
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionContractDTO, error)
	GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionContractDTO, error)
}

// IUserQuotaContract manages daily quota evaluations, updates, and constraints
type IUserQuotaContract interface {
	GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error
	UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error
}

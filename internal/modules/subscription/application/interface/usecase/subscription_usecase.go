package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/contract"

	"github.com/google/uuid"
)

type ISubscriptionUseCase interface {
	GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error)
	ProcessScheduledRenewals(ctx context.Context) error
	ToggleAutoRenew(ctx context.Context, userID uuid.UUID) (bool, error)
}

package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/contract"

	"github.com/google/uuid"
)

type ISubscriptionUseCase interface {
	GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error)
	ProcessScheduledRenewals(ctx context.Context) error
	SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error)
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error)
}

package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"

	"github.com/google/uuid"
)

type ISubscriptionPlanUseCase interface {
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error)
	GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error)
	IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error)
}

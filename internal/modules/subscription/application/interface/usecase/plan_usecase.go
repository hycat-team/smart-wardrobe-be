package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/contract"
)

type ISubscriptionPlanUseCase interface {
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error)

	contract.ISubscriptionPlanContract
}

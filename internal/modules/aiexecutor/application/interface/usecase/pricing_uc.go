package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
)

type IPricingUseCase interface {
	GetActivePricingConfig(ctx context.Context, provider string, model string) (*dto.PricingConfigDTO, error)
}

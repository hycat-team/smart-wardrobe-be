package usecase

import (
	"context"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
)

type IPricingUseCase interface {
	GetActivePricingConfig(ctx context.Context, provider string, model string) (*dto.PricingConfigDTO, error)
}

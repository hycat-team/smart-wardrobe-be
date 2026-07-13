package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"
)

type IPricingUseCase interface {
	GetActivePricingConfig(ctx context.Context, provider aiprovider.Provider, model aimodel.Model) (*dto.PricingConfigDTO, error)
}

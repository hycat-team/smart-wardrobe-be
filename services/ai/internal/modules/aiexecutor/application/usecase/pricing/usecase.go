package pricing

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	aiexecutor_caching "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	aiexecutor_mapper "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/mapper"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
)

const pricingCacheTTL = 30 * time.Minute

type PricingUseCase struct {
	repo         repositories.IPricingConfigRepository
	cacheService aiexecutor_caching.IPricingCacheService
}

func NewPricingUseCase(
	repo repositories.IPricingConfigRepository,
	cacheService aiexecutor_caching.IPricingCacheService,
) uc_interfaces.IPricingUseCase {
	return &PricingUseCase{
		repo:         repo,
		cacheService: cacheService,
	}
}

func (uc *PricingUseCase) GetActivePricingConfig(ctx context.Context, provider string, model string) (*aiexecutor_dto.PricingConfigDTO, error) {
	cached, err := uc.cacheService.Get(ctx, provider, model)
	if err == nil && cached != nil {
		return cached, nil
	}

	cfg, err := uc.repo.GetActiveByProviderAndModel(ctx, provider, model)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("no active pricing config for provider %s / model %s: this is a deployment error", provider, model)
	}

	dto := aiexecutor_mapper.PricingConfigToDTO(cfg)

	_ = uc.cacheService.Set(ctx, provider, model, dto, pricingCacheTTL)

	return dto, nil
}

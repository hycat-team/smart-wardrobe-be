package pricing

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	aiexecutor_caching "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
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

	dto := mapToDTO(cfg)

	_ = uc.cacheService.Set(ctx, provider, model, dto, pricingCacheTTL)

	return dto, nil
}

func mapToDTO(cfg *entities.AIModelConfig) *aiexecutor_dto.PricingConfigDTO {
	if cfg == nil {
		return nil
	}
	return &aiexecutor_dto.PricingConfigDTO{
		ID:                  cfg.ID,
		Provider:            cfg.Provider,
		Model:               cfg.Model,
		InputUSDPerMillion:  cfg.InputUSDPerMillion,
		OutputUSDPerMillion: cfg.OutputUSDPerMillion,
		Currency:            cfg.Currency,
		USDToVND:            cfg.USDToVND,
		Version:             cfg.Version,
		EffectiveFrom:       cfg.EffectiveFrom,
		EffectiveTo:         cfg.EffectiveTo,
	}
}

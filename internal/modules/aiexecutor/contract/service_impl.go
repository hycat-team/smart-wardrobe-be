package contract

import (
	"context"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
)

type PricingConfigService struct {
	repo repositories.IPricingConfigRepository
}

func NewPricingConfigService(repo repositories.IPricingConfigRepository) IAIExecutorContract {
	return &PricingConfigService{repo: repo}
}

func mapToPricingConfigDTO(cfg *entities.AIPricingConfig) *PricingConfigDTO {
	if cfg == nil {
		return nil
	}
	return &PricingConfigDTO{
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

func (s *PricingConfigService) GetActivePricingConfig(ctx context.Context, provider string, model string) (*PricingConfigDTO, error) {
	cfg, err := s.repo.GetActiveByProviderAndModel(ctx, provider, model)
	if err != nil {
		return nil, err
	}
	return mapToPricingConfigDTO(cfg), nil
}

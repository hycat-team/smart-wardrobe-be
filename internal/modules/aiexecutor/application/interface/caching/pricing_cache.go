package caching

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"
)

type IPricingCacheService interface {
	Get(ctx context.Context, provider aiprovider.Provider, model aimodel.Model) (*dto.PricingConfigDTO, error)
	Set(ctx context.Context, provider aiprovider.Provider, model aimodel.Model, pricing *dto.PricingConfigDTO, expiry time.Duration) error
}

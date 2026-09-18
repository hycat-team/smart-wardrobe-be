package caching

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
)

type IPricingCacheService interface {
	Get(ctx context.Context, provider string, model string) (*dto.PricingConfigDTO, error)
	Set(ctx context.Context, provider string, model string, pricing *dto.PricingConfigDTO, expiry time.Duration) error
}

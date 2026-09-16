package caching

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	aiexecutor_caching "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/caching"
	shared_caching "smart-wardrobe-be/shared/infrastructure/caching"
)

type RedisPricingCacheService struct {
	cache shared_caching.ICacheService
}

func NewRedisPricingCacheService(cache shared_caching.ICacheService) aiexecutor_caching.IPricingCacheService {
	return &RedisPricingCacheService{
		cache: cache,
	}
}

func (s *RedisPricingCacheService) Get(ctx context.Context, provider string, model string) (*dto.PricingConfigDTO, error) {
	if s.cache == nil {
		return nil, nil
	}
	key := pricingCacheKey(provider, model)
	var res dto.PricingConfigDTO
	err := s.cache.Get(ctx, key, &res)
	if err != nil {
		if errors.Is(err, shared_caching.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (s *RedisPricingCacheService) Set(ctx context.Context, provider string, model string, pricing *dto.PricingConfigDTO, expiry time.Duration) error {
	if s.cache == nil {
		return nil
	}
	key := pricingCacheKey(provider, model)
	return s.cache.Set(ctx, key, pricing, expiry)
}

func pricingCacheKey(provider string, model string) string {
	return "ai:pricing:" + provider + ":" + model
}

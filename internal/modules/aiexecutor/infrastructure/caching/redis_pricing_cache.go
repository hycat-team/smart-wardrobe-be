package caching

import (
	"context"
	"encoding/json"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	aiexecutor_caching "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/caching"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"

	"github.com/redis/go-redis/v9"
)

type RedisPricingCacheService struct {
	redisClient *redis.Client
}

func NewRedisPricingCacheService(client *redis.Client) aiexecutor_caching.IPricingCacheService {
	return &RedisPricingCacheService{
		redisClient: client,
	}
}

func (s *RedisPricingCacheService) Get(ctx context.Context, provider aiprovider.Provider, model aimodel.Model) (*dto.PricingConfigDTO, error) {
	if s.redisClient == nil {
		return nil, nil
	}
	key := pricingCacheKey(provider, model)
	cached, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var res dto.PricingConfigDTO
	if err := json.Unmarshal([]byte(cached), &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *RedisPricingCacheService) Set(ctx context.Context, provider aiprovider.Provider, model aimodel.Model, pricing *dto.PricingConfigDTO, expiry time.Duration) error {
	if s.redisClient == nil {
		return nil
	}
	key := pricingCacheKey(provider, model)
	data, err := json.Marshal(pricing)
	if err != nil {
		return err
	}
	return s.redisClient.Set(ctx, key, data, expiry).Err()
}

func pricingCacheKey(provider aiprovider.Provider, model aimodel.Model) string {
	return "ai:pricing:" + string(provider) + ":" + string(model)
}

package caching

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/internal/config"
	ucaching "smart-wardrobe-be/internal/modules/brand/application/interface/caching"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisBrandFeatureAccessCache struct {
	redisClient *redis.Client
	ttl         time.Duration
	hitCount    atomic.Uint64
	missCount   atomic.Uint64
}

func NewRedisBrandFeatureAccessCache(client *redis.Client, cfg *config.Config) ucaching.IBrandFeatureAccessCache {
	return &RedisBrandFeatureAccessCache{
		redisClient: client,
		ttl:         cfg.BrandFeatureAccess.TTL,
	}
}

func (s *RedisBrandFeatureAccessCache) HitCount() uint64 {
	return s.hitCount.Load()
}

func (s *RedisBrandFeatureAccessCache) MissCount() uint64 {
	return s.missCount.Load()
}

func (s *RedisBrandFeatureAccessCache) Get(ctx context.Context, userID, brandID uuid.UUID) ([]string, error) {
	if s.redisClient == nil {
		s.missCount.Add(1)
		return nil, nil
	}
	key := featureAccessCacheKey(userID, brandID)
	cached, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		s.missCount.Add(1)
		return nil, nil
	}
	var featureCodes []string
	if err := json.Unmarshal([]byte(cached), &featureCodes); err != nil {
		s.missCount.Add(1)
		return nil, nil
	}
	s.hitCount.Add(1)
	return featureCodes, nil
}

func (s *RedisBrandFeatureAccessCache) Set(ctx context.Context, userID, brandID uuid.UUID, featureCodes []string) error {
	if s.redisClient == nil {
		return nil
	}
	data, err := json.Marshal(featureCodes)
	if err != nil {
		return err
	}
	return s.redisClient.Set(ctx, featureAccessCacheKey(userID, brandID), data, s.ttl).Err()
}

func (s *RedisBrandFeatureAccessCache) Invalidate(ctx context.Context, userID, brandID uuid.UUID) error {
	if s.redisClient == nil {
		return nil
	}
	return s.redisClient.Del(ctx, featureAccessCacheKey(userID, brandID)).Err()
}

func (s *RedisBrandFeatureAccessCache) InvalidateByBrand(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) error {
	if s.redisClient == nil {
		return nil
	}
	if len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(userIDs))
	for _, uid := range userIDs {
		keys = append(keys, featureAccessCacheKey(uid, brandID))
	}
	return s.redisClient.Del(ctx, keys...).Err()
}

func featureAccessCacheKey(userID, brandID uuid.UUID) string {
	return fmt.Sprintf("brand:feature_access:%s:%s", userID.String(), brandID.String())
}

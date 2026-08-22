package caching

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/internal/config"
	ucaching "smart-wardrobe-be/internal/modules/brand/application/interface/caching"
	shared_caching "smart-wardrobe-be/internal/shared/infrastructure/caching"

	"github.com/google/uuid"
)

type RedisBrandFeatureAccessCache struct {
	cache     shared_caching.ICacheService
	ttl       time.Duration
	hitCount  atomic.Uint64
	missCount atomic.Uint64
}

func NewRedisBrandFeatureAccessCache(cache shared_caching.ICacheService, cfg *config.Config) ucaching.IBrandFeatureAccessCache {
	return &RedisBrandFeatureAccessCache{
		cache: cache,
		ttl:   cfg.BrandFeatureAccess.TTL,
	}
}

func (s *RedisBrandFeatureAccessCache) HitCount() uint64 {
	return s.hitCount.Load()
}

func (s *RedisBrandFeatureAccessCache) MissCount() uint64 {
	return s.missCount.Load()
}

func (s *RedisBrandFeatureAccessCache) Get(ctx context.Context, userID, brandID uuid.UUID) ([]string, error) {
	if s.cache == nil {
		s.missCount.Add(1)
		return nil, nil
	}
	key := featureAccessCacheKey(userID, brandID)
	var featureCodes []string
	err := s.cache.Get(ctx, key, &featureCodes)
	if err != nil {
		if errors.Is(err, shared_caching.ErrKeyNotFound) {
			s.missCount.Add(1)
			return nil, nil
		}
		s.missCount.Add(1)
		return nil, nil
	}
	s.hitCount.Add(1)
	return featureCodes, nil
}

func (s *RedisBrandFeatureAccessCache) Set(ctx context.Context, userID, brandID uuid.UUID, featureCodes []string) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Set(ctx, featureAccessCacheKey(userID, brandID), featureCodes, s.ttl)
}

func (s *RedisBrandFeatureAccessCache) Invalidate(ctx context.Context, userID, brandID uuid.UUID) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Del(ctx, featureAccessCacheKey(userID, brandID))
}

func (s *RedisBrandFeatureAccessCache) InvalidateByBrand(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) error {
	if s.cache == nil {
		return nil
	}
	if len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(userIDs))
	for _, uid := range userIDs {
		keys = append(keys, featureAccessCacheKey(uid, brandID))
	}
	return s.cache.Del(ctx, keys...)
}

func featureAccessCacheKey(userID, brandID uuid.UUID) string {
	return fmt.Sprintf("brand:feature_access:%s:%s", userID.String(), brandID.String())
}

package caching

import (
	"context"
	"fmt"
	"time"

	ucaching "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/caching"
	shared_caching "smart-wardrobe-be/shared/infrastructure/caching"

	"github.com/google/uuid"
)

const personalizationRecommendationSnapshotTTL = 7 * 24 * time.Hour

// RedisRecommendationSnapshotCache implements the recommendation snapshot
// cache used by the feedback endpoint (033 US2/US6, plan D8).
//
// Every method is fail-soft: a nil cache, a miss or a backend error yields a
// cache miss so the caller answers "snapshot not found".
type RedisRecommendationSnapshotCache struct {
	cache shared_caching.ICacheService
	ttl   time.Duration
}

func NewRedisRecommendationSnapshotCache(cache shared_caching.ICacheService) ucaching.IRecommendationSnapshotCache {
	return &RedisRecommendationSnapshotCache{
		cache: cache,
		ttl:   personalizationRecommendationSnapshotTTL,
	}
}

func (s *RedisRecommendationSnapshotCache) Set(
	ctx context.Context,
	recommendationID uuid.UUID,
	snapshot *ucaching.RecommendationSnapshot,
) error {
	if s.cache == nil || snapshot == nil || recommendationID == uuid.Nil {
		return nil
	}
	return s.cache.Set(ctx, recommendationSnapshotCacheKey(recommendationID), snapshot, s.ttl)
}

func (s *RedisRecommendationSnapshotCache) Get(
	ctx context.Context,
	recommendationID uuid.UUID,
) (*ucaching.RecommendationSnapshot, error) {
	if s.cache == nil || recommendationID == uuid.Nil {
		return nil, nil
	}
	var snapshot ucaching.RecommendationSnapshot
	if err := s.cache.Get(ctx, recommendationSnapshotCacheKey(recommendationID), &snapshot); err != nil {
		return nil, nil
	}
	if snapshot.UserID == uuid.Nil {
		return nil, nil
	}
	return &snapshot, nil
}

func (s *RedisRecommendationSnapshotCache) Delete(ctx context.Context, recommendationID uuid.UUID) error {
	if s.cache == nil || recommendationID == uuid.Nil {
		return nil
	}
	return s.cache.Del(ctx, recommendationSnapshotCacheKey(recommendationID))
}

func recommendationSnapshotCacheKey(recommendationID uuid.UUID) string {
	return fmt.Sprintf("personalization:rec:%s", recommendationID.String())
}

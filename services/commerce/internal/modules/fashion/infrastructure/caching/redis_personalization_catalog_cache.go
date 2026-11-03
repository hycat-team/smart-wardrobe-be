package caching

import (
	"context"
	"time"

	ucaching "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/caching"
	shared_caching "smart-wardrobe-be/shared/infrastructure/caching"

	"github.com/google/uuid"
)

const (
	personalizationCatalogCacheKey = "personalization:catalog:v1"
	personalizationCatalogCacheTTL = 5 * time.Minute
)

type personalizationCatalogCacheEntry struct {
	IDs       []uuid.UUID `json:"ids"`
	FetchedAt time.Time   `json:"fetchedAt"`
}

// RedisPersonalizationCatalogCache implements the catalog id cache used to
// build gap-filling suggestions (033 US2, plan D5).
//
// Every method is fail-soft: a nil cache, a miss or a backend error yields a
// cache miss so the caller falls back to the Wardrobe contract.
type RedisPersonalizationCatalogCache struct {
	cache shared_caching.ICacheService
	ttl   time.Duration
}

func NewRedisPersonalizationCatalogCache(cache shared_caching.ICacheService) ucaching.IPersonalizationCatalogCache {
	return &RedisPersonalizationCatalogCache{
		cache: cache,
		ttl:   personalizationCatalogCacheTTL,
	}
}

func (s *RedisPersonalizationCatalogCache) Get(ctx context.Context) ([]uuid.UUID, error) {
	if s.cache == nil {
		return nil, nil
	}
	var entry personalizationCatalogCacheEntry
	if err := s.cache.Get(ctx, personalizationCatalogCacheKey, &entry); err != nil {
		return nil, nil
	}
	if len(entry.IDs) == 0 {
		return nil, nil
	}
	return entry.IDs, nil
}

func (s *RedisPersonalizationCatalogCache) Set(ctx context.Context, ids []uuid.UUID) error {
	if s.cache == nil || len(ids) == 0 {
		return nil
	}
	return s.cache.Set(ctx, personalizationCatalogCacheKey, personalizationCatalogCacheEntry{
		IDs:       ids,
		FetchedAt: time.Now().UTC(),
	}, s.ttl)
}

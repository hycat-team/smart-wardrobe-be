package caching

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/config"
	subscriptioncaching "smart-wardrobe-be/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_caching "smart-wardrobe-be/internal/shared/infrastructure/caching"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const defaultOverviewTTL = 60 * time.Second

// RedisSubscriptionOverviewCache stores the per-user subscription overview in
// Redis. Every failure degrades to a cache miss so callers fall back to the
// use case directly (spec 019 FR-008).
type RedisSubscriptionOverviewCache struct {
	cache shared_caching.ICacheService
	ttl   time.Duration
	log   logger.Interface
}

func NewRedisSubscriptionOverviewCache(cache shared_caching.ICacheService, cfg *config.Config, log logger.Interface) subscriptioncaching.ISubscriptionOverviewCache {
	ttl := defaultOverviewTTL
	if cfg.Cache.OverviewTTLSeconds > 0 {
		ttl = time.Duration(cfg.Cache.OverviewTTLSeconds) * time.Second
	}
	return &RedisSubscriptionOverviewCache{cache: cache, ttl: ttl, log: log}
}

func overviewCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("sub:overview:%s", userID.String())
}

func (c *RedisSubscriptionOverviewCache) Get(ctx context.Context, userID uuid.UUID) (*subscriptioncaching.SubscriptionOverviewCacheEntry, error) {
	if c.cache == nil {
		return nil, nil
	}
	var entry subscriptioncaching.SubscriptionOverviewCacheEntry
	if err := c.cache.Get(ctx, overviewCacheKey(userID), &entry); err != nil {
		return nil, nil
	}
	if entry.PlanID == uuid.Nil {
		return nil, nil
	}
	return &entry, nil
}

func (c *RedisSubscriptionOverviewCache) Set(ctx context.Context, userID uuid.UUID, entry *subscriptioncaching.SubscriptionOverviewCacheEntry) error {
	if c.cache == nil || entry == nil {
		return nil
	}
	entry.CachedAt = time.Now().UTC()
	return c.cache.Set(ctx, overviewCacheKey(userID), entry, c.ttl)
}

func (c *RedisSubscriptionOverviewCache) Invalidate(ctx context.Context, userID uuid.UUID) error {
	if c.cache == nil {
		return nil
	}
	c.log.Info("Invalidated subscription overview cache",
		zap.String("user_id", userID.String()),
	)
	return c.cache.Del(ctx, overviewCacheKey(userID))
}

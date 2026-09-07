package caching_test

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptioncaching "smart-wardrobe-be/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/internal/modules/subscription/application/mapper"
	subscription_cache "smart-wardrobe-be/internal/modules/subscription/infrastructure/caching"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// NewSubscriptionOverviewCache with a nil backing cache must degrade to a
// no-op: Get returns miss, Set/Invalidate are safe no-ops (spec 019 FR-008).
func TestRedisSubscriptionOverviewCache_NilClientFallback(t *testing.T) {
	cfg := &config.Config{Cache: config.CacheConfig{OverviewTTLSeconds: 60}}
	cache := subscription_cache.NewRedisSubscriptionOverviewCache(nil, cfg, nil)
	ctx := context.Background()
	userID := uuid.New()

	entry, err := cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, entry)

	assert.NoError(t, cache.Set(ctx, userID, &subscriptioncaching.SubscriptionOverviewCacheEntry{PlanID: uuid.New()}))
	assert.NoError(t, cache.Invalidate(ctx, userID))
}

// Round-trip through the mapper proves the cached entry preserves the full
// contract payload shape (zero field drift between cache and contract).
func TestSubscriptionOverviewCacheEntryRoundTrip(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour).UTC()
	contractDTO := &dto.UserSubscriptionContractDTO{
		PlanID:             uuid.New(),
		PlanName:           "Premium",
		PlanSlug:           "premium",
		ExpiresAt:          &expires,
		IsAutoRenewEnabled: true,
		MaxWardrobeItems:   100,
		MaxOutfits:         50,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
	}

	entry := mapper.ContractDTOToCacheEntry(contractDTO)
	assert.NotNil(t, entry)
	assert.Equal(t, contractDTO.PlanID, entry.PlanID)
	assert.Equal(t, contractDTO.IsAutoRenewEnabled, entry.IsAutoRenewEnabled)
	assert.Equal(t, contractDTO.AiOutfitDailyQuota, entry.AiOutfitDailyQuota)
	assert.False(t, entry.CachedAt.IsZero())

	got := mapper.CacheEntryToContractDTO(entry)
	assert.Equal(t, contractDTO, got)
}

// Mapper must not fabricate payloads from nil inputs.
func TestOverviewCacheMappersHandleNil(t *testing.T) {
	assert.Nil(t, mapper.ContractDTOToCacheEntry(nil))
	assert.Nil(t, mapper.CacheEntryToContractDTO(nil))
}

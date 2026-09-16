package caching_test

import (
	"context"
	"testing"
	"time"

	brand_cache "smart-wardrobe-be/services/commerce/internal/modules/brand/infrastructure/caching"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRedisBrandMembershipCache_NilClientFallback(t *testing.T) {
	cfg := &config.Config{
		BrandMembership: config.BrandMembershipCacheTTL{
			TTL: 10 * time.Minute,
		},
	}
	cache := brand_cache.NewRedisBrandMembershipCache(nil, cfg)
	ctx := context.Background()
	userID := uuid.New()
	brandID := uuid.New()

	// Get on nil client -> miss, no error
	memberships, err := cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, memberships)

	// Set on nil client -> no error
	err = cache.Set(ctx, userID, map[uuid.UUID]brandmemberrole.BrandMemberRole{
		brandID: brandmemberrole.Owner,
	})
	assert.NoError(t, err)

	// InvalidateByUserID on nil client -> no error
	err = cache.InvalidateByUserID(ctx, userID)
	assert.NoError(t, err)

	// InvalidateByBrandID on nil client -> no error
	err = cache.InvalidateByBrandID(ctx, brandID, []uuid.UUID{userID})
	assert.NoError(t, err)
}

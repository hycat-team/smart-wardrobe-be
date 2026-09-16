package caching

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/caching"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"
	shared_caching "smart-wardrobe-be/shared/infrastructure/caching"

	"github.com/google/uuid"
)

type RedisBrandMembershipCache struct {
	cache     shared_caching.ICacheService
	ttl       time.Duration
	hitCount  atomic.Uint64
	missCount atomic.Uint64
}

func NewRedisBrandMembershipCache(cache shared_caching.ICacheService, cfg *config.Config) caching.IBrandMembershipCache {
	return &RedisBrandMembershipCache{
		cache: cache,
		ttl:   cfg.BrandMembership.TTL,
	}
}

func (s *RedisBrandMembershipCache) HitCount() uint64 {
	return s.hitCount.Load()
}

func (s *RedisBrandMembershipCache) MissCount() uint64 {
	return s.missCount.Load()
}

func (s *RedisBrandMembershipCache) Get(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]brandmemberrole.BrandMemberRole, error) {
	if s.cache == nil {
		s.missCount.Add(1)
		return nil, nil
	}
	key := brandMembershipCacheKey(userID)
	var raw map[string]string
	err := s.cache.Get(ctx, key, &raw)
	if err != nil {
		if errors.Is(err, shared_caching.ErrKeyNotFound) {
			s.missCount.Add(1)
			return nil, nil
		}
		s.missCount.Add(1)
		return nil, nil
	}
	if len(raw) == 0 {
		s.missCount.Add(1)
		return nil, nil
	}
	s.hitCount.Add(1)
	memberships := make(map[uuid.UUID]brandmemberrole.BrandMemberRole, len(raw))
	for brandIDStr, role := range raw {
		brandID, err := uuid.Parse(brandIDStr)
		if err != nil {
			continue
		}
		memberships[brandID] = brandmemberrole.BrandMemberRole(role)
	}
	return memberships, nil
}

func (s *RedisBrandMembershipCache) Set(ctx context.Context, userID uuid.UUID, memberships map[uuid.UUID]brandmemberrole.BrandMemberRole) error {
	if s.cache == nil {
		return nil
	}
	raw := make(map[string]string, len(memberships))
	for brandID, role := range memberships {
		raw[brandID.String()] = string(role)
	}
	return s.cache.Set(ctx, brandMembershipCacheKey(userID), raw, s.ttl)
}

func (s *RedisBrandMembershipCache) InvalidateByUserID(ctx context.Context, userID uuid.UUID) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Del(ctx, brandMembershipCacheKey(userID))
}

func (s *RedisBrandMembershipCache) InvalidateByBrandID(ctx context.Context, brandID uuid.UUID, memberUserIDs []uuid.UUID) error {
	if s.cache == nil {
		return nil
	}
	if len(memberUserIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(memberUserIDs))
	for _, uid := range memberUserIDs {
		keys = append(keys, brandMembershipCacheKey(uid))
	}
	return s.cache.Del(ctx, keys...)
}

func brandMembershipCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("brand:members:%s", userID.String())
}

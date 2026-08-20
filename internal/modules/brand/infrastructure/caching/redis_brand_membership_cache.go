package caching

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/brand/application/interface/caching"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisBrandMembershipCache struct {
	redisClient *redis.Client
	ttl         time.Duration
	hitCount    atomic.Uint64
	missCount   atomic.Uint64
}

func NewRedisBrandMembershipCache(client *redis.Client, cfg *config.Config) caching.IBrandMembershipCache {
	return &RedisBrandMembershipCache{
		redisClient: client,
		ttl:         cfg.BrandMembership.TTL,
	}
}

func (s *RedisBrandMembershipCache) HitCount() uint64 {
	return s.hitCount.Load()
}

func (s *RedisBrandMembershipCache) MissCount() uint64 {
	return s.missCount.Load()
}

func (s *RedisBrandMembershipCache) Get(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]brandmemberrole.BrandMemberRole, error) {
	if s.redisClient == nil {
		s.missCount.Add(1)
		return nil, nil
	}
	key := brandMembershipCacheKey(userID)
	cached, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		s.missCount.Add(1)
		return nil, nil // miss or backend unavailable -> caller falls back to DB
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(cached), &raw); err != nil {
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
	if s.redisClient == nil {
		return nil
	}
	raw := make(map[string]string, len(memberships))
	for brandID, role := range memberships {
		raw[brandID.String()] = string(role)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return s.redisClient.Set(ctx, brandMembershipCacheKey(userID), data, s.ttl).Err()
}

func (s *RedisBrandMembershipCache) InvalidateByUserID(ctx context.Context, userID uuid.UUID) error {
	if s.redisClient == nil {
		return nil
	}
	return s.redisClient.Del(ctx, brandMembershipCacheKey(userID)).Err()
}

func (s *RedisBrandMembershipCache) InvalidateByBrandID(ctx context.Context, brandID uuid.UUID, memberUserIDs []uuid.UUID) error {
	if s.redisClient == nil {
		return nil
	}
	if len(memberUserIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(memberUserIDs))
	for _, uid := range memberUserIDs {
		keys = append(keys, brandMembershipCacheKey(uid))
	}
	return s.redisClient.Del(ctx, keys...).Err()
}

func brandMembershipCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("brand:members:%s", userID.String())
}

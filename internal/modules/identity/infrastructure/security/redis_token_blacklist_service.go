package security

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	shared_caching "smart-wardrobe-be/internal/shared/infrastructure/caching"

	"github.com/google/uuid"
)

type RedisTokenBlacklistService struct {
	cache shared_caching.ICacheService
}

const (
	blacklistPrefix     = "blacklist:token:"
	userBlacklistPrefix = "blacklist:user:"
)

func NewRedisTokenBlacklistService(cache shared_caching.ICacheService) security.ITokenBlacklistService {
	return &RedisTokenBlacklistService{
		cache: cache,
	}
}

func (s *RedisTokenBlacklistService) BlacklistToken(ctx context.Context, token string, expiry time.Duration) error {
	if expiry <= 0 {
		return nil
	}
	key := blacklistPrefix + token
	return s.cache.SetRaw(ctx, key, "revoked", expiry)
}

func (s *RedisTokenBlacklistService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := blacklistPrefix + token
	return s.cache.Exists(ctx, key)
}

func (s *RedisTokenBlacklistService) BlacklistTokenWithPrefix(ctx context.Context, token string, prefix string, expiry time.Duration) error {
	if expiry <= 0 {
		return nil
	}
	key := prefix + token
	return s.cache.SetRaw(ctx, key, "revoked", expiry)
}

func (s *RedisTokenBlacklistService) IsTokenBlacklistedWithPrefix(ctx context.Context, token string, prefix string) (bool, error) {
	key := prefix + token
	return s.cache.Exists(ctx, key)
}

func (s *RedisTokenBlacklistService) BlacklistUser(ctx context.Context, userID uuid.UUID, expiry time.Duration) error {
	if expiry <= 0 {
		return nil
	}
	key := userBlacklistPrefix + userID.String()
	return s.cache.SetRaw(ctx, key, "revoked", expiry)
}

func (s *RedisTokenBlacklistService) IsUserBlacklisted(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := userBlacklistPrefix + userID.String()
	return s.cache.Exists(ctx, key)
}

func (s *RedisTokenBlacklistService) UnblacklistUser(ctx context.Context, userID uuid.UUID) error {
	key := userBlacklistPrefix + userID.String()
	return s.cache.Del(ctx, key)
}

func (s *RedisTokenBlacklistService) CheckBlacklist(ctx context.Context, token string, userID uuid.UUID) (bool, bool, error) {
	tokenKey := blacklistPrefix + token
	userKey := userBlacklistPrefix + userID.String()

	tokenBlacklisted, err := s.cache.Exists(ctx, tokenKey)
	if err != nil {
		return false, false, err
	}

	userBlacklisted, err := s.cache.Exists(ctx, userKey)
	if err != nil {
		return false, false, err
	}

	return tokenBlacklisted, userBlacklisted, nil
}

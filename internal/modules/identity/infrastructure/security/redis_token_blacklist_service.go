package security

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/interface/security"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisTokenBlacklistService struct {
	redisClient *redis.Client
}

const (
	blacklistPrefix     = "blacklist:token:"
	userBlacklistPrefix = "blacklist:user:"
)

func NewRedisTokenBlacklistService(client *redis.Client) security.ITokenBlacklistService {
	return &RedisTokenBlacklistService{
		redisClient: client,
	}
}

func (s *RedisTokenBlacklistService) BlacklistToken(ctx context.Context, token string, expiry time.Duration) error {
	if expiry <= 0 {
		return nil
	}

	key := blacklistPrefix + token
	return s.redisClient.Set(ctx, key, "revoked", expiry).Err()
}

func (s *RedisTokenBlacklistService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := blacklistPrefix + token
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisTokenBlacklistService) BlacklistTokenWithPrefix(ctx context.Context, token string, prefix string, expiry time.Duration) error {
	if expiry <= 0 {
		return nil
	}
	key := prefix + token
	return s.redisClient.Set(ctx, key, "revoked", expiry).Err()
}

func (s *RedisTokenBlacklistService) IsTokenBlacklistedWithPrefix(ctx context.Context, token string, prefix string) (bool, error) {
	key := prefix + token
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisTokenBlacklistService) BlacklistUser(ctx context.Context, userID uuid.UUID, expiry time.Duration) error {
	if expiry <= 0 {
		return nil
	}

	key := userBlacklistPrefix + userID.String()
	return s.redisClient.Set(ctx, key, "revoked", expiry).Err()
}

func (s *RedisTokenBlacklistService) IsUserBlacklisted(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := userBlacklistPrefix + userID.String()
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisTokenBlacklistService) UnblacklistUser(ctx context.Context, userID uuid.UUID) error {
	key := userBlacklistPrefix + userID.String()
	return s.redisClient.Del(ctx, key).Err()
}

func (s *RedisTokenBlacklistService) CheckBlacklist(ctx context.Context, token string, userID uuid.UUID) (bool, bool, error) {
	tokenKey := blacklistPrefix + token
	userKey := userBlacklistPrefix + userID.String()

	results, err := s.redisClient.MGet(ctx, tokenKey, userKey).Result()
	if err != nil {
		return false, false, err
	}

	tokenBlacklisted := false
	userBlacklisted := false

	if len(results) > 0 && results[0] != nil {
		tokenBlacklisted = true
	}
	if len(results) > 1 && results[1] != nil {
		userBlacklisted = true
	}

	return tokenBlacklisted, userBlacklisted, nil
}

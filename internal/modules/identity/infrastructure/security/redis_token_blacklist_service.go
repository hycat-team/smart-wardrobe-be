package security

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/interface/security"

	"github.com/redis/go-redis/v9"
)

type RedisTokenBlacklistService struct {
	redisClient *redis.Client
}

const blacklistPrefix = "blacklist:token:"

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

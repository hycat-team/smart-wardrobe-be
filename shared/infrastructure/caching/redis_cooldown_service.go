package caching

import (
	"context"
	"time"

	shared_caching "smart-wardrobe-be/shared/application/interface/caching"

	"github.com/redis/go-redis/v9"
)

type RedisCooldownService struct {
	client *redis.Client
}

func NewRedisCooldownService(client *redis.Client) shared_caching.ICooldownService {
	return &RedisCooldownService{client: client}
}

func (c *RedisCooldownService) SetCooldown(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Set(ctx, key, "locked", ttl).Err()
}

func (c *RedisCooldownService) IsInCooldown(ctx context.Context, key string) (bool, error) {
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (c *RedisCooldownService) ClearCooldown(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

package caching

import (
	"context"

	shared_caching "smart-wardrobe-be/shared/application/interface/caching"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) shared_caching.IRateLimiter {
	return &RedisRateLimiter{client: client}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	script := redis.NewScript(`
		local current = redis.call("INCR", KEYS[1])
		if current == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		if current > tonumber(ARGV[2]) then
			return 0
		end
		return 1
	`)
	res, err := script.Run(ctx, r.client, []string{key}, windowSeconds, limit).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

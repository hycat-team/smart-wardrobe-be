package caching

import "context"

type IRateLimiter interface {
	Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error)
}

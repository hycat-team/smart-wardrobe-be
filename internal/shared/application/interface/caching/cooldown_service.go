package caching

import (
	"context"
	"time"
)

type ICooldownService interface {
	SetCooldown(ctx context.Context, key string, ttl time.Duration) error
	IsInCooldown(ctx context.Context, key string) (bool, error)
	ClearCooldown(ctx context.Context, key string) error
}

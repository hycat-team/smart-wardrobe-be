package security

import (
	"context"
	"time"
)

type ITokenBlacklistService interface {
	BlacklistToken(ctx context.Context, token string, expiry time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

package security

import (
	"context"
	"time"
)

type ITokenBlacklistService interface {
	BlacklistToken(ctx context.Context, token string, expiry time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	BlacklistTokenWithPrefix(ctx context.Context, token string, prefix string, expiry time.Duration) error
	IsTokenBlacklistedWithPrefix(ctx context.Context, token string, prefix string) (bool, error)
}

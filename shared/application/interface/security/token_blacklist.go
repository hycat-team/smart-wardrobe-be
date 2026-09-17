// Package security holds cross-service authentication contracts shared by
// every service (FR-022/FR-023): the session-revocation port used by the
// shared auth middleware and any use case that revokes sessions.
package security

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ITokenBlacklistService is the session-revocation port. Implementations MUST
// share one Redis so a revocation performed by any service is visible to all
// (FR-023).
type ITokenBlacklistService interface {
	BlacklistToken(ctx context.Context, token string, expiry time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	BlacklistTokenWithPrefix(ctx context.Context, token string, prefix string, expiry time.Duration) error
	IsTokenBlacklistedWithPrefix(ctx context.Context, token string, prefix string) (bool, error)
	BlacklistUser(ctx context.Context, userID uuid.UUID, expiry time.Duration) error
	IsUserBlacklisted(ctx context.Context, userID uuid.UUID) (bool, error)
	UnblacklistUser(ctx context.Context, userID uuid.UUID) error
	CheckBlacklist(ctx context.Context, token string, userID uuid.UUID) (tokenBlacklisted bool, userBlacklisted bool, err error)
}

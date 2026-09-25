package identity

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/application/vo"
)

// IOAuthStateStore persists one-time OAuth state for the web redirect flow.
// Consume removes the entry so a state can never be replayed.
type IOAuthStateStore interface {
	Save(ctx context.Context, state string, data vo.OAuthState, ttl time.Duration) error
	// Consume returns nil, nil when the state does not exist or already expired.
	Consume(ctx context.Context, state string) (*vo.OAuthState, error)
}

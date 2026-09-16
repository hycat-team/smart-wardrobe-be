package caching

import (
	"context"

	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

// IBrandMembershipCache caches the active brand memberships of a user so that
// brand RBAC checks (portal + dashboard) do not hit the database on every
// request. One user may belong to multiple brands, so the cache is keyed by
// userID and stores a map of brandID -> role for memberships that are active
// (member active + brand active).
type IBrandMembershipCache interface {
	// Get returns the cached memberships for a user, or nil when there is a
	// cache miss or the cache backend is unavailable (caller falls back to DB).
	Get(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]brandmemberrole.BrandMemberRole, error)
	// Set stores the memberships for a user with the configured TTL.
	Set(ctx context.Context, userID uuid.UUID, memberships map[uuid.UUID]brandmemberrole.BrandMemberRole) error
	// InvalidateByUserID removes the cached memberships of a single user.
	InvalidateByUserID(ctx context.Context, userID uuid.UUID) error
	// InvalidateByBrandID removes the cached memberships of every member of a brand.
	InvalidateByBrandID(ctx context.Context, brandID uuid.UUID, memberUserIDs []uuid.UUID) error
}

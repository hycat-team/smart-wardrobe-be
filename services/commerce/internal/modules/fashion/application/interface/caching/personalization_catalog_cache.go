package caching

import (
	"context"

	"github.com/google/uuid"
)

// IPersonalizationCatalogCache caches the list of system catalog fashion item
// ids used to build gap-filling suggestions (033 US2, plan D5).
//
// The list rarely changes and is read on every recommendation, so it is cached
// in Redis (key `personalization:catalog:v1`, TTL 5 minutes). A cache error
// MUST degrade to a miss: the caller then reads the Wardrobe contract directly.
type IPersonalizationCatalogCache interface {
	// Get returns the cached catalog fashion item ids, or nil when the key is
	// missing or the cache backend is unavailable.
	Get(ctx context.Context) ([]uuid.UUID, error)
	// Set stores the catalog fashion item ids with the configured TTL.
	Set(ctx context.Context, ids []uuid.UUID) error
}

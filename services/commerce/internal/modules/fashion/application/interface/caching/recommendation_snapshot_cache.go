package caching

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RecommendationSnapshot is the Redis-stored snapshot of one recommendation,
// used by the feedback endpoint to authenticate the owner and recover the
// fashion item ids without trusting the client (033 US2/US6, plan D8).
type RecommendationSnapshot struct {
	UserID         uuid.UUID   `json:"userId"`
	FashionItemIDs []uuid.UUID `json:"fashionItemIds"`
	Styles         []string    `json:"styles"`
	CreatedAt      time.Time   `json:"createdAt"`
}

// IRecommendationSnapshotCache stores recommendation snapshots in Redis under
// key `personalization:rec:{recommendationId}` with a 7-day TTL.
//
// A cache error MUST degrade to a miss: feedback then answers "snapshot not
// found" and never blocks the recommendation itself.
type IRecommendationSnapshotCache interface {
	// Set stores the snapshot for a recommendation id.
	Set(ctx context.Context, recommendationID uuid.UUID, snapshot *RecommendationSnapshot) error
	// Get returns the snapshot, or nil when it is missing, expired or the cache
	// backend is unavailable.
	Get(ctx context.Context, recommendationID uuid.UUID) (*RecommendationSnapshot, error)
	// Delete removes the snapshot for a recommendation id.
	Delete(ctx context.Context, recommendationID uuid.UUID) error
}

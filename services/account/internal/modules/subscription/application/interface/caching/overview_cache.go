package caching

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SubscriptionOverviewCacheEntry mirrors the fields of
// dto.UserSubscriptionContractDTO (plus a cache timestamp). Daily usage
// counters are intentionally NOT part of this entry (spec 019 FR-007).
type SubscriptionOverviewCacheEntry struct {
	PlanID             uuid.UUID  `json:"planId"`
	PlanName           string     `json:"planName"`
	PlanSlug           string     `json:"planSlug"`
	ExpiresAt          *time.Time `json:"expiresAt"`
	IsAutoRenewEnabled bool       `json:"isAutoRenewEnabled"`
	MaxWardrobeItems   int        `json:"maxWardrobeItems"`
	MaxOutfits         int        `json:"maxOutfits"`
	AiOutfitDailyQuota int        `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int        `json:"aiChatDailyQuota"`
	// PersonalizationEnabled is additive; `omitempty` keeps old payloads
	// byte-compatible and an absent key decodes as false (033 D12).
	PersonalizationEnabled bool      `json:"personalizationEnabled,omitempty"`
	CachedAt               time.Time `json:"cachedAt"`
}

// ISubscriptionOverviewCache caches the per-user subscription overview served
// through IUserSubscriptionContract so AI hot paths avoid repeated
// cross-module queries. Any cache failure must degrade to a miss (nil, nil) —
// callers fall back to the use case directly (spec 019 FR-008).
type ISubscriptionOverviewCache interface {
	Get(ctx context.Context, userID uuid.UUID) (*SubscriptionOverviewCacheEntry, error)
	Set(ctx context.Context, userID uuid.UUID, entry *SubscriptionOverviewCacheEntry) error
	Invalidate(ctx context.Context, userID uuid.UUID) error
}

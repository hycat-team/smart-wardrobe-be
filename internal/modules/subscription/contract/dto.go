package contract

import (
	"smart-wardrobe-be/internal/shared/domain/constants/plankind"
	"time"

	"github.com/google/uuid"
)

// UserSubscriptionDTO aggregates plan details and daily quota metrics
type UserSubscriptionDTO struct {
	PlanID               uuid.UUID          `json:"planID"`
	PlanName             string             `json:"planName"`
	PlanSlug             string             `json:"planSlug"`
	PlanKind             plankind.PlanKind  `json:"planKind"`
	TierRank             int                `json:"tierRank"`
	ExpiresAt            *time.Time         `json:"expiresAt"`
	IsAutoRenewEnabled   bool               `json:"isAutoRenewEnabled"`
	FallbackPlanCode     *string            `json:"fallbackPlanCode,omitempty"`
	FallbackTierRank     *int               `json:"fallbackTierRank,omitempty"`
	FallbackPlanKind     *plankind.PlanKind `json:"fallbackPlanKind,omitempty"`
	MaxWardrobeItems     int                `json:"maxWardrobeItems"`
	MaxOutfits           int                `json:"maxOutfits"`
	AiOutfitDailyQuota   int                `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota     int                `json:"aiChatDailyQuota"`
	OutfitRecommendCount int                `json:"outfitRecommendCount"`
	AiUsageCount         int                `json:"aiUsageCount"`
	LastResetDate        time.Time          `json:"lastResetDate"`
}

// UserSubscriptionOverviewDTO aggregates only subscription plan limits without daily quota usage metrics
type UserSubscriptionOverviewDTO struct {
	PlanID             uuid.UUID          `json:"planID"`
	PlanName           string             `json:"planName"`
	PlanSlug           string             `json:"planSlug"`
	PlanKind           plankind.PlanKind  `json:"planKind"`
	TierRank           int                `json:"tierRank"`
	ExpiresAt          *time.Time         `json:"expiresAt"`
	IsAutoRenewEnabled bool               `json:"isAutoRenewEnabled"`
	FallbackPlanCode   *string            `json:"fallbackPlanCode,omitempty"`
	FallbackTierRank   *int               `json:"fallbackTierRank,omitempty"`
	FallbackPlanKind   *plankind.PlanKind `json:"fallbackPlanKind,omitempty"`
	MaxWardrobeItems   int                `json:"maxWardrobeItems"`
	MaxOutfits         int                `json:"maxOutfits"`
	AiOutfitDailyQuota int                `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int                `json:"aiChatDailyQuota"`
}

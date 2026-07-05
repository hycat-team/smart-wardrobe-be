package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"

	"github.com/google/uuid"
)

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

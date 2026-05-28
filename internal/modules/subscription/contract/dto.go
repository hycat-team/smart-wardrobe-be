package contract

import (
	"time"

	"github.com/google/uuid"
)

// UserSubscriptionDTO aggregates plan details and daily quota metrics
type UserSubscriptionDTO struct {
	PlanID               uuid.UUID
	PlanName             string
	PlanSlug             string
	ExpiresAt            *time.Time
	IsAutoRenewEnabled   bool
	MaxWardrobeItems     int
	MaxOutfits           int
	AiOutfitDailyQuota   int
	AiChatDailyQuota     int
	OutfitRecommendCount int
	AiUsageCount         int
	LastResetDate        time.Time
}

// UserSubscriptionOverviewDTO aggregates only subscription plan limits without daily quota usage metrics
type UserSubscriptionOverviewDTO struct {
	PlanID             uuid.UUID
	PlanName           string
	PlanSlug           string
	ExpiresAt          *time.Time
	IsAutoRenewEnabled bool
	MaxWardrobeItems   int
	MaxOutfits         int
	AiOutfitDailyQuota int
	AiChatDailyQuota   int
}

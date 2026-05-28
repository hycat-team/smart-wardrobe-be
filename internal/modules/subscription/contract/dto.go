package contract

import (
	"time"

	"github.com/google/uuid"
)

// UserSubscriptionDTO aggregates plan details and daily quota metrics
type UserSubscriptionDTO struct {
	PlanID               uuid.UUID
	PlanName             string
	ExpiresAt            *time.Time
	MaxWardrobeItems     int
	MaxOutfits           int
	AiOutfitDailyQuota   int
	AiChatDailyQuota     int
	OutfitRecommendCount int
	AiUsageCount         int
	LastResetDate        time.Time
}

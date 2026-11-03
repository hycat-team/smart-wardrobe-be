package dto

import (
	"time"

	"github.com/google/uuid"
)

type UserSubscriptionContractDTO struct {
	PlanID             uuid.UUID  `json:"planId"`
	PlanName           string     `json:"planName"`
	PlanSlug           string     `json:"planSlug"`
	ExpiresAt          *time.Time `json:"expiresAt"`
	IsAutoRenewEnabled bool       `json:"isAutoRenewEnabled"`
	MaxWardrobeItems   int        `json:"maxWardrobeItems"`
	MaxOutfits         int        `json:"maxOutfits"`
	AiOutfitDailyQuota int        `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int        `json:"aiChatDailyQuota"`
	// PersonalizationEnabled is the plan entitlement for personalized styling
	// (033 D2). Additive field: old consumers read the zero value (false).
	PersonalizationEnabled bool `json:"personalizationEnabled"`
}

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
}

type UserSubscriptionOverviewDTO = UserSubscriptionContractDTO

type UserQuotaContractDTO struct {
	UserID               uuid.UUID `json:"user_id"`
	AiOutfitDailyQuota   int       `json:"ai_outfit_daily_quota"`
	OutfitRecommendCount int       `json:"outfit_recommend_count"`
	AiChatDailyQuota     int       `json:"ai_chat_daily_quota"`
	AiUsageCount         int       `json:"ai_usage_count"`
}

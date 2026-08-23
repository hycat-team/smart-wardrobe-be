package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type UserSubscriptionDTO struct {
	PlanID               uuid.UUID  `json:"planID"`
	PlanName             string     `json:"planName"`
	PlanSlug             string     `json:"planSlug"`
	ExpiresAt            *time.Time `json:"expiresAt"`
	IsAutoRenewEnabled   bool       `json:"isAutoRenewEnabled"`
	AiOutfitDailyQuota   int        `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota     int        `json:"aiChatDailyQuota"`
	OutfitRecommendCount int        `json:"outfitRecommendCount"`
	AiUsageCount         int        `json:"aiUsageCount"`
	LastResetDate        time.Time  `json:"lastResetDate"`
}

type UserSubscriptionOverviewDTO struct {
	PlanID             uuid.UUID  `json:"planID"`
	PlanName           string     `json:"planName"`
	PlanSlug           string     `json:"planSlug"`
	ExpiresAt          *time.Time `json:"expiresAt"`
	IsAutoRenewEnabled bool       `json:"isAutoRenewEnabled"`
	MaxWardrobeItems   int        `json:"maxWardrobeItems"`
	MaxOutfits         int        `json:"maxOutfits"`
	AiOutfitDailyQuota int        `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int        `json:"aiChatDailyQuota"`
}

type AdminUserSubscriptionItemDTO struct {
	UserID             uuid.UUID         `json:"userId"`
	PlanID             uuid.UUID         `json:"planId"`
	PlanName           string            `json:"planName"`
	PlanSlug           string            `json:"planSlug"`
	PlanKind           plankind.PlanKind `json:"planKind"`
	TierRank           int               `json:"tierRank"`
	StartedAt          time.Time         `json:"startedAt"`
	ExpiresAt          *time.Time        `json:"expiresAt"`
	IsAutoRenewEnabled bool              `json:"isAutoRenewEnabled"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type AdminUserSubscriptionListRes = shared_dto.PaginationResult[*AdminUserSubscriptionItemDTO]

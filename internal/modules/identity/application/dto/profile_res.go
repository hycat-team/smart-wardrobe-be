package dto

import (
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"time"

	"github.com/google/uuid"
)

type UserRes struct {
	ID           uuid.UUID           `json:"id"`
	Username     string              `json:"username"`
	Email        string              `json:"email"`
	RoleSlug     string              `json:"roleSlug"`
	FirstName    string              `json:"firstName"`
	LastName     string              `json:"lastName,omitempty"`
	Address      string              `json:"address,omitempty"`
	Gender       gender.Gender       `json:"gender"`
	Status       int                 `json:"status"`
	CreatedAt    time.Time           `json:"createdAt"`
	Subscription UserSubscriptionRes `json:"subscription"`
	Quota        UserQuotaRes        `json:"quota"`
	BodyProfile  *UserBodyProfileRes `json:"bodyProfile,omitempty"`
}

type UserSubscriptionRes struct {
	PlanID             uuid.UUID  `json:"planId"`
	PlanName           string     `json:"planName"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	MaxWardrobeItems   int        `json:"maxWardrobeItems"`
	MaxOutfits         int        `json:"maxOutfits"`
	AiOutfitDailyQuota int        `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int        `json:"aiChatDailyQuota"`
}

type UserQuotaRes struct {
	OutfitRecommendCount int       `json:"outfitRecommendCount"`
	AiUsageCount         int       `json:"aiUsageCount"`
	LastResetDate        time.Time `json:"lastResetDate"`
}

type UserBodyProfileRes struct {
	Height             float64 `json:"height"`
	Weight             float64 `json:"weight"`
	BodyType           string  `json:"bodyType"`
	FitPreference      string  `json:"fitPreference"`
	SkinTone           string  `json:"skinTone"`
	EstimatedBodyShape string  `json:"estimatedBodyShape"`
	RecommendedSize    string  `json:"recommendedSize"`
	StylingNotes       string  `json:"stylingNotes"`
}

package dto

import (
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"time"

	"github.com/google/uuid"
)

type UserRes struct {
	ID             uuid.UUID             `json:"id"`
	Username       string                `json:"username"`
	Email          string                `json:"email"`
	RoleSlug       roleslug.RoleSlug     `json:"roleSlug"`
	FirstName      string                `json:"firstName"`
	LastName       string                `json:"lastName,omitempty"`
	DateOfBirth    *string               `json:"dateOfBirth,omitempty"`
	Address        string                `json:"address,omitempty"`
	Gender         gender.Gender         `json:"gender"`
	Status         userstatus.UserStatus `json:"status"`
	CreatedAt      time.Time             `json:"createdAt"`
	Subscription   UserSubscriptionRes   `json:"subscription"`
	BodyProfile    *UserBodyProfileRes   `json:"bodyProfile,omitempty"`
	AvatarUrl      *string               `json:"avatarUrl,omitempty"`
	AvatarPublicID *string               `json:"avatarPublicId,omitempty"`
}

type UpdateAvatarReq struct {
	AvatarUrl      string `json:"avatarUrl" binding:"required" label:"đường dẫn ảnh đại diện"`
	AvatarPublicID string `json:"avatarPublicId" binding:"required" label:"mã ảnh đại diện"`
}

type UserSubscriptionRes struct {
	PlanID             uuid.UUID  `json:"planId"`
	PlanName           string     `json:"planName"`
	PlanSlug           string     `json:"planSlug"`
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

type UserStyleProfileRes struct {
	UserID          uuid.UUID                 `json:"userId"`
	TasteEmbedding  entities.Vector           `json:"tasteEmbedding,omitempty"`
	PreferredColors *entities.PreferredColors `json:"preferredColors,omitempty"`
}

type UserBodyProfileRes struct {
	HeightCM       float64                  `json:"heightCm"`
	WeightKG       float64                  `json:"weightKg"`
	BodyShape      string                   `json:"bodyShape"`
	Measurements   *UserBodyMeasurementsRes `json:"measurements,omitempty"`
	InferredByAI   *UserInferredBodyRes     `json:"inferredByAi,omitempty"`
	VerifiedByUser bool                     `json:"verifiedByUser"`
	LastUpdatedAt  *time.Time               `json:"lastUpdatedAt,omitempty"`
}

type UserBodyMeasurementsRes struct {
	ChestCM float64 `json:"chestCm,omitempty"`
	WaistCM float64 `json:"waistCm,omitempty"`
	HipCM   float64 `json:"hipCm,omitempty"`
}

type UserInferredBodyRes struct {
	BodyShape       string   `json:"bodyShape"`
	ConfidenceScore *float64 `json:"confidenceScore,omitempty"`
}

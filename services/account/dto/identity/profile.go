package dto

import (
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"
	"time"

	"github.com/google/uuid"
)

type UserRes struct {
	ID           uuid.UUID             `json:"id"`
	Username     string                `json:"username"`
	Email        string                `json:"email"`
	RoleSlug     roleslug.RoleSlug     `json:"roleSlug"`
	FirstName    string                `json:"firstName"`
	LastName     string                `json:"lastName,omitempty"`
	DateOfBirth  *string               `json:"dateOfBirth,omitempty"`
	Address      string                `json:"address,omitempty"`
	Gender       gender.Gender         `json:"gender"`
	Status       userstatus.UserStatus `json:"status"`
	CreatedAt    time.Time             `json:"createdAt"`
	Subscription UserSubscriptionRes   `json:"subscription"`
	BodyProfile  *UserBodyProfileRes   `json:"bodyProfile,omitempty"`
	AvatarUrl    *string               `json:"avatarUrl,omitempty"`
}

type UpdateAvatarReq struct {
	AvatarUrl      string `json:"avatarUrl" binding:"required" label:"đường dẫn ảnh đại diện"`
	AvatarPublicID string `json:"avatarPublicId" binding:"required" label:"mã ảnh đại diện"`
}

// UserBriefRes is the HTTP API response for a minimal user summary.
// Field set mirrors UserBriefContractDTO (the inter-module payload) but the two
// types evolve independently per the DTO Convention (design-pattern.md).
type UserBriefRes struct {
	ID        uuid.UUID             `json:"id"`
	Username  string                `json:"username"`
	Email     string                `json:"email,omitempty"`
	FirstName string                `json:"firstName"`
	LastName  string                `json:"lastName,omitempty"`
	Gender    gender.Gender         `json:"gender"`
	Status    userstatus.UserStatus `json:"status"`
	AvatarUrl *string               `json:"avatarUrl,omitempty"`
}

// UserSubscriptionRes is a projection of
// subscription_dto.UserSubscriptionContractDTO embedded in UserRes (drops
// IsAutoRenewEnabled, ExpiresAt omitempty) — keep JSON fields in sync with the source.
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
type UpdateProfileReq struct {
	FirstName   string         `json:"firstName" binding:"required" label:"tên"`
	LastName    *string        `json:"lastName" binding:"omitempty" label:"họ"`
	DateOfBirth string         `json:"dateOfBirth" binding:"omitempty,datetime=2006-01-02" label:"ngày sinh"`
	Gender      *gender.Gender `json:"gender" binding:"omitempty,oneof=0 1 2 3" label:"giới tính"`
	Address     *string        `json:"address" binding:"omitempty" label:"địa chỉ"`
}

type UpdateBodyProfileReq struct {
	HeightCM       float64                    `json:"heightCm" binding:"required,gte=0" label:"chiều cao"`
	WeightKG       float64                    `json:"weightKg" binding:"required,gte=0" label:"cân nặng"`
	BodyShape      string                     `json:"bodyShape" binding:"required" label:"dáng người"`
	Measurements   *UpdateBodyMeasurementsReq `json:"measurements,omitempty"`
	InferredByAI   *InferredBodyProfileReq    `json:"inferredByAi,omitempty"`
	VerifiedByUser bool                       `json:"verifiedByUser"`
}

type UpdateBodyMeasurementsReq struct {
	ChestCM float64 `json:"chestCm" binding:"omitempty,gte=0" label:"vòng ngực"`
	WaistCM float64 `json:"waistCm" binding:"omitempty,gte=0" label:"vòng eo"`
	HipCM   float64 `json:"hipCm" binding:"omitempty,gte=0" label:"vòng hông"`
}

type InferredBodyProfileReq struct {
	BodyShape       string   `json:"bodyShape" binding:"required" label:"dáng người AI suy luận"`
	ConfidenceScore *float64 `json:"confidenceScore,omitempty" binding:"omitempty,gte=0,lte=1" label:"độ tự tin AI"`
}

type UpdateUserStatusReq struct {
	Status userstatus.UserStatus `json:"status" binding:"oneof=0 1" label:"trạng thái tài khoản"`
}

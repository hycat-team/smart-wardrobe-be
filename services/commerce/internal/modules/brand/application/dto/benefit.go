package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type CreateBrandBenefitReq struct {
	Name           string     `json:"name" binding:"required,max=255" label:"tên quyền lợi"`
	Description    *string    `json:"description" binding:"omitempty" label:"mô tả"`
	BenefitType    string     `json:"benefitType" binding:"required" label:"loại quyền lợi"`
	UnlockType     string     `json:"unlockType" binding:"required" label:"loại mở khóa"`
	RequiredPoints *int       `json:"requiredPoints" binding:"omitempty,min=0" label:"điểm yêu cầu"`
	RequiredTierID *uuid.UUID `json:"requiredTierId" binding:"omitempty" label:"mã hạng yêu cầu"`
	FeatureCode    *string    `json:"featureCode" binding:"omitempty,max=100" label:"mã tính năng"`
	FeatureConfig  any        `json:"featureConfig" binding:"omitempty" label:"cấu hình tính năng"`
}

type UpdateBenefitStatusReq struct {
	Status string `json:"status" binding:"required" label:"trang thai"`
}

type BrandBenefitRes struct {
	ID               uuid.UUID  `json:"id"`
	BrandID          uuid.UUID  `json:"brandId"`
	Name             string     `json:"name"`
	Description      *string    `json:"description"`
	BenefitType      string     `json:"benefitType"`
	UnlockType       string     `json:"unlockType"`
	RequiredPoints   *int       `json:"requiredPoints"`
	RequiredTierID   *uuid.UUID `json:"requiredTierId"`
	RequiredTierName *string    `json:"requiredTierName,omitempty"`
	RequiredTierRank *int       `json:"requiredTierRank,omitempty"`
	FeatureCode      *string    `json:"featureCode"`
	FeatureConfig    any        `json:"featureConfig"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type BrandBenefitListRes = shared_dto.PaginationResult[*BrandBenefitRes]

type BenefitBriefRes struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Description      *string   `json:"description"`
	BenefitType      string    `json:"benefitType"`
	UnlockType       string    `json:"unlockType"`
	RequiredPoints   *int      `json:"requiredPoints"`
	RequiredTierName *string   `json:"requiredTierName,omitempty"`
	FeatureCode      *string   `json:"featureCode"`
	FeatureConfig    any       `json:"featureConfig"`
	Status           string    `json:"status"`
}

type BenefitRedemptionRes struct {
	ID              uuid.UUID        `json:"id"`
	BenefitID       uuid.UUID        `json:"benefitId"`
	Benefit         *BenefitBriefRes `json:"benefit,omitempty"`
	Brand           *BrandBriefRes   `json:"brand,omitempty"`
	BrandCustomerID uuid.UUID        `json:"brandCustomerId"`
	UserID          *uuid.UUID       `json:"userId"`
	PointsSpent     int              `json:"pointsSpent"`
	Status          string           `json:"status"`
	RedeemedAt      time.Time        `json:"redeemedAt"`
	UsedAt          *time.Time       `json:"usedAt"`
	ExpiresAt       *time.Time       `json:"expiresAt"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type GetBenefitRedemptionsQueryReq struct {
	shared_dto.PaginationQuery
	BrandID *string `form:"brandId" binding:"omitempty,uuid" label:"mã thương hiệu"`
}

type BenefitRedemptionListRes = shared_dto.PaginationResult[*BenefitRedemptionRes]

type MyBenefitRedemptionRes struct {
	ID          uuid.UUID  `json:"id"`
	Status      string     `json:"status"`
	PointsSpent int        `json:"pointsSpent"`
	RedeemedAt  time.Time  `json:"redeemedAt"`
	UsedAt      *time.Time `json:"usedAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}

// MyBrandBenefitRes is a brand benefit together with the caller's unlock and
// redemption state.
type MyBrandBenefitRes struct {
	BrandBenefitRes
	Unlocked   bool                    `json:"unlocked"`
	Redemption *MyBenefitRedemptionRes `json:"redemption,omitempty"`
}

// MyBrandBenefitListRes is the caller's benefit view for one brand.
type MyBrandBenefitListRes struct {
	CurrentTier   *LoyaltyTierRes               `json:"currentTier"`
	CurrentPoints int                           `json:"currentPoints"`
	Items         []*MyBrandBenefitRes          `json:"items"`
	Metadata      shared_dto.PaginationMetadata `json:"metadata"`
}

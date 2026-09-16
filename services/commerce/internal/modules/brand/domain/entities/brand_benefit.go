package entities

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitunlocktype"

	"github.com/google/uuid"
)

type BrandBenefit struct {
	AuditableEntity
	BrandID        uuid.UUID                              `gorm:"type:uuid;not null"`
	Brand          *Brand                                 `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	Name           string                                 `gorm:"type:varchar(255);not null"`
	Description    *string                                `gorm:"type:text"`
	BenefitType    benefittype.BenefitType                `gorm:"type:varchar(50);not null"`
	UnlockType     benefitunlocktype.BenefitUnlockType    `gorm:"type:varchar(50);not null"`
	RequiredPoints *int                                   `gorm:"type:int"`
	RequiredTierID *uuid.UUID                             `gorm:"type:uuid"`
	RequiredTier   *LoyaltyTier                           `gorm:"foreignKey:RequiredTierID;constraint:OnDelete:SET NULL"`
	FeatureCode    *benefitfeaturecode.BenefitFeatureCode `gorm:"type:varchar(100)"`
	FeatureConfig  JSONDocument                           `gorm:"type:jsonb"`
	Status         benefitstatus.BenefitStatus            `gorm:"type:varchar(50);not null;default:ACTIVE"`
}

type BenefitRedemption struct {
	AuditableEntity
	BenefitID       uuid.UUID                                       `gorm:"type:uuid;not null"`
	Benefit         *BrandBenefit                                   `gorm:"foreignKey:BenefitID;constraint:OnDelete:CASCADE"`
	BrandID         uuid.UUID                                       `gorm:"type:uuid;not null"`
	Brand           *Brand                                          `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	BrandCustomerID uuid.UUID                                       `gorm:"type:uuid;not null"`
	BrandCustomer   *BrandCustomer                                  `gorm:"foreignKey:BrandCustomerID;constraint:OnDelete:CASCADE"`
	UserID          *uuid.UUID                                      `gorm:"type:uuid"`
	PointsSpent     int                                             `gorm:"type:int;not null;default:0"`
	Status          benefitredemptionstatus.BenefitRedemptionStatus `gorm:"type:varchar(50);not null;default:REDEEMED"`
	RedeemedAt      time.Time                                       `gorm:"type:timestamp with time zone;not null;default:now()"`
	UsedAt          *time.Time                                      `gorm:"type:timestamp with time zone"`
	ExpiresAt       *time.Time                                      `gorm:"type:timestamp with time zone"`
}

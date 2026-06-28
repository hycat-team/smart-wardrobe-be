package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltyroundingmode"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltytransactiontype"

	"github.com/google/uuid"
)

type LoyaltyProgram struct {
	AuditableEntity
	BrandID         uuid.UUID                               `gorm:"type:uuid;not null"`
	Brand           *Brand                                  `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	Name            string                                  `gorm:"type:varchar(255);not null"`
	AmountPerPoint  float64                                 `gorm:"type:decimal(12,2);not null"`
	PointExpiryDays *int                                    `gorm:"type:int"`
	RoundingMode    loyaltyroundingmode.LoyaltyRoundingMode `gorm:"type:varchar(50);not null;default:FLOOR"`
	IsActive        bool                                    `gorm:"type:boolean;not null;default:true"`
}

type LoyaltyTier struct {
	AuditableEntity
	BrandID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_loyalty_tiers_brand_rank;uniqueIndex:uq_loyalty_tiers_brand_name"`
	Brand         *Brand    `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	Name          string    `gorm:"type:varchar(255);not null;uniqueIndex:uq_loyalty_tiers_brand_name"`
	Rank          int       `gorm:"type:int;not null;uniqueIndex:uq_loyalty_tiers_brand_rank"`
	MinTotalSpend float64   `gorm:"type:decimal(12,2);not null;default:0"`
	Description   *string   `gorm:"type:text"`
}

type LoyaltyAccount struct {
	AuditableEntity
	BrandID         uuid.UUID      `gorm:"type:uuid;not null"`
	Brand           *Brand         `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	BrandCustomerID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	BrandCustomer   *BrandCustomer `gorm:"foreignKey:BrandCustomerID;constraint:OnDelete:CASCADE"`
	UserID          *uuid.UUID     `gorm:"type:uuid"`
	User            *User          `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
	CurrentPoints   int            `gorm:"type:int;not null;default:0"`
	LifetimePoints  int            `gorm:"type:int;not null;default:0"`
	TotalSpend      float64        `gorm:"type:decimal(12,2);not null;default:0"`
	CurrentTierID   *uuid.UUID     `gorm:"type:uuid"`
	CurrentTier     *LoyaltyTier   `gorm:"foreignKey:CurrentTierID;constraint:OnDelete:SET NULL"`
}

type LoyaltyPointTransaction struct {
	ID               uuid.UUID                                     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	LoyaltyAccountID uuid.UUID                                     `gorm:"type:uuid;not null"`
	LoyaltyAccount   *LoyaltyAccount                               `gorm:"foreignKey:LoyaltyAccountID;constraint:OnDelete:CASCADE"`
	BrandID          uuid.UUID                                     `gorm:"type:uuid;not null"`
	Brand            *Brand                                        `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	BrandCustomerID  uuid.UUID                                     `gorm:"type:uuid;not null"`
	BrandCustomer    *BrandCustomer                                `gorm:"foreignKey:BrandCustomerID;constraint:OnDelete:CASCADE"`
	UserID           *uuid.UUID                                    `gorm:"type:uuid"`
	User             *User                                         `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
	PointsDelta      int                                           `gorm:"type:int;not null"`
	BalanceAfter     int                                           `gorm:"type:int;not null"`
	TransactionType  loyaltytransactiontype.LoyaltyTransactionType `gorm:"type:varchar(50);not null"`
	Reason           *string                                       `gorm:"type:varchar(255)"`
	SpendAmount      *float64                                      `gorm:"type:decimal(12,2)"`
	ReferenceType    *string                                       `gorm:"type:varchar(100)"`
	ReferenceID      *uuid.UUID                                    `gorm:"type:uuid"`
	ExpiresAt        *time.Time                                    `gorm:"type:timestamp with time zone"`
	IdempotencyKey   *string                                       `gorm:"type:varchar(100)"`
	CreatedByUserID  *uuid.UUID                                    `gorm:"type:uuid"`
	CreatedByUser    *User                                         `gorm:"foreignKey:CreatedByUserID;constraint:OnDelete:SET NULL"`
	CreatedAt        time.Time                                     `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type LoyaltyPointLot struct {
	AuditableEntity
	LoyaltyAccountID  uuid.UUID                                   `gorm:"type:uuid;not null"`
	LoyaltyAccount    *LoyaltyAccount                             `gorm:"foreignKey:LoyaltyAccountID;constraint:OnDelete:CASCADE"`
	BrandID           uuid.UUID                                   `gorm:"type:uuid;not null"`
	Brand             *Brand                                      `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	BrandCustomerID   uuid.UUID                                   `gorm:"type:uuid;not null"`
	BrandCustomer     *BrandCustomer                              `gorm:"foreignKey:BrandCustomerID;constraint:OnDelete:CASCADE"`
	UserID            *uuid.UUID                                  `gorm:"type:uuid"`
	User              *User                                       `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
	EarnTransactionID uuid.UUID                                   `gorm:"type:uuid;not null;uniqueIndex"`
	EarnTransaction   *LoyaltyPointTransaction                    `gorm:"foreignKey:EarnTransactionID;constraint:OnDelete:RESTRICT"`
	EarnedPoints      int                                         `gorm:"type:int;not null"`
	RemainingPoints   int                                         `gorm:"type:int;not null"`
	ExpiresAt         *time.Time                                  `gorm:"type:timestamp with time zone"`
	Status            loyaltypointlotstatus.LoyaltyPointLotStatus `gorm:"type:varchar(50);not null;default:ACTIVE"`
}

type BrandCustomerClaim struct {
	BaseEntity
	BrandCustomerID uuid.UUID      `gorm:"type:uuid;not null"`
	BrandCustomer   *BrandCustomer `gorm:"foreignKey:BrandCustomerID;constraint:OnDelete:CASCADE"`
	ClaimTokenHash  string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt       time.Time      `gorm:"type:timestamp with time zone;not null"`
	ConsumedAt      *time.Time     `gorm:"type:timestamp with time zone"`
}

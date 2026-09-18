package entities

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserSubscription struct {
	UserID                   uuid.UUID          `gorm:"type:uuid;primaryKey"`
	SubscriptionPlanID       uuid.UUID          `gorm:"type:uuid;not null"`
	SubscriptionPlan         *SubscriptionPlan  `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:RESTRICT"`
	CurrentPlanCode          string             `gorm:"type:varchar(100);not null"`
	CurrentTierRank          int                `gorm:"type:int;not null"`
	CurrentPlanKind          plankind.PlanKind  `gorm:"type:smallint;not null"`
	CurrentBenefitSnapshot   JSONDocument       `gorm:"type:jsonb;not null;default:'{}'"`
	StartedAt                time.Time          `gorm:"type:timestamp with time zone;not null;default:now()"`
	ExpiresAt                *time.Time         `gorm:"type:timestamp with time zone"`
	FallbackPlanID           *uuid.UUID         `gorm:"type:uuid"`
	FallbackPlanCode         *string            `gorm:"type:varchar(100)"`
	FallbackTierRank         *int               `gorm:"type:int"`
	FallbackPlanKind         *plankind.PlanKind `gorm:"type:smallint"`
	FallbackBenefitSnapshot  JSONDocument       `gorm:"type:jsonb"`
	LastDepositTransactionID *uuid.UUID         `gorm:"type:uuid"`
	IsAutoRenewEnabled       bool               `gorm:"type:boolean;not null;default:false"`
	Version                  int64              `gorm:"type:bigint;not null;default:0"`
	CreatedAt                time.Time          `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt                time.Time          `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type UserDailyQuota struct {
	UserID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	OutfitRecommendCount int       `gorm:"type:int;not null;default:0"`
	AiUsageCount         int       `gorm:"type:int;not null;default:0"`
	LastResetDate        time.Time `gorm:"type:date;not null"`
	CreatedAt            time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt            time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type UserWallet struct {
	UserID    uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Balance   decimal.Decimal   `gorm:"type:numeric(12,2);not null;default:0.00"`
	Currency  currency.Currency `gorm:"type:varchar(10);not null;default:'VND'"`
	CreatedAt time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
}

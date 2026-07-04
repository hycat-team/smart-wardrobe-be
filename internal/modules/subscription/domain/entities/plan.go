package entities

import (
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"

	
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SubscriptionPlan struct {
	AuditableEntity
	Slug               string          `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name               string          `gorm:"type:varchar(100);not null"`
	Price              decimal.Decimal `gorm:"type:numeric(12,2);not null;default:0.00"`
	MaxWardrobeItems   int             `gorm:"type:int;not null"`
	MaxOutfits         int             `gorm:"type:int;not null"`
	AiOutfitDailyQuota int             `gorm:"type:int;not null"`
	AiChatDailyQuota   int             `gorm:"type:int;not null"`
	DurationDays       *int            `gorm:"type:int"`
	IsActive           bool            `gorm:"type:boolean;not null;default:true"`
	PlanKind           plankind.PlanKind `gorm:"type:smallint;not null;default:0"`
	TierRank           int             `gorm:"type:int;not null;default:0"`
	PricingVersion     int64           `gorm:"type:bigint;not null;default:1"`
	AICostPolicyID     uuid.UUID       `gorm:"type:uuid;not null"`
}

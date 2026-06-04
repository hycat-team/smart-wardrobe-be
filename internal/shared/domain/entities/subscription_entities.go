package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"

	"github.com/google/uuid"
)

type SubscriptionPlan struct {
	AuditableEntity
	Slug               string  `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name               string  `gorm:"type:varchar(100);not null"`
	Price              float64 `gorm:"type:numeric(12,2);not null;default:0.00"`
	MaxWardrobeItems   int     `gorm:"type:int;not null"`
	MaxOutfits         int     `gorm:"type:int;not null"`
	AiOutfitDailyQuota int     `gorm:"type:int;not null"`
	AiChatDailyQuota   int     `gorm:"type:int;not null"`
	DurationDays       *int    `gorm:"type:int"`
	IsActive           bool    `gorm:"type:boolean;not null;default:true"`
}

type UserSubscription struct {
	UserID             uuid.UUID         `gorm:"type:uuid;primaryKey"`
	SubscriptionPlanID uuid.UUID         `gorm:"type:uuid;not null"`
	SubscriptionPlan   *SubscriptionPlan `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:RESTRICT"`
	ExpiresAt          *time.Time        `gorm:"type:timestamp with time zone"`
	IsActive           bool              `gorm:"type:boolean;not null;default:true"`
	IsAutoRenewEnabled bool              `gorm:"type:boolean;not null;default:false"`
	CreatedAt          time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt          time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
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
	User      *User             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Balance   float64           `gorm:"type:numeric(12,2);not null;default:0.00"`
	Currency  currency.Currency `gorm:"type:varchar(10);not null;default:'VND'"`
	CreatedAt time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type DepositTransaction struct {
	AuditableEntity
	UserID             uuid.UUID                                     `gorm:"type:uuid;not null"`
	User               *User                                         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount             float64                                       `gorm:"type:numeric(12,2);not null"`
	Currency           currency.Currency                             `gorm:"type:varchar(10);not null;default:'VND'"`
	Status             depositstatus.DepositStatus                   `gorm:"type:smallint;not null;default:0"` // 0: PENDING, 1: SUCCESS, 2: FAILED
	TransactionType    deposittransactiontype.DepositTransactionType `gorm:"type:varchar(50);not null"`        // DIRECT_PURCHASE, WALLET_TOPUP
	SubscriptionPlanID *uuid.UUID                                    `gorm:"type:uuid"`
	SubscriptionPlan   *SubscriptionPlan                             `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:SET NULL"`
	OrderCode          int64                                         `gorm:"type:bigint;autoIncrement;uniqueIndex;not null"`
	GatewayReference   *string                                       `gorm:"type:varchar(255);uniqueIndex"` // PayOS payment transaction reference
	GatewayDetails     *string                                       `gorm:"type:text"`                     // Raw JSON string payload from gateway
	PaymentUrl         *string                                       `gorm:"type:varchar(500)"`
}

type WalletStatement struct {
	AuditableEntity
	UserID          uuid.UUID                               `gorm:"type:uuid;not null"`
	User            *User                                   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount          float64                                 `gorm:"type:numeric(12,2);not null"`
	TransactionType walletstatementtype.WalletStatementType `gorm:"type:varchar(50);not null"` // TOPUP, SUBSCRIPTION_PURCHASE, SUBSCRIPTION_RENEWAL
	PreviousBalance float64                                 `gorm:"type:numeric(12,2);not null"`
	NewBalance      float64                                 `gorm:"type:numeric(12,2);not null"`
	ReferenceID     *uuid.UUID                              `gorm:"type:uuid"`
	Description     string                                  `gorm:"type:varchar(255);not null"`
}

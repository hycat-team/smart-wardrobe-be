package entities

import (
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefitresolution"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DepositTransaction struct {
	AuditableEntity
	UserID                      uuid.UUID                                     `gorm:"type:uuid;not null"`
	Amount                      decimal.Decimal                               `gorm:"type:numeric(12,2);not null"`
	Currency                    currency.Currency                             `gorm:"type:varchar(10);not null;default:'VND'"`
	Status                      depositstatus.DepositStatus                   `gorm:"type:smallint;not null;default:3"`
	TransactionType             deposittransactiontype.DepositTransactionType `gorm:"type:varchar(50);not null"`
	SubscriptionPlanID          *uuid.UUID                                    `gorm:"type:uuid"`
	SubscriptionPlan            *SubscriptionPlan                             `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:SET NULL"`
	OrderCode                   int64                                         `gorm:"type:bigint;autoIncrement;uniqueIndex;not null"`
	Provider                    string                                        `gorm:"type:varchar(50);not null;default:'PAYOS'"`
	PaymentLinkID               *string                                       `gorm:"type:varchar(255)"`
	ProviderStatus              *string                                       `gorm:"type:varchar(50)"`
	SuccessfulProviderReference *string                                       `gorm:"type:varchar(255)"`
	GatewayDetails              *string                                       `gorm:"type:text"`
	PaymentUrl                  *string                                       `gorm:"type:varchar(500)"`
	ExpiresAt                   *time.Time                                    `gorm:"type:timestamp with time zone"`
	NextReconciliationAt        *time.Time                                    `gorm:"type:timestamp with time zone"`
	ReconciliationAttempts      int                                           `gorm:"type:int;not null;default:0"`
	ProcessingToken             *uuid.UUID                                    `gorm:"type:uuid"`
	ProcessingLeaseUntil        *time.Time                                    `gorm:"type:timestamp with time zone"`
	LastProviderErrorCode       *string                                       `gorm:"type:varchar(100)"`
	LastProviderErrorAt         *time.Time                                    `gorm:"type:timestamp with time zone"`
	FailureReason               *string                                       `gorm:"type:varchar(100)"`
	RefundedAt                  *time.Time                                    `gorm:"type:timestamp with time zone"`
	RefundAmount                *decimal.Decimal                              `gorm:"type:numeric(12,2)"`
	RefundReason                *string                                       `gorm:"type:varchar(255)"`
	CancelledAt                 *time.Time                                    `gorm:"type:timestamp with time zone"`
	ExpiredAt                   *time.Time                                    `gorm:"type:timestamp with time zone"`
	PlanCode                    *string                                       `gorm:"type:varchar(100)"`
	PlanName                    *string                                       `gorm:"type:varchar(100)"`
	TierRank                    *int                                          `gorm:"type:int"`
	PlanKind                    *plankind.PlanKind                            `gorm:"type:smallint"`
	PurchasedDurationDays       *int                                          `gorm:"type:int"`
	ExpectedAmount              decimal.Decimal                               `gorm:"type:numeric(12,2);not null"`
	BenefitSnapshot             JSONDocument                                  `gorm:"type:jsonb"`
	PricingVersion              *int64                                        `gorm:"type:bigint"`
	BenefitResolution           *benefitresolution.BenefitResolution          `gorm:"type:varchar(100)"`
	BenefitAppliedAt            *time.Time                                    `gorm:"type:timestamp with time zone"`
	BenefitResultSnapshot       JSONDocument                                  `gorm:"type:jsonb"`
	WalletCreditAmount          *decimal.Decimal                              `gorm:"type:numeric(12,2)"`
	SubscriptionVersionBefore   *int64                                        `gorm:"type:bigint"`
	SubscriptionVersionAfter    *int64                                        `gorm:"type:bigint"`
}

type WalletStatement struct {
	AuditableEntity
	UserID                     uuid.UUID                               `gorm:"type:uuid;not null"`
	Amount                     decimal.Decimal                         `gorm:"type:numeric(12,2);not null"`
	TransactionType            walletstatementtype.WalletStatementType `gorm:"type:varchar(50);not null"`
	PreviousBalance            decimal.Decimal                         `gorm:"type:numeric(12,2);not null"`
	NewBalance                 decimal.Decimal                         `gorm:"type:numeric(12,2);not null"`
	ReferenceID                *uuid.UUID                              `gorm:"type:uuid"`
	SourcePlanCode             *string                                 `gorm:"type:varchar(100)"`
	SourceTierRank             *int                                    `gorm:"type:int"`
	ActiveTierRankAtCompletion *int                                    `gorm:"type:int"`
	RenewalAttemptKey          *string                                 `gorm:"type:varchar(255);uniqueIndex"`
	Description                string                                  `gorm:"type:varchar(255);not null"`
}

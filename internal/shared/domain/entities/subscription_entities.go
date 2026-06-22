package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/benefitresolution"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/plankind"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SubscriptionPlan struct {
	AuditableEntity
	Slug               string            `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name               string            `gorm:"type:varchar(100);not null"`
	Price              decimal.Decimal   `gorm:"type:numeric(12,2);not null;default:0.00"`
	MaxWardrobeItems   int               `gorm:"type:int;not null"`
	MaxOutfits         int               `gorm:"type:int;not null"`
	AiOutfitDailyQuota int               `gorm:"type:int;not null"`
	AiChatDailyQuota   int               `gorm:"type:int;not null"`
	DurationDays       *int              `gorm:"type:int"`
	IsActive           bool              `gorm:"type:boolean;not null;default:true"`
	PlanKind           plankind.PlanKind `gorm:"type:smallint;not null;default:0"`
	TierRank           int               `gorm:"type:int;not null;default:0"`
	PricingVersion     int64             `gorm:"type:bigint;not null;default:1"`
	AICostPolicyID     uuid.UUID         `gorm:"type:uuid;not null"`
	AICostPolicy       *AICostPolicy     `gorm:"foreignKey:AICostPolicyID;constraint:OnDelete:RESTRICT"`
}

type AICostPolicy struct {
	AuditableEntity
	Code                         string                   `gorm:"type:varchar(100);not null;uniqueIndex:ux_ai_cost_policy_code_version,priority:1"`
	Version                      int64                    `gorm:"type:bigint;not null;uniqueIndex:ux_ai_cost_policy_code_version,priority:2"`
	Name                         string                   `gorm:"type:varchar(150);not null"`
	EnforcementMode              string                   `gorm:"type:varchar(30);not null"`
	PeriodDays                   int                      `gorm:"type:int;not null"`
	HardCostMicroVND             *int64                   `gorm:"type:bigint"`
	CompactThresholdBPS          int                      `gorm:"type:int;not null"`
	FreeRouteThresholdBPS        int                      `gorm:"type:int;not null"`
	UnknownHoldMinutes           int                      `gorm:"type:int;not null"`
	MaxUnknownPaidRequestsPerDay int                      `gorm:"type:int;not null"`
	IsActive                     bool                     `gorm:"type:boolean;not null"`
	Operations                   []*AICostPolicyOperation `gorm:"foreignKey:PolicyID"`
}

type AICostPolicyOperation struct {
	AuditableEntity
	PolicyID               uuid.UUID `gorm:"type:uuid;not null"`
	Operation              string    `gorm:"type:varchar(30);not null"`
	NormalRoute            string    `gorm:"type:varchar(50);not null"`
	ReducedRoute           string    `gorm:"type:varchar(50);not null"`
	FreeRoute              string    `gorm:"type:varchar(50);not null"`
	NormalMaxInputTokens   int       `gorm:"type:int;not null"`
	NormalMaxOutputTokens  int       `gorm:"type:int;not null"`
	ReducedMaxInputTokens  int       `gorm:"type:int;not null"`
	ReducedMaxOutputTokens int       `gorm:"type:int;not null"`
	MaxPaidAttemptsPerDay  int       `gorm:"type:int;not null"`
	PaidFallbackEnabled    bool      `gorm:"type:boolean;not null"`
	IsEnabled              bool      `gorm:"type:boolean;not null"`
}

type UserAIPolicyGrant struct {
	AuditableEntity
	UserID                     uuid.UUID    `gorm:"type:uuid;not null"`
	PolicyID                   uuid.UUID    `gorm:"type:uuid;not null"`
	PlanID                     uuid.UUID    `gorm:"type:uuid;not null"`
	PlanCode                   string       `gorm:"type:varchar(100);not null"`
	TierRank                   int          `gorm:"type:int;not null"`
	PolicySnapshot             JSONDocument `gorm:"type:jsonb;not null"`
	EffectiveFrom              time.Time    `gorm:"type:timestamp with time zone;not null"`
	EffectiveTo                *time.Time   `gorm:"type:timestamp with time zone"`
	Status                     string       `gorm:"type:varchar(20);not null"`
	SourceEventID              *uuid.UUID   `gorm:"type:uuid"`
	SourceDepositTransactionID *uuid.UUID   `gorm:"type:uuid"`
}

type AIUsagePeriodLedger struct {
	AuditableEntity
	GrantID              uuid.UUID `gorm:"type:uuid;not null"`
	UserID               uuid.UUID `gorm:"type:uuid;not null"`
	PeriodIndex          int       `gorm:"type:int;not null"`
	PeriodStart          time.Time `gorm:"type:timestamp with time zone;not null"`
	PeriodEnd            time.Time `gorm:"type:timestamp with time zone;not null"`
	PaidInputTokens      int64     `gorm:"type:bigint;not null"`
	PaidOutputTokens     int64     `gorm:"type:bigint;not null"`
	FreeInputTokens      int64     `gorm:"type:bigint;not null"`
	FreeOutputTokens     int64     `gorm:"type:bigint;not null"`
	ActualCostMicroVND   int64     `gorm:"type:bigint;not null"`
	ReservedCostMicroVND int64     `gorm:"type:bigint;not null"`
}

type AIUsageEvent struct {
	AuditableEntity
	RequestID                uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex"`
	LedgerID                 uuid.UUID        `gorm:"type:uuid;not null"`
	UserID                   uuid.UUID        `gorm:"type:uuid;not null"`
	Operation                string           `gorm:"type:varchar(30);not null"`
	LogicalRoute             string           `gorm:"type:varchar(50);not null"`
	Provider                 *string          `gorm:"type:varchar(50)"`
	Model                    *string          `gorm:"type:varchar(150)"`
	PricingVersion           *string          `gorm:"type:varchar(100)"`
	InputUSDPerMillion       *decimal.Decimal `gorm:"type:numeric(18,8)"`
	OutputUSDPerMillion      *decimal.Decimal `gorm:"type:numeric(18,8)"`
	USDToVND                 *decimal.Decimal `gorm:"type:numeric(18,4)"`
	PromptTokens             int64            `gorm:"type:bigint;not null"`
	OutputTokens             int64            `gorm:"type:bigint;not null"`
	ThinkingTokens           int64            `gorm:"type:bigint;not null"`
	ReservedCostMicroVND     int64            `gorm:"type:bigint;not null"`
	ActualCostMicroVND       int64            `gorm:"type:bigint;not null"`
	EstimatedMaxCostMicroVND int64            `gorm:"type:bigint;not null"`
	Status                   string           `gorm:"type:varchar(30);not null"`
	FinishReason             *string          `gorm:"type:varchar(100)"`
	ErrorCode                *string          `gorm:"type:varchar(100)"`
	SentAt                   *time.Time       `gorm:"type:timestamp with time zone"`
	CompletedAt              *time.Time       `gorm:"type:timestamp with time zone"`
	UnknownExpiresAt         *time.Time       `gorm:"type:timestamp with time zone"`
}

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
	User      *User             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Balance   decimal.Decimal   `gorm:"type:numeric(12,2);not null;default:0.00"`
	Currency  currency.Currency `gorm:"type:varchar(10);not null;default:'VND'"`
	CreatedAt time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type DepositTransaction struct {
	AuditableEntity
	UserID                      uuid.UUID                                     `gorm:"type:uuid;not null"`
	User                        *User                                         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount                      decimal.Decimal                               `gorm:"type:numeric(12,2);not null"`
	Currency                    currency.Currency                             `gorm:"type:varchar(10);not null;default:'VND'"`
	Status                      depositstatus.DepositStatus                   `gorm:"type:smallint;not null;default:3"`
	TransactionType             deposittransactiontype.DepositTransactionType `gorm:"type:varchar(50);not null"` // DIRECT_PURCHASE, WALLET_TOPUP
	SubscriptionPlanID          *uuid.UUID                                    `gorm:"type:uuid"`
	SubscriptionPlan            *SubscriptionPlan                             `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:SET NULL"`
	OrderCode                   int64                                         `gorm:"type:bigint;autoIncrement;uniqueIndex;not null"`
	Provider                    string                                        `gorm:"type:varchar(50);not null;default:'PAYOS'"`
	PaymentLinkID               *string                                       `gorm:"type:varchar(255)"`
	ProviderStatus              *string                                       `gorm:"type:varchar(50)"`
	SuccessfulProviderReference *string                                       `gorm:"type:varchar(255)"`
	GatewayDetails              *string                                       `gorm:"type:text"` // Raw JSON string payload from gateway
	PaymentUrl                  *string                                       `gorm:"type:varchar(500)"`
	ExpiresAt                   *time.Time                                    `gorm:"type:timestamp with time zone"`
	NextReconciliationAt        *time.Time                                    `gorm:"type:timestamp with time zone"`
	ReconciliationAttempts      int                                           `gorm:"type:int;not null;default:0"`
	ProcessingToken             *uuid.UUID                                    `gorm:"type:uuid"`
	ProcessingLeaseUntil        *time.Time                                    `gorm:"type:timestamp with time zone"`
	LastProviderErrorCode       *string                                       `gorm:"type:varchar(100)"`
	LastProviderErrorAt         *time.Time                                    `gorm:"type:timestamp with time zone"`
	FailureReason               *string                                       `gorm:"type:varchar(100)"`
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
	User                       *User                                   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount                     decimal.Decimal                         `gorm:"type:numeric(12,2);not null"`
	TransactionType            walletstatementtype.WalletStatementType `gorm:"type:varchar(50);not null"` // TOPUP, SUBSCRIPTION_PURCHASE, SUBSCRIPTION_RENEWAL
	PreviousBalance            decimal.Decimal                         `gorm:"type:numeric(12,2);not null"`
	NewBalance                 decimal.Decimal                         `gorm:"type:numeric(12,2);not null"`
	ReferenceID                *uuid.UUID                              `gorm:"type:uuid"`
	SourcePlanCode             *string                                 `gorm:"type:varchar(100)"`
	SourceTierRank             *int                                    `gorm:"type:int"`
	ActiveTierRankAtCompletion *int                                    `gorm:"type:int"`
	RenewalAttemptKey          *string                                 `gorm:"type:varchar(255);uniqueIndex"`
	Description                string                                  `gorm:"type:varchar(255);not null"`
}

type ProviderPaymentEvent struct {
	AuditableEntity
	Provider          string            `gorm:"type:varchar(50);not null;uniqueIndex:ux_provider_reference,priority:1"`
	ProviderReference string            `gorm:"type:varchar(255);not null;uniqueIndex:ux_provider_reference,priority:2"`
	EventCode         string            `gorm:"type:varchar(100);not null"`
	OrderCode         int64             `gorm:"type:bigint;not null"`
	PaymentLinkID     string            `gorm:"type:varchar(255)"`
	Amount            decimal.Decimal   `gorm:"type:numeric(12,2);not null"`
	Currency          currency.Currency `gorm:"type:varchar(10);not null"`
}

type ProviderWebhookInbox struct {
	AuditableEntity
	Provider             string            `gorm:"type:varchar(50);not null"`
	ProviderReference    *string           `gorm:"type:varchar(255)"`
	EventCode            string            `gorm:"type:varchar(100);not null"`
	OrderCode            int64             `gorm:"type:bigint;not null"`
	PaymentLinkID        *string           `gorm:"type:varchar(255)"`
	Amount               decimal.Decimal   `gorm:"type:numeric(12,2);not null"`
	Currency             currency.Currency `gorm:"type:varchar(10);not null"`
	CanonicalPayloadHash string            `gorm:"type:varchar(64);not null"`
	RawPayload           JSONDocument      `gorm:"type:jsonb;not null"`
	ProcessingStatus     string            `gorm:"type:varchar(50);not null"`
	ProcessingAttempts   int               `gorm:"type:int;not null;default:0"`
	NextProcessingAt     *time.Time        `gorm:"type:timestamp with time zone"`
	ProcessingToken      *uuid.UUID        `gorm:"type:uuid"`
	ProcessingLeaseUntil *time.Time        `gorm:"type:timestamp with time zone"`
	ProcessingError      *string           `gorm:"type:text"`
	ReceivedAt           time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	ProcessedAt          *time.Time        `gorm:"type:timestamp with time zone"`
}

type UserSubscriptionEvent struct {
	AuditableEntity
	EventKey                   string       `gorm:"type:varchar(255);not null;uniqueIndex"`
	UserID                     uuid.UUID    `gorm:"type:uuid;not null"`
	EventType                  string       `gorm:"type:varchar(100);not null"`
	FromPlanCode               *string      `gorm:"type:varchar(100)"`
	FromTierRank               *int         `gorm:"type:int"`
	ToPlanCode                 *string      `gorm:"type:varchar(100)"`
	ToTierRank                 *int         `gorm:"type:int"`
	SourceDepositTransactionID *uuid.UUID   `gorm:"type:uuid"`
	ActorAdminID               *uuid.UUID   `gorm:"type:uuid"`
	OccurredAt                 time.Time    `gorm:"type:timestamp with time zone;not null"`
	EffectiveAt                time.Time    `gorm:"type:timestamp with time zone;not null"`
	Metadata                   JSONDocument `gorm:"type:jsonb"`
}

type SubscriptionRenewalAttempt struct {
	AuditableEntity
	RenewalAttemptKey           string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	UserID                      uuid.UUID  `gorm:"type:uuid;not null"`
	ExpectedPlanID              uuid.UUID  `gorm:"type:uuid;not null"`
	ExpectedExpiresAt           time.Time  `gorm:"type:timestamp with time zone;not null"`
	ExpectedSubscriptionVersion int64      `gorm:"type:bigint;not null"`
	Status                      string     `gorm:"type:varchar(50);not null"`
	AttemptCount                int        `gorm:"type:int;not null;default:0"`
	LastErrorCode               *string    `gorm:"type:varchar(100)"`
	LastErrorMessage            *string    `gorm:"type:text"`
	ProcessingToken             *uuid.UUID `gorm:"type:uuid"`
	ProcessingLeaseUntil        *time.Time `gorm:"type:timestamp with time zone"`
	CompletedAt                 *time.Time `gorm:"type:timestamp with time zone"`
}

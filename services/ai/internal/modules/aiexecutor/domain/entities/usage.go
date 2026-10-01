package entities

import (
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	"smart-wardrobe-be/shared/domain/constants/tokenestimationmethod"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AIUsagePeriodLedger struct {
	AuditableEntity
	// GrantID is nullable for exactly one case: a fallback ledger row, created
	// for an account that has no policy grant yet (024 FR-007). Such a row is
	// deduped by the partial unique index ux_ledgers_user_period_fallback on
	// (user_id, period_index) WHERE grant_id IS NULL, because the pre-existing
	// UNIQUE (grant_id, period_index) cannot dedupe NULLs — SQL treats NULL as
	// distinct from NULL. See research.md R8.
	GrantID              *uuid.UUID `gorm:"type:uuid"`
	UserID               uuid.UUID  `gorm:"type:uuid;not null"`
	PeriodIndex          int        `gorm:"type:int;not null"`
	PeriodStart          time.Time  `gorm:"type:timestamp with time zone;not null"`
	PeriodEnd            time.Time  `gorm:"type:timestamp with time zone;not null"`
	PaidInputTokens      int64      `gorm:"type:bigint;not null"`
	PaidOutputTokens     int64      `gorm:"type:bigint;not null"`
	FreeInputTokens      int64      `gorm:"type:bigint;not null"`
	FreeOutputTokens     int64      `gorm:"type:bigint;not null"`
	ActualCostMicroVND   int64      `gorm:"type:bigint;not null"`
	ReservedCostMicroVND int64      `gorm:"type:bigint;not null"`
}

type AIUsageEvent struct {
	AuditableEntity
	RequestID                uuid.UUID                             `gorm:"type:uuid;not null;uniqueIndex"`
	LedgerID                 *uuid.UUID                            `gorm:"type:uuid"`
	UserID                   *uuid.UUID                            `gorm:"type:uuid"`
	PricingConfigID          *uuid.UUID                            `gorm:"type:uuid"`
	Operation                aioperation.AIOperation               `gorm:"type:varchar(30);not null"`
	LogicalRoute             airoute.AIRoute                       `gorm:"type:varchar(50);not null"`
	Provider                 *string                               `gorm:"type:varchar(50)"`
	Model                    *string                               `gorm:"type:varchar(150)"`
	InputUSDPerMillion       *decimal.Decimal                      `gorm:"type:numeric(18,8)"`
	OutputUSDPerMillion      *decimal.Decimal                      `gorm:"type:numeric(18,8)"`
	USDToVND                 *decimal.Decimal                      `gorm:"type:numeric(18,4)"`
	PromptTokens             int64                                 `gorm:"type:bigint;not null"`
	OutputTokens             int64                                 `gorm:"type:bigint;not null"`
	ThinkingTokens           int64                                 `gorm:"type:bigint;not null"`
	ReservedCostMicroVND     int64                                 `gorm:"type:bigint;not null"`
	ActualCostMicroVND       int64                                 `gorm:"type:bigint;not null"`
	EstimatedMaxCostMicroVND int64                                 `gorm:"type:bigint;not null"`
	Status                   aiusageeventstatus.AIUsageEventStatus `gorm:"type:varchar(30);not null"`
	FinishReason             *string                               `gorm:"type:varchar(100)"`
	ErrorCode                *string                               `gorm:"type:varchar(100)"`
	EstimatedPromptTokens    *int64                                `gorm:"type:bigint"`
	TokenEstimationMethod    *tokenestimationmethod.Method         `gorm:"type:varchar(30)"`
	TokenCountLatencyMs      *int64                                `gorm:"type:bigint"`
	SentAt                   *time.Time                            `gorm:"type:timestamp with time zone"`
	CompletedAt              *time.Time                            `gorm:"type:timestamp with time zone"`
	UnknownExpiresAt         *time.Time                            `gorm:"type:timestamp with time zone"`
}

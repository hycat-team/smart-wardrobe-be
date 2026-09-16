package entities

import (
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"

	"github.com/google/uuid"
)

type AICostPolicy struct {
	AuditableEntity
	Code                         string                              `gorm:"type:varchar(100);not null;uniqueIndex:ux_ai_cost_policy_code_version,priority:1"`
	Version                      int64                               `gorm:"type:bigint;not null;uniqueIndex:ux_ai_cost_policy_code_version,priority:2"`
	Name                         string                              `gorm:"type:varchar(150);not null"`
	EnforcementMode              aienforcementmode.AIEnforcementMode `gorm:"type:varchar(30);not null"`
	PeriodDays                   int                                 `gorm:"type:int;not null"`
	HardCostMicroVND             *int64                              `gorm:"type:bigint"`
	CompactThresholdBPS          int                                 `gorm:"type:int;not null"`
	FreeRouteThresholdBPS        int                                 `gorm:"type:int;not null"`
	UnknownHoldMinutes           int                                 `gorm:"type:int;not null"`
	MaxUnknownPaidRequestsPerDay int                                 `gorm:"type:int;not null"`
	IsActive                     bool                                `gorm:"type:boolean;not null"`
	Operations                   []*AICostPolicyOperation            `gorm:"foreignKey:PolicyID"`
}

type AICostPolicyOperation struct {
	AuditableEntity
	PolicyID               uuid.UUID               `gorm:"type:uuid;not null"`
	Operation              aioperation.AIOperation `gorm:"type:varchar(30);not null"`
	NormalRoute            airoute.AIRoute         `gorm:"type:varchar(50);not null"`
	ReducedRoute           airoute.AIRoute         `gorm:"type:varchar(50);not null"`
	FreeRoute              airoute.AIRoute         `gorm:"type:varchar(50);not null"`
	NormalMaxInputTokens   int                     `gorm:"type:int;not null"`
	NormalMaxOutputTokens  int                     `gorm:"type:int;not null"`
	ReducedMaxInputTokens  int                     `gorm:"type:int;not null"`
	ReducedMaxOutputTokens int                     `gorm:"type:int;not null"`
	MaxPaidAttemptsPerDay  int                     `gorm:"type:int;not null"`
	PaidFallbackEnabled    bool                    `gorm:"type:boolean;not null"`
	IsEnabled              bool                    `gorm:"type:boolean;not null"`
}

type AIPolicyGrant struct {
	AuditableEntity
	UserID                     uuid.UUID                               `gorm:"type:uuid;not null"`
	PolicyID                   uuid.UUID                               `gorm:"type:uuid;not null"`
	PlanID                     uuid.UUID                               `gorm:"type:uuid;not null"`
	PlanCode                   string                                  `gorm:"type:varchar(100);not null"`
	TierRank                   int                                     `gorm:"type:int;not null"`
	PolicySnapshot             JSONDocument                            `gorm:"type:jsonb;not null"`
	EffectiveFrom              time.Time                               `gorm:"type:timestamp with time zone;not null"`
	EffectiveTo                *time.Time                              `gorm:"type:timestamp with time zone"`
	Status                     aipolicygrantstatus.AIPolicyGrantStatus `gorm:"type:varchar(20);not null"`
	SourceEventID              *uuid.UUID                              `gorm:"type:uuid"`
	SourceDepositTransactionID *uuid.UUID                              `gorm:"type:uuid"`
}

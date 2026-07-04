package entities

import (
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/constants/subscriptionrenewalstatus"

	
	"github.com/google/uuid"
)

type UserSubscriptionEvent struct {
	AuditableEntity
	EventKey                   string                    `gorm:"type:varchar(255);not null;uniqueIndex"`
	UserID                     uuid.UUID                 `gorm:"type:uuid;not null"`
	EventType                  string                    `gorm:"type:varchar(100);not null"`
	FromPlanCode               *string                   `gorm:"type:varchar(100)"`
	FromTierRank               *int                      `gorm:"type:int"`
	ToPlanCode                 *string                   `gorm:"type:varchar(100)"`
	ToTierRank                 *int                      `gorm:"type:int"`
	SourceDepositTransactionID *uuid.UUID                `gorm:"type:uuid"`
	ActorAdminID               *uuid.UUID                `gorm:"type:uuid"`
	OccurredAt                 time.Time                 `gorm:"type:timestamp with time zone;not null"`
	EffectiveAt                time.Time                 `gorm:"type:timestamp with time zone;not null"`
	Metadata                   JSONDocument `gorm:"type:jsonb"`
}

type SubscriptionRenewalAttempt struct {
	AuditableEntity
	RenewalAttemptKey           string                                              `gorm:"type:varchar(255);not null;uniqueIndex"`
	UserID                      uuid.UUID                                           `gorm:"type:uuid;not null"`
	ExpectedPlanID              uuid.UUID                                           `gorm:"type:uuid;not null"`
	ExpectedExpiresAt           time.Time                                           `gorm:"type:timestamp with time zone;not null"`
	ExpectedSubscriptionVersion int64                                               `gorm:"type:bigint;not null"`
	Status                      subscriptionrenewalstatus.SubscriptionRenewalStatus `gorm:"type:varchar(50);not null"`
	AttemptCount                int                                                 `gorm:"type:int;not null;default:0"`
	LastErrorCode               *string                                             `gorm:"type:varchar(100)"`
	LastErrorMessage            *string                                             `gorm:"type:text"`
	ProcessingToken             *uuid.UUID                                          `gorm:"type:uuid"`
	ProcessingLeaseUntil        *time.Time                                          `gorm:"type:timestamp with time zone"`
	CompletedAt                 *time.Time                                          `gorm:"type:timestamp with time zone"`
}

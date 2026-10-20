package entities

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"

	"github.com/google/uuid"
)

// UserPlanPeriod is one continuous period during which a user held a plan
// (`subscription.user_plan_periods`, 031 data-model §1).
//
// It is an append-only read model of the subscription's past, written by the
// recorder inside the same transaction as the plan change. The plan attributes
// are SNAPSHOTS: renaming or repricing the catalogue must not rewrite history,
// and `plan_code` is the code the user actually held.
//
// `ValidTo == nil` means open-ended (free / lifetime). With the exclusion
// constraint installed two intervals of one user can never overlap; in the
// fallback variant only the open interval is unique, and the recorder's
// per-user advisory lock, row lock and close rules keep the sequence contiguous.
type UserPlanPeriod struct {
	AuditableEntity
	UserID             uuid.UUID         `gorm:"type:uuid;not null"`
	SubscriptionPlanID uuid.UUID         `gorm:"type:uuid;not null"`
	PlanCode           string            `gorm:"type:varchar(100);not null"`
	PlanName           string            `gorm:"type:varchar(255);not null"`
	TierRank           int               `gorm:"type:int;not null"`
	PlanKind           plankind.PlanKind `gorm:"type:smallint;not null"`
	ValidFrom          time.Time         `gorm:"type:timestamp with time zone;not null"`
	ValidTo            *time.Time        `gorm:"type:timestamp with time zone"`
	// SourceEventKey is UNIQUE but nullable: the backfill and any future
	// maintenance row are not tied to a plan-change event. A NULL is allowed
	// once per row because SQL treats NULLs as distinct.
	SourceEventKey *string `gorm:"type:varchar(255);uniqueIndex"`
}

// PlanHistoryMeta records when complete history starts
// (`subscription.plan_history_meta`, 031 data-model §2).
//
// Only the `cutover` row is used. Its absence (manual truncation, partial
// restore) MUST NOT be read as "history starts at the beginning of time"; the
// report turns a period request into `unavailable` instead.
type PlanHistoryMeta struct {
	ID        string    `gorm:"type:varchar(20);primaryKey"`
	StartedAt time.Time `gorm:"type:timestamp with time zone;not null"`
	Note      *string   `gorm:"type:text"`
}

package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

type IUserSubscriptionEventRepository interface {
	Create(ctx context.Context, event *entities.UserSubscriptionEvent) error
	// CreateIfAbsent writes the history row unless its `event_key` already exists.
	//
	// It returns whether the row was actually inserted. `Create` cannot answer
	// that, and the caller needs to know: 024 T148 added it because `event_key` is
	// UNIQUE and two concurrent first-provisions of one account compute the SAME
	// key (`DEFAULT_PROVISION:{userID}:{version}`), so the check-then-insert in
	// PublishPlanChange has a window between them. The loser of that race used to
	// fail with a unique-violation and return HTTP 500 for an account whose
	// subscription was in fact created correctly.
	//
	// Separate from Create rather than a change to it: Create's plain INSERT is the
	// right behaviour everywhere the key is unique by construction, and turning it
	// into a silent no-op would hide genuine duplicate-key bugs.
	CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (inserted bool, err error)
	GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error)
}

type ISubscriptionRenewalAttemptRepository interface {
	Create(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error
	GetByKey(ctx context.Context, key string) (*entities.SubscriptionRenewalAttempt, error)
	Update(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error
	// GetRenewalStats summarises renewal attempts created in [from, to) (029 R8
	// /data-model §6: the attempt belongs to the period it was created in). A
	// zero bound leaves that side open; both zero keeps the all-time behaviour.
	GetRenewalStats(ctx context.Context, from, to time.Time) (*RenewalStatsDTO, error)
}

package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// SubscriptionDistributionResult is the report's read model (031 data-model §5).
//
// `Items` are the plan buckets. `DataAvailableFrom` is the cutover instant when
// complete history begins, or nil when the `plan_history_meta` row is absent.
// `DataCompleteness` is one of `complete`, `partial`, `unavailable`; the use
// case computes it from the requested bounds and the cutover. The no-period
// snapshot stays `complete` but still carries the cutover date when it exists.
type SubscriptionDistributionResult struct {
	Items             []SubscriptionDistributionDTO
	DataAvailableFrom *time.Time
	DataCompleteness  string
}

// IUserPlanPeriodRepository owns the validity-interval history
// (`subscription.user_plan_periods`, 031) and its cutover metadata.
//
// Every method resolves its handle from the transaction injected into ctx, so
// the recorder and the business mutation commit or roll back together
// (constitution V). The interface stays in the domain layer; GORM lives in
// infrastructure.
type IUserPlanPeriodRepository interface {
	// GetDistribution counts DISTINCT users per plan for intervals overlapping
	// [from, to). A zero bound means unbounded. Both bounds zero means "the
	// snapshot at now()". A `to`-only period additionally excludes intervals
	// whose valid_from is in the future (FR-008/FR-012).
	GetDistribution(ctx context.Context, from, to time.Time) ([]SubscriptionDistributionDTO, error)

	// LockUser serializes the recorders of one user until the caller's
	// transaction ends. It takes a PostgreSQL transaction-scoped advisory lock
	// keyed on the user id text, so no extension, lock table or configuration
	// switch is required. The recorder calls it BEFORE reading the current
	// interval: in the fallback (partial-unique) variant, where the exclusion
	// constraint is absent, this is what prevents two concurrent plan changes
	// from both reading the old row and opening overlapping intervals.
	LockUser(ctx context.Context, userID uuid.UUID) error

	// GetCurrentByUserIDWithLock returns the user's current interval - the row
	// with the greatest valid_from (created_at DESC, id DESC break ties) -
	// locked FOR UPDATE, or nil when the user has no row at all. The row may be
	// open-ended or finite with a future end: a paid interval stores
	// valid_to = expires_at, so "current" must not be filtered on `valid_to IS
	// NULL`. The row lock is the second serialization guard, behind LockUser;
	// on databases that installed btree_gist the exclusion constraint is the
	// third.
	GetCurrentByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserPlanPeriod, error)

	// CloseCurrent moves the interval's exclusive end to closeAt and touches
	// updated_at. It updates by id without a `valid_to IS NULL` predicate, so a
	// finite interval's future expiry can be moved earlier to the switch
	// instant. Callers hold the row lock taken by GetCurrentByUserIDWithLock,
	// which makes repeating the same close idempotent.
	CloseCurrent(ctx context.Context, periodID uuid.UUID, closeAt time.Time) error

	// Create inserts one interval row.
	Create(ctx context.Context, period *entities.UserPlanPeriod) error

	// Update replaces an existing interval row in place. The recorder uses it
	// when a change lands exactly at (or before) the current interval's start:
	// replacing keeps one row and avoids a zero-length interval, which the
	// CHECK constraint forbids (031 data-model §3).
	Update(ctx context.Context, period *entities.UserPlanPeriod) error

	// UpdateValidTo moves the current interval's exclusive end (EXTENDED) and
	// touches updated_at. A nil validTo is stored as NULL.
	UpdateValidTo(ctx context.Context, periodID uuid.UUID, validTo *time.Time) error

	// GetByUserID returns the user's intervals ordered by valid_from.
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.UserPlanPeriod, error)

	// GetHistoryCutover returns the `cutover` metadata row, or nil when it is
	// absent. Absence is NOT an error: the caller must render `unavailable`
	// rather than inventing a start date.
	GetHistoryCutover(ctx context.Context) (*entities.PlanHistoryMeta, error)
}

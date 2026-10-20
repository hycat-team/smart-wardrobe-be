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

	// GetOpenByUserIDWithLock returns the user's open interval (valid_to IS NULL)
	// locked FOR UPDATE, or nil when the user has none. This is the serialization
	// point of the recorder in the fallback (partial-unique) variant.
	GetOpenByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserPlanPeriod, error)

	// CloseOpen moves the interval's exclusive end to closeAt and touches
	// updated_at. It is a no-op for an id that is no longer open.
	CloseOpen(ctx context.Context, periodID uuid.UUID, closeAt time.Time) error

	// Create inserts one interval row.
	Create(ctx context.Context, period *entities.UserPlanPeriod) error

	// Update replaces an existing interval row in place. The recorder uses it
	// when a change lands exactly at (or before) the open interval's start:
	// replacing keeps one row and avoids a zero-length interval, which the
	// CHECK constraint forbids (031 data-model §3).
	Update(ctx context.Context, period *entities.UserPlanPeriod) error

	// UpdateValidTo moves the open interval's exclusive end (EXTENDED) and
	// touches updated_at. A nil validTo is stored as NULL.
	UpdateValidTo(ctx context.Context, periodID uuid.UUID, validTo *time.Time) error

	// GetByUserID returns the user's intervals ordered by valid_from.
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.UserPlanPeriod, error)

	// GetHistoryCutover returns the `cutover` metadata row, or nil when it is
	// absent. Absence is NOT an error: the caller must render `unavailable`
	// rather than inventing a start date.
	GetHistoryCutover(ctx context.Context) (*entities.PlanHistoryMeta, error)
}

package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserPlanPeriodRepository reads and writes the validity-interval history
// (`subscription.user_plan_periods`, 031).
//
// The report queries use the range expression rather than `valid_from`/
// `valid_to` comparisons so PostgreSQL can serve them from
// `idx_user_plan_periods_range` (GiST, no extension required). The recorder
// first takes a per-user advisory lock (`LockUser`) and then reads the current
// interval FOR UPDATE; the advisory lock is what serializes concurrent changes
// of one user in the fallback (partial-unique) variant, where the exclusion
// constraint is absent.
type UserPlanPeriodRepository struct {
	*shared_repos.GenericRepository[entities.UserPlanPeriod, uuid.UUID]
}

func NewUserPlanPeriodRepository(dbConn *gorm.DB) repositories.IUserPlanPeriodRepository {
	return &UserPlanPeriodRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.UserPlanPeriod, uuid.UUID](dbConn, nil),
	}
}

// GetDistribution counts DISTINCT users per plan for the requested period
// (031 data-model §5, D4/FR-008).
//
// The bounds select one of four shapes:
//   - both zero: snapshot at now() (`@> now()`), the historical "current state"
//     semantics;
//   - both set: half-open [from, to) overlap;
//   - from only (`to` zero): [from, infinity) AND valid_from <= now(), so a
//     future-dated interval is never counted when the missing end means "up to
//     now" (FR-008/A2);
//   - to only (`from` zero): (-infinity, to).
//
// A zero bound is passed to SQL as NULL, never as Go's zero time.
func (r *UserPlanPeriodRepository) GetDistribution(ctx context.Context, from, to time.Time) ([]repositories.SubscriptionDistributionDTO, error) {
	query := r.GetDB(ctx).Model(&entities.UserPlanPeriod{})

	switch {
	case from.IsZero() && to.IsZero():
		query = query.Where("tstzrange(valid_from, valid_to, '[)') @> now()")
	case !from.IsZero() && !to.IsZero():
		query = query.Where("tstzrange(valid_from, valid_to, '[)') && tstzrange(?, ?, '[)')", from, to)
	case !from.IsZero():
		query = query.
			Where("tstzrange(valid_from, valid_to, '[)') && tstzrange(?, NULL, '[)')", from).
			Where("valid_from <= now()")
	default:
		query = query.Where("tstzrange(valid_from, valid_to, '[)') && tstzrange(NULL, ?, '[)')", to)
	}

	var rows []repositories.SubscriptionDistributionDTO
	err := query.
		Select("plan_code, COUNT(DISTINCT user_id) AS user_count").
		Group("plan_code").
		Order("user_count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// LockUser takes a transaction-scoped PostgreSQL advisory lock for one user.
//
// `pg_advisory_xact_lock(hashtext($1))` is a plain PostgreSQL function: it
// needs no extension and the server releases it automatically at COMMIT or
// ROLLBACK. Passing the user id as text gives every transaction of that user
// the same 64-bit key, so their writes serialize even in the fallback
// (partial-unique) variant, where two closed intervals could otherwise overlap.
// The lock is resolved from ctx, so without a caller transaction it attaches to
// the implicit transaction of the statement and is useless for serialization -
// the recorder always runs inside the unit of work.
func (r *UserPlanPeriodRepository) LockUser(ctx context.Context, userID uuid.UUID) error {
	return r.GetDB(ctx).
		Exec("SELECT pg_advisory_xact_lock(hashtext(?))", userID.String()).Error
}

// GetCurrentByUserIDWithLock returns the user's current interval locked FOR
// UPDATE.
//
// "Current" is the row with the greatest valid_from, NOT the row whose valid_to
// is NULL: every finite paid interval stores valid_to = expires_at, so filtering
// on `valid_to IS NULL` made the recorder miss paid holdings entirely. created_at
// and id make the choice deterministic when two rows share a start. The row may
// be open-ended or finite; the caller decides whether to close or extend it.
//
// The row lock is the second guard against two concurrent plan changes of the
// same user, behind the advisory lock the recorder takes first. It is taken
// inside the caller's transaction (resolved from ctx), so it is released at
// COMMIT/ROLLBACK and never leaks onto a pooled connection.
func (r *UserPlanPeriodRepository) GetCurrentByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserPlanPeriod, error) {
	var period entities.UserPlanPeriod
	err := r.GetDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		Order("valid_from DESC, created_at DESC, id DESC").
		Take(&period).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &period, nil
}

// CloseCurrent sets the exclusive end of an interval and moves updated_at.
//
// It updates by id without a `valid_to IS NULL` predicate. That is deliberate:
// a finite paid interval already carries its contractual expiry, and a switch
// must be able to move that end earlier to the actual close instant. The caller
// holds the row lock taken by GetCurrentByUserIDWithLock, so repeating the same
// close writes the same value and stays idempotent.
func (r *UserPlanPeriodRepository) CloseCurrent(ctx context.Context, periodID uuid.UUID, closeAt time.Time) error {
	return r.GetDB(ctx).
		Model(&entities.UserPlanPeriod{}).
		Where("id = ?", periodID).
		Updates(map[string]any{
			"valid_to":   closeAt,
			"updated_at": time.Now().UTC(),
		}).Error
}

// Create inserts one interval row inside the caller's transaction.
func (r *UserPlanPeriodRepository) Create(ctx context.Context, period *entities.UserPlanPeriod) error {
	return r.GetDB(ctx).Omit(clause.Associations).Create(period).Error
}

// Update replaces an existing interval row in place. Used by the recorder's
// same-instant branch: closing and re-opening at the same instant would create
// a zero-length interval that `chk_user_plan_periods_valid_range` rejects.
func (r *UserPlanPeriodRepository) Update(ctx context.Context, period *entities.UserPlanPeriod) error {
	return r.GetDB(ctx).Omit(clause.Associations).Save(period).Error
}

// UpdateValidTo moves the current interval's exclusive end (EXTENDED, FR-004)
// and moves updated_at with it. It NEVER inserts a row: one continuous holding
// must stay one interval.
func (r *UserPlanPeriodRepository) UpdateValidTo(ctx context.Context, periodID uuid.UUID, validTo *time.Time) error {
	return r.GetDB(ctx).
		Model(&entities.UserPlanPeriod{}).
		Where("id = ?", periodID).
		Updates(map[string]any{
			"valid_to":   validTo,
			"updated_at": time.Now().UTC(),
		}).Error
}

// GetByUserID returns the user's intervals ordered by start (US2).
func (r *UserPlanPeriodRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.UserPlanPeriod, error) {
	var periods []*entities.UserPlanPeriod
	err := r.GetDB(ctx).
		Where("user_id = ?", userID).
		Order("valid_from ASC").
		Find(&periods).Error
	if err != nil {
		return nil, err
	}
	return periods, nil
}

// GetHistoryCutover returns the `cutover` metadata row.
//
// A missing row is reported as (nil, nil), not as an error: the report has a
// defined answer for it (`unavailable` for a period request), and collapsing
// "missing" into "broken database" would make a truncated metadata table look
// like an outage.
func (r *UserPlanPeriodRepository) GetHistoryCutover(ctx context.Context) (*entities.PlanHistoryMeta, error) {
	var meta entities.PlanHistoryMeta
	err := r.GetDB(ctx).Where("id = ?", "cutover").First(&meta).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

var _ repositories.IUserPlanPeriodRepository = (*UserPlanPeriodRepository)(nil)

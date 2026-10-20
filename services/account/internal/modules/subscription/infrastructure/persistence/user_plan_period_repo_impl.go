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
// reads the open interval FOR UPDATE, which is what serializes concurrent
// changes of one user in the fallback (partial-unique) variant.
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

// GetOpenByUserIDWithLock returns the user's open interval locked FOR UPDATE.
//
// The row lock is the serialization point for two concurrent plan changes of
// the same user. It is taken inside the caller's transaction (resolved from
// ctx), so it is released at COMMIT/ROLLBACK and never leaks onto a pooled
// connection.
func (r *UserPlanPeriodRepository) GetOpenByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserPlanPeriod, error) {
	var period entities.UserPlanPeriod
	err := r.GetDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND valid_to IS NULL", userID).
		First(&period).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &period, nil
}

// CloseOpen sets the exclusive end of an interval and moves updated_at with it.
//
// `valid_to IS NULL` in the predicate makes a repeat harmless: a retry or a
// concurrent writer that already closed the row cannot move a historical end.
func (r *UserPlanPeriodRepository) CloseOpen(ctx context.Context, periodID uuid.UUID, closeAt time.Time) error {
	return r.GetDB(ctx).
		Model(&entities.UserPlanPeriod{}).
		Where("id = ? AND valid_to IS NULL", periodID).
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

// UpdateValidTo moves the open interval's exclusive end (EXTENDED, FR-004) and
// moves updated_at with it. It NEVER inserts a row: one continuous holding must
// stay one interval.
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

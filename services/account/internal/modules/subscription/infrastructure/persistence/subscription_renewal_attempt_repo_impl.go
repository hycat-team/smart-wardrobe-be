package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/infrastructure/db"
)

type SubscriptionRenewalAttemptRepository struct{ db *gorm.DB }

func NewSubscriptionRenewalAttemptRepository(conn *gorm.DB) repositories.ISubscriptionRenewalAttemptRepository {
	return &SubscriptionRenewalAttemptRepository{db: conn}
}
func (r *SubscriptionRenewalAttemptRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *SubscriptionRenewalAttemptRepository) Create(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error {
	return r.current(ctx).Create(attempt).Error
}
func (r *SubscriptionRenewalAttemptRepository) GetByKey(ctx context.Context, key string) (*entities.SubscriptionRenewalAttempt, error) {
	var attempt entities.SubscriptionRenewalAttempt
	err := r.current(ctx).Where("renewal_attempt_key = ?", key).First(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attempt, err
}
func (r *SubscriptionRenewalAttemptRepository) Update(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error {
	return r.current(ctx).Save(attempt).Error
}

// GetRenewalStats summarises renewal attempts. Only attempts that have left
// the `processing` state are counted: an in-flight attempt is not a completed
// renewal, so including it would silently deflate the success rate. The
// success/failed split stays literal; the caller derives the success rate from
// those two counts so non-payment outcomes (skipped/downgraded/renewed) do not
// read as failures.
//
// 029 R8/data-model §6: with a period, only attempts CREATED in [from, to)
// count — an attempt belongs to the period it was created in, which keeps
// in-flight attempts of the period visible while an attempt created before the
// period and completed inside it stays out. A zero bound leaves that side
// open; both zero keeps the all-time behaviour.
func (r *SubscriptionRenewalAttemptRepository) GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error) {
	var result dto.RenewalStatsDTO
	query := r.current(ctx).
		Table("subscription_renewal_attempts").
		Select("COUNT(*) FILTER (WHERE status <> 'processing') AS total_attempts, " +
			"COALESCE(SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END), 0) AS success_count, " +
			"COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_count, " +
			"COALESCE(AVG(attempt_count) FILTER (WHERE status <> 'processing'), 0) AS avg_retry_count")
	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at < ?", to)
	}
	err := query.Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

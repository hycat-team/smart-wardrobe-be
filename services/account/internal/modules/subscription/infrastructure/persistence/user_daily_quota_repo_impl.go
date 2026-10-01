package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserDailyQuotaRepository implements database actions for daily user resource usage quotas
type UserDailyQuotaRepository struct {
	*shared_repos.GenericRepository[entities.UserDailyQuota, uuid.UUID]
}

// NewUserDailyQuotaRepository creates a new instance of quota repository
func NewUserDailyQuotaRepository(dbConn *gorm.DB) repositories.IUserDailyQuotaRepository {
	relations := []string{}
	return &UserDailyQuotaRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.UserDailyQuota, uuid.UUID](dbConn, relations),
	}
}

// GetByUserID retrieves daily usage metrics for a specific user
func (r *UserDailyQuotaRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	var quota entities.UserDailyQuota
	err := r.GetDB(ctx).Where("user_id = ?", userID).First(&quota).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &quota, nil
}

// CreateIfAbsent writes the quota row unless the account already has one, and
// reports which of the two happened (024 T148).
//
// `user_id` is the primary key, so the plain INSERT inherited from the generic
// repository is a landmine: two concurrent first reads of a brand-new account -
// `GET /subscriptions/me` and `GET /subscriptions/me/daily-quota` in parallel, or
// the AI hot path racing the profile fetch - both see no row, both insert, and the
// second one fails. Measured on the development database, that produced 2 HTTP 500
// responses out of 12 parallel requests.
//
// The insert is what serialises the race, not any check that precedes it: the
// losing transaction blocks on the winner's primary key and then reports zero rows
// affected. The caller re-reads and proceeds, so both callers observe the same
// quota rather than one of them failing.
func (r *UserDailyQuotaRepository) CreateIfAbsent(ctx context.Context, quota *entities.UserDailyQuota) (bool, error) {
	result := r.GetDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(quota)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *UserDailyQuotaRepository) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	var quota entities.UserDailyQuota
	err := r.GetDB(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&quota).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &quota, nil
}

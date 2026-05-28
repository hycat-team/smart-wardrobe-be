package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserDailyQuotaRepository implements database actions for daily user resource usage quotas
type UserDailyQuotaRepository struct {
	*shared_repos.GenericRepository[entities.UserDailyQuota, uuid.UUID]
}

// NewUserDailyQuotaRepository creates a new instance of quota repository
func NewUserDailyQuotaRepository(dbConn *gorm.DB) repositories.IUserDailyQuotaRepository {
	return &UserDailyQuotaRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.UserDailyQuota, uuid.UUID](dbConn),
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

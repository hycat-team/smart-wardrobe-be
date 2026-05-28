package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

// IUserDailyQuotaRepository handles database operations for user daily quotas
type IUserDailyQuotaRepository interface {
	repositories.IGenericRepository[entities.UserDailyQuota, uuid.UUID]
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error)
}

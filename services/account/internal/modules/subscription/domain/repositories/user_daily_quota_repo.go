package repositories

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

// IUserDailyQuotaRepository handles database operations for user daily quotas
type IUserDailyQuotaRepository interface {
	repositories.IGenericRepository[entities.UserDailyQuota, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error)
	GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error)
}

package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IUserRepository interface {
	repositories.IGenericRepository[entities.User, uuid.UUID]
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	IsEmailExists(ctx context.Context, email string) (bool, error)
	IsUsernameExists(ctx context.Context, username string) (bool, error)
	GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error)
	UpdateOutfitQuota(ctx context.Context, user *entities.User, count int, resetDate bool) error
	UpdateAiChatQuota(ctx context.Context, user *entities.User, count int, resetDate bool) error
	ResetDailyQuotas(ctx context.Context, user *entities.User) error
}

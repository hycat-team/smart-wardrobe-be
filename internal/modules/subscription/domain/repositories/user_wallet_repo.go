package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IUserWalletRepository interface {
	shared_repos.IGenericRepository[entities.UserWallet, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error)
}

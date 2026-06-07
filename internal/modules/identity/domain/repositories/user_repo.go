package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

// IUserRepository handles repository operations for basic user identities
type IUserRepository interface {
	repositories.IGenericRepository[entities.User, uuid.UUID]
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	IsEmailExists(ctx context.Context, email string) (bool, error)
	IsUsernameExists(ctx context.Context, username string) (bool, error)
	GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error)
}

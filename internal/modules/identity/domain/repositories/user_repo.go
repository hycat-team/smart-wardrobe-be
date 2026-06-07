package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type UserFilter struct {
	RoleSlug *string
	IsActive *bool
	Query    *string
	Page     int
	Limit    int
}

type UserListResult struct {
	Users      []*entities.User
	TotalCount int64
}

// IUserRepository handles repository operations for basic user identities
type IUserRepository interface {
	repositories.IGenericRepository[entities.User, uuid.UUID]
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	IsEmailExists(ctx context.Context, email string) (bool, error)
	IsUsernameExists(ctx context.Context, username string) (bool, error)
	GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error)
	GetUsersForAdmin(ctx context.Context, filter UserFilter) (*UserListResult, error)
}


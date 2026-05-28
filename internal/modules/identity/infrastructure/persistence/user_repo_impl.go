package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository implements identity repository actions using GORM
type UserRepository struct {
	shared_persist.GenericRepository[entities.User, uuid.UUID]
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *gorm.DB) repositories.IUserRepository {
	return &UserRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.User, uuid.UUID](db),
	}
}

// GetPreloadRelations returns zero preloaded relationships by default
func (r *UserRepository) GetPreloadRelations() []string {
	return []string{}
}

// FindByEmail searches for a user matching the provided email address
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.GetDB(ctx).Where("email = ? AND is_deleted = ?", email, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// IsEmailExists checks if email already exists in system
func (r *UserRepository) IsEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.User{}).Where("email = ? AND is_deleted = ?", email, false).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsUsernameExists checks if username already exists in system
func (r *UserRepository) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.User{}).Where("username = ? AND is_deleted = ?", username, false).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByUsernameOrEmail searches for a user by email or username
func (r *UserRepository) GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error) {
	var user entities.User
	err := r.GetDB(ctx).Where("(username = ? OR email = ?) AND is_deleted = ?", loginName, loginName, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

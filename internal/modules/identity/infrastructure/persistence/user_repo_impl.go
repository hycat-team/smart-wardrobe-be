package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	shared_persist.GenericRepository[entities.User, uuid.UUID]
}

func NewUserRepository(db *gorm.DB) repositories.IUserRepository {
	return &UserRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.User, uuid.UUID](db),
	}
}

func (r *UserRepository) GetPreloadRelations() []string {
	return []string{}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.GenericRepository.DB.WithContext(ctx).Where("email = ? AND is_deleted = ?", email, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) IsEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.GenericRepository.DB.WithContext(ctx).Model(&entities.User{}).Where("email = ? AND is_deleted = ?", email, false).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.GenericRepository.DB.WithContext(ctx).Model(&entities.User{}).Where("username = ? AND is_deleted = ?", username, false).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error) {
	var user entities.User
	err := r.GenericRepository.DB.WithContext(ctx).Where("(username = ? OR email = ?) AND is_deleted = ?", loginName, loginName, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateOutfitQuota(ctx context.Context, user *entities.User, count int, resetDate bool) error {
	if resetDate {
		now := time.Now()
		return r.GenericRepository.DB.WithContext(ctx).Model(user).Updates(map[string]any{
			"outfit_recommend_count": count,
			"last_reset_date":        now,
		}).Error
	}
	return r.GenericRepository.DB.WithContext(ctx).Model(user).Update("outfit_recommend_count", count).Error
}

func (r *UserRepository) UpdateAiChatQuota(ctx context.Context, user *entities.User, count int, resetDate bool) error {
	if resetDate {
		now := time.Now()
		return r.GenericRepository.DB.WithContext(ctx).Model(user).Updates(map[string]any{
			"ai_usage_count":  count,
			"last_reset_date": now,
		}).Error
	}
	return r.GenericRepository.DB.WithContext(ctx).Model(user).Update("ai_usage_count", count).Error
}

func (r *UserRepository) ResetDailyQuotas(ctx context.Context, user *entities.User) error {
	now := time.Now()
	return r.GenericRepository.DB.WithContext(ctx).Model(user).Updates(map[string]any{
		"outfit_recommend_count": 0,
		"ai_usage_count":         0,
		"last_reset_date":        now,
	}).Error
}

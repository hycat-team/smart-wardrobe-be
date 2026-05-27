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

type RefreshTokenRepository struct {
	shared_persist.GenericRepository[entities.RefreshToken, uuid.UUID]
}

func NewRefreshTokenRepository(db *gorm.DB) repositories.IRefreshTokenRepository {
	return &RefreshTokenRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.RefreshToken, uuid.UUID](db),
	}
}

func (r *RefreshTokenRepository) FindByToken(ctx context.Context, token string) (*entities.RefreshToken, error) {
	var rt entities.RefreshToken
	err := r.GenericRepository.DB.WithContext(ctx).Where("token = ? AND is_revoked = ?", token, false).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) RevokeToken(ctx context.Context, token string) error {
	return r.GenericRepository.DB.WithContext(ctx).Model(&entities.RefreshToken{}).
		Where("token = ?", token).
		Update("is_revoked", true).Error
}

func (r *RefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.GenericRepository.DB.WithContext(ctx).Model(&entities.RefreshToken{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Update("is_revoked", true).Error
}

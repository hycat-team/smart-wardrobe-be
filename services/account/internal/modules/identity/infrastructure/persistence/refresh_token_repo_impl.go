package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	shared_persist.GenericRepository[entities.RefreshToken, uuid.UUID]
}

func NewRefreshTokenRepository(db *gorm.DB) repositories.IRefreshTokenRepository {
	relations := []string{"User"}
	return &RefreshTokenRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.RefreshToken, uuid.UUID](db, relations),
	}
}

func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*entities.RefreshToken, error) {
	var rt entities.RefreshToken
	err := r.GetDB(ctx).Where("token = ? AND is_revoked = ?", token, false).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.RefreshToken, error) {
	var tokens []*entities.RefreshToken
	err := r.GetDB(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *RefreshTokenRepository) RevokeToken(ctx context.Context, token string) error {
	return r.GetDB(ctx).Model(&entities.RefreshToken{}).
		Where("token = ?", token).
		Update("is_revoked", true).Error
}

func (r *RefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.GetDB(ctx).Model(&entities.RefreshToken{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Update("is_revoked", true).Error
}

func (r *RefreshTokenRepository) UpdateLastActive(ctx context.Context, token string, lastActiveAt time.Time, deviceIP, userAgent *string) error {
	updates := map[string]any{
		"last_active_at": lastActiveAt,
	}
	if deviceIP != nil && *deviceIP != "" {
		updates["device_ip"] = *deviceIP
	}
	if userAgent != nil && *userAgent != "" {
		updates["user_agent"] = *userAgent
	}
	return r.GetDB(ctx).Model(&entities.RefreshToken{}).
		Where("token = ?", token).
		Updates(updates).Error
}

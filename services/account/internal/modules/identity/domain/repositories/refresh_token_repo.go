package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type IRefreshTokenRepository interface {
	shared_repos.IGenericRepository[entities.RefreshToken, uuid.UUID]
	GetByToken(ctx context.Context, token string) (*entities.RefreshToken, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.RefreshToken, error)
	RevokeToken(ctx context.Context, token string) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
	UpdateLastActive(ctx context.Context, token string, lastActiveAt time.Time, deviceIP, userAgent *string) error
}

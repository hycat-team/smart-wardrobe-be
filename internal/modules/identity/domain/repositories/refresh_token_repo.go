package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IRefreshTokenRepository interface {
	shared_repos.IGenericRepository[entities.RefreshToken, uuid.UUID]
	GetByToken(ctx context.Context, token string) (*entities.RefreshToken, error)
	RevokeToken(ctx context.Context, token string) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
}

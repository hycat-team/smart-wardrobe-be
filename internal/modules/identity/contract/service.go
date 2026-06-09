package contract

import (
	"context"
	"smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

// IUserContract exposes external identity lookups for other modules
type IUserContract interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error)
	GetByUsername(ctx context.Context, username string) (*dto.UserRes, error)
	GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error)
}

package contract

import (
	"context"
	"smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

type IUserContract interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error)
	GetByUsername(ctx context.Context, username string) (*dto.UserRes, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error)
	GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error)
}

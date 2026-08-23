package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

type IUserBriefContract interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefRes, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefRes, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefRes, error)
}

package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"

	"github.com/google/uuid"
)

type IUserBriefUseCase interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefContractDTO, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefContractDTO, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error)
	SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*dto.UserBriefContractDTO, error)
}

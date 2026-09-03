package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

type IUserBriefUseCase interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefContractDTO, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefContractDTO, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error)
}

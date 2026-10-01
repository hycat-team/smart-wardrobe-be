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
	// GetByUsernamesOrEmailsIncludingDeleted keeps soft-deleted rows so the admin
	// grant flow can report `user_soft_deleted` (024 T136).
	//
	// It is on the use case interface rather than the cross-service contract on
	// purpose: it is served in-process only, because a deleted account should not
	// travel between services. The plain method keeps excluding them (FR-049),
	// which is what commerce relies on.
	GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error)
	SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*dto.UserBriefContractDTO, error)
}

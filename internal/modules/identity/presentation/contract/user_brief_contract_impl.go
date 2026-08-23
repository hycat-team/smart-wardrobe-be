package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"

	"github.com/google/uuid"
)

type UserBriefContract struct {
	userBriefUC uc_interfaces.IUserBriefUseCase
}

func NewUserBriefContract(userBriefUC uc_interfaces.IUserBriefUseCase) IUserBriefContract {
	return &UserBriefContract{userBriefUC: userBriefUC}
}

func (c *UserBriefContract) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefRes, error) {
	return c.userBriefUC.GetByID(ctx, userID)
}

func (c *UserBriefContract) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefRes, error) {
	return c.userBriefUC.GetByIDs(ctx, userIDs)
}

func (c *UserBriefContract) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefRes, error) {
	return c.userBriefUC.GetByUsernamesOrEmails(ctx, identifiers)
}

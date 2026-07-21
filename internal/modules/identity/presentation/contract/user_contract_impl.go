package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"

	"github.com/google/uuid"
)

type UserContract struct {
	userUC uc_interfaces.IUserUseCase
}

func NewUserContract(userUC uc_interfaces.IUserUseCase) IUserContract {
	return &UserContract{userUC: userUC}
}

func (c *UserContract) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error) {
	return c.userUC.GetByID(ctx, userID)
}

func (c *UserContract) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error) {
	return c.userUC.GetByIDs(ctx, userIDs)
}

func (c *UserContract) GetByUsername(ctx context.Context, username string) (*dto.UserRes, error) {
	return c.userUC.GetByUsername(ctx, username)
}

func (c *UserContract) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error) {
	return c.userUC.GetByUsernamesOrEmails(ctx, identifiers)
}

func (c *UserContract) GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error) {
	return c.userUC.GetStyleProfile(ctx, userID)
}

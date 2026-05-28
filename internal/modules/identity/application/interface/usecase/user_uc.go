package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

type IUserUseCase interface {
	ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileReq) (*dto.UserRes, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error)
}

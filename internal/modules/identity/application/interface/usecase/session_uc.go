package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
)

type ISessionUseCase interface {
	Login(ctx context.Context, input dto.LoginReq) (*dto.TokenRes, error)
	RefreshToken(ctx context.Context, input dto.RefreshTokenReq) (*dto.TokenRes, error)
	Logout(ctx context.Context, input dto.LogoutReq) (bool, error)
}

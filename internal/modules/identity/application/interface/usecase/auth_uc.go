package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
)

type IAuthUseCase interface {
	Register(ctx context.Context, input dto.RegisterReq) (bool, error)
	ConfirmRegisterOtp(ctx context.Context, input dto.ConfirmRegisterOtpReq) (bool, error)
	Login(ctx context.Context, input dto.LoginReq) (*dto.TokenRes, error)
	RefreshToken(ctx context.Context, input dto.RefreshTokenReq) (*dto.TokenRes, error)
	Logout(ctx context.Context, input dto.LogoutReq) (bool, error)
	SendForgotPasswordOtp(ctx context.Context, input dto.SendForgotPasswordOtpReq) (bool, error)
	ConfirmForgotPasswordOtp(ctx context.Context, input dto.ConfirmForgotPasswordOtpReq) (string, error)
	ResetPassword(ctx context.Context, input dto.ResetPasswordReq, resetToken string) (bool, error)
}

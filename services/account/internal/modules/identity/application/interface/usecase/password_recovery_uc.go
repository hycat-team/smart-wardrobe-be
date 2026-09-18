package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/dto/identity"
)

type IPasswordRecoveryUseCase interface {
	SendForgotPasswordOtp(ctx context.Context, input dto.SendForgotPasswordOtpReq) (bool, error)
	ConfirmForgotPasswordOtp(ctx context.Context, input dto.ConfirmForgotPasswordOtpReq) (string, error)
	ResetPassword(ctx context.Context, input dto.ResetPasswordReq, resetToken string) (bool, error)
	ResendForgotPasswordOtp(ctx context.Context, input dto.ResendOtpReq) (bool, error)
}

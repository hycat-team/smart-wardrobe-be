package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/dto/identity"
)

type IRegisterUseCase interface {
	Register(ctx context.Context, input dto.RegisterReq) (bool, error)
	ConfirmRegisterOtp(ctx context.Context, input dto.ConfirmRegisterOtpReq) (bool, error)
	ResendRegisterOtp(ctx context.Context, input dto.ResendOtpReq) (bool, error)
}

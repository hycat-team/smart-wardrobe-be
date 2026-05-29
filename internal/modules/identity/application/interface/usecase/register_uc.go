package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
)

type IRegisterUseCase interface {
	Register(ctx context.Context, input dto.RegisterReq) (bool, error)
	ConfirmRegisterOtp(ctx context.Context, input dto.ConfirmRegisterOtpReq) (bool, error)
}

package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"

	"github.com/google/uuid"
)

type UserUseCase struct {
	userRepo             repositories.IUserRepository
	passwordHasher       security.IPasswordHasher
	refreshTokenRepo     repositories.IRefreshTokenRepository
	subscriptionContract subscription_contract.ISubscriptionModuleContract
}

func NewUserUseCase(
	userRepo repositories.IUserRepository,
	passwordHasher security.IPasswordHasher,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	subscriptionContract subscription_contract.ISubscriptionModuleContract,
) uc_interfaces.IUserUseCase {
	return &UserUseCase{
		userRepo:             userRepo,
		passwordHasher:       passwordHasher,
		refreshTokenRepo:     refreshTokenRepo,
		subscriptionContract: subscriptionContract,
	}
}

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, errorcode.NewUnauthorized("Người dùng không tồn tại.")
	}

	isValid := uc.passwordHasher.VerifyPassword(input.OldPassword, user.PasswordHash)
	if !isValid {
		return false, errorcode.NewBadRequest("Mật khẩu cũ không chính xác.")
	}

	newPasswordHash, err := uc.passwordHasher.HashPassword(input.NewPassword)
	if err != nil {
		return false, err
	}

	user.ChangePasswordHash(newPasswordHash)

	if input.LogoutAllDevices {
		err = uc.refreshTokenRepo.RevokeAllByUserID(ctx, userID)
		if err != nil {
			return false, err
		}
	}

	err = uc.userRepo.Update(ctx, user)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileReq) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin người dùng.")
	}

	dob, err := time.Parse(time.DateOnly, input.DateOfBirth)
	if err != nil {
		return nil, errorcode.NewBadRequest("Ngày sinh không hợp lệ. Vui lòng định dạng yyyy-mm-dd.")
	}

	genVal := gender.Unknown
	if input.Gender != nil {
		genVal = *input.Gender
	} else if user.Gender != nil {
		genVal = *user.Gender
	}

	user.UpdateProfile(input.FirstName, input.LastName, dob, genVal)
	user.ChangeAddress(input.Address)

	err = uc.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin người dùng.")
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

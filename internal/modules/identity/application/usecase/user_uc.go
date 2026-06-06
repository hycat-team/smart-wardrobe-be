package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"

	"github.com/google/uuid"
)

type UserUseCase struct {
	userRepo             repositories.IUserRepository
	passwordHasher       security.IPasswordHasher
	refreshTokenRepo     repositories.IRefreshTokenRepository
	subscriptionContract subscription_contract.IUserSubscriptionContract
	mediaService         media.IMediaService
	cfg                  *config.Config
}

func NewUserUseCase(
	userRepo repositories.IUserRepository,
	passwordHasher security.IPasswordHasher,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	subscriptionContract subscription_contract.IUserSubscriptionContract,
	mediaService media.IMediaService,
	cfg *config.Config,
) uc_interfaces.IUserUseCase {
	return &UserUseCase{
		userRepo:             userRepo,
		passwordHasher:       passwordHasher,
		refreshTokenRepo:     refreshTokenRepo,
		subscriptionContract: subscriptionContract,
		mediaService:         mediaService,
		cfg:                  cfg,
	}
}

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, apperror.NewUnauthorized("Người dùng không tồn tại.")
	}

	isValid := uc.passwordHasher.VerifyPassword(input.OldPassword, user.PasswordHash)
	if !isValid {
		return false, apperror.NewBadRequest("Mật khẩu cũ không chính xác.")
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
		return nil, apperror.NewNotFound("Không tìm thấy thông tin người dùng.")
	}

	dob, err := time.Parse(time.DateOnly, input.DateOfBirth)
	if err != nil {
		return nil, apperror.NewBadRequest("Ngày sinh không hợp lệ. Vui lòng định dạng yyyy-mm-dd.")
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
		return nil, apperror.NewNotFound("Không tìm thấy thông tin người dùng.")
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetAvatarSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	params := shared_dto.UploadSignatureParams{
		Folder:   uc.cfg.Cloudinary.AvatarFolder,
		PublicID: userID.String(),
	}
	return uc.mediaService.GenerateUploadSignature(ctx, params)
}

func (uc *UserUseCase) UpdateAvatar(ctx context.Context, userID uuid.UUID, input dto.UpdateAvatarReq) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, apperror.NewNotFound("Không tìm thấy thông tin người dùng.")
	}

	user.UpdateAvatar(input.AvatarUrl, input.AvatarPublicID)
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

func (uc *UserUseCase) UpdateUserStatus(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, input dto.UpdateUserStatusReq) (*dto.UserRes, error) {
	if adminUserID == targetUserID {
		return nil, apperror.NewForbidden("Bạn không thể tự thay đổi trạng thái tài khoản admin của chính mình.")
	}

	adminUser, err := uc.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if adminUser == nil || adminUser.IsDeleted {
		return nil, apperror.NewUnauthorized("Không tìm thấy thông tin tài khoản admin.")
	}

	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil || targetUser.IsDeleted {
		return nil, apperror.NewNotFound("Không tìm thấy thông tin người dùng.")
	}
	if targetUser.RoleSlug != roleslug.Member {
		return nil, apperror.NewForbidden("Chỉ có thể thay đổi trạng thái tài khoản member.")
	}

	targetUser.Status = input.Status
	if err := uc.userRepo.Update(ctx, targetUser); err != nil {
		return nil, err
	}

	if input.Status == userstatus.Inactive {
		if err := uc.refreshTokenRepo.RevokeAllByUserID(ctx, targetUserID); err != nil {
			return nil, err
		}
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, targetUserID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(targetUser, sub), nil
}

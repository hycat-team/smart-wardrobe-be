package usecase

import (
	"context"
	"encoding/json"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/communication"
	"smart-wardrobe-be/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/vo"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/jwttype"
	"smart-wardrobe-be/internal/shared/domain/constants/otpconstants"
	"smart-wardrobe-be/pkg/utils/jwtutils"

	"github.com/google/uuid"
)

type PasswordRecoveryUseCase struct {
	userRepo         repositories.IUserRepository
	refreshTokenRepo repositories.IRefreshTokenRepository
	otpService       identity.IOtpService
	emailService     communication.IEmailService
	passwordHasher   security.IPasswordHasher
	cfg              *config.Config
}

func NewPasswordRecoveryUseCase(
	userRepo repositories.IUserRepository,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	otpService identity.IOtpService,
	emailService communication.IEmailService,
	passwordHasher security.IPasswordHasher,
	cfg *config.Config,
) uc_interfaces.IPasswordRecoveryUseCase {
	return &PasswordRecoveryUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		otpService:       otpService,
		emailService:     emailService,
		passwordHasher:   passwordHasher,
		cfg:              cfg,
	}
}

func (uc *PasswordRecoveryUseCase) SendForgotPasswordOtp(ctx context.Context, input dto.SendForgotPasswordOtpReq) (bool, error) {
	user, err := uc.userRepo.GetByUsernameOrEmail(ctx, input.Email)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errorcode.NewNotFound("Email này chưa được đăng ký trong hệ thống.")
	}

	isCooldown, err := uc.otpService.IsInResendCooldown(ctx, input.Email, otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}
	if isCooldown {
		return false, errorcode.NewBadRequest("Vui lòng đợi một lát trước khi yêu cầu mã mới.")
	}

	tempData := vo.TempOtpData{
		UserId: user.ID.String(),
	}

	tempUserDataJson, err := json.Marshal(tempData)
	if err != nil {
		return false, errorcode.NewInternalError("Lỗi khi chuyển đổi thông tin tạm thời")
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, string(tempUserDataJson), otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendForgotPasswordOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, errorcode.NewInternalError("Lỗi khi gửi email khôi phục mật khẩu")
	}

	return true, nil
}

func (uc *PasswordRecoveryUseCase) ConfirmForgotPasswordOtp(ctx context.Context, input dto.ConfirmForgotPasswordOtpReq) (string, error) {
	tempUserDataJson, err := uc.otpService.VerifyOtp(ctx, input.Email, input.OtpCode, otpconstants.PurposeForgotPassword)
	if err != nil {
		return "", err
	}

	if len(tempUserDataJson) == 0 {
		return "", errorcode.NewBadRequest("Dữ liệu xác thực không hợp lệ")
	}

	var tempData vo.TempOtpData
	err = json.Unmarshal([]byte(tempUserDataJson), &tempData)
	if err != nil {
		return "", errorcode.NewBadRequest("Dữ liệu xác thực không hợp lệ.")
	}

	userId, err := uuid.Parse(tempData.UserId)
	if err != nil {
		return "", errorcode.NewBadRequest("Dữ liệu xác thực không hợp lệ.")
	}

	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return "", err
	}
	if user == nil || user.IsDeleted {
		return "", errorcode.NewNotFound("Người dùng không tồn tại.")
	}

	resetToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, string(user.RoleSlug),
		jwttype.ResetPasswordToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		time.Duration(uc.cfg.Jwt.ForgotPasswordExpirationMinutes)*time.Minute,
	)
	if err != nil {
		return "", errorcode.NewInternalError("Lỗi khi cấp mã khôi phục mật khẩu")
	}

	return resetToken, nil
}

func (uc *PasswordRecoveryUseCase) ResetPassword(ctx context.Context, input dto.ResetPasswordReq, resetToken string) (bool, error) {
	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), resetToken, jwttype.ResetPasswordToken)
	if err != nil {
		return false, errorcode.NewUnauthorized("Token không hợp lệ.")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return false, errorcode.NewUnauthorized("Token không hợp lệ.")
	}

	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, errorcode.NewUnauthorized("Người dùng không tồn tại.")
	}

	newPasswordHash, err := uc.passwordHasher.HashPassword(input.NewPassword)
	if err != nil {
		return false, err
	}

	user.ChangePasswordHash(newPasswordHash)

	if input.LogoutAllDevices {
		err = uc.refreshTokenRepo.RevokeAllByUserID(ctx, userId)
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

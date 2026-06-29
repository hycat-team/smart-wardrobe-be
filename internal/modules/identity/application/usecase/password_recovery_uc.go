package usecase

import (
	"context"
	"encoding/json"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	"smart-wardrobe-be/internal/modules/identity/application/interface/communication"
	"smart-wardrobe-be/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/vo"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/jwttype"
	"smart-wardrobe-be/internal/shared/application/constants/otpconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/jwtutils"

	"github.com/google/uuid"
)

const forgotPasswordBlacklistPrefix = "blacklist:forgot_password:"

type PasswordRecoveryUseCase struct {
	userRepo              repositories.IUserRepository
	refreshTokenRepo      repositories.IRefreshTokenRepository
	otpService            identity.IOtpService
	emailService          communication.IEmailService
	passwordHasher        security.IPasswordHasher
	tokenBlacklistService security.ITokenBlacklistService
	cfg                   *config.Config
	uow                   shared_repos.IUnitOfWork
}

func NewPasswordRecoveryUseCase(
	userRepo repositories.IUserRepository,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	otpService identity.IOtpService,
	emailService communication.IEmailService,
	passwordHasher security.IPasswordHasher,
	tokenBlacklistService security.ITokenBlacklistService,
	cfg *config.Config,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IPasswordRecoveryUseCase {
	return &PasswordRecoveryUseCase{
		userRepo:              userRepo,
		refreshTokenRepo:      refreshTokenRepo,
		otpService:            otpService,
		emailService:          emailService,
		passwordHasher:        passwordHasher,
		tokenBlacklistService: tokenBlacklistService,
		cfg:                   cfg,
		uow:                   uow,
	}
}

func (uc *PasswordRecoveryUseCase) SendForgotPasswordOtp(ctx context.Context, input dto.SendForgotPasswordOtpReq) (bool, error) {
	user, err := uc.userRepo.GetByUsernameOrEmail(ctx, input.Email)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted || user.Status != userstatus.Active {
		return true, nil
	}

	isCooldown, err := uc.otpService.IsInResendCooldown(ctx, input.Email, otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}
	if isCooldown {
		return false, identityerrors.ErrOtpCooldown()
	}

	tempData := vo.TempOtpData{
		UserId: user.ID.String(),
	}

	tempUserDataJSON, err := json.Marshal(tempData)
	if err != nil {
		return false, identityerrors.ErrTempConvertFailed()
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, string(tempUserDataJSON), otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}

	if err := uc.emailService.SendForgotPasswordOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes); err != nil {
		return false, identityerrors.ErrRecoveryEmailFailed()
	}

	return true, nil
}

func (uc *PasswordRecoveryUseCase) ConfirmForgotPasswordOtp(ctx context.Context, input dto.ConfirmForgotPasswordOtpReq) (string, error) {
	tempUserDataJSON, err := uc.otpService.VerifyOtp(ctx, input.Email, input.OtpCode, otpconstants.PurposeForgotPassword)
	if err != nil {
		return "", err
	}

	if len(tempUserDataJSON) == 0 {
		return "", identityerrors.ErrOtpValidationInvalid()
	}

	var tempData vo.TempOtpData
	if err := json.Unmarshal([]byte(tempUserDataJSON), &tempData); err != nil {
		return "", identityerrors.ErrOtpValidationInvalid()
	}

	userID, err := uuid.Parse(tempData.UserId)
	if err != nil {
		return "", identityerrors.ErrOtpValidationInvalid()
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if err := ensureUserEligibleForRecovery(user); err != nil {
		return "", err
	}

	resetToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, string(user.RoleSlug),
		jwttype.ResetPasswordToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		time.Duration(uc.cfg.Jwt.ForgotPasswordExpirationMinutes)*time.Minute,
	)
	if err != nil {
		return "", identityerrors.ErrResetTokenGenFailed()
	}

	return resetToken, nil
}

func (uc *PasswordRecoveryUseCase) ResetPassword(ctx context.Context, input dto.ResetPasswordReq, resetToken string) (bool, error) {
	isBlacklisted, err := uc.tokenBlacklistService.IsTokenBlacklistedWithPrefix(ctx, resetToken, forgotPasswordBlacklistPrefix)
	if err != nil {
		return false, err
	}
	if isBlacklisted {
		return false, identityerrors.ErrInvalidToken()
	}

	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), resetToken, jwttype.ResetPasswordToken)
	if err != nil {
		return false, identityerrors.ErrInvalidToken()
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return false, identityerrors.ErrInvalidToken()
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, identityerrors.ErrUserNotFound()
	}

	newPasswordHash, err := uc.passwordHasher.HashPassword(input.NewPassword)
	if err != nil {
		return false, err
	}

	user.ChangePasswordHash(newPasswordHash)

	changePassword := func(txCtx context.Context) error {
		if input.LogoutAllDevices {
			if err := uc.refreshTokenRepo.RevokeAllByUserID(txCtx, userID); err != nil {
				return err
			}
		}

		if err := uc.userRepo.Update(txCtx, user); err != nil {
			return err
		}
		return nil
	}

	if err := uc.uow.Execute(ctx, changePassword); err != nil {
		return false, err
	}

	// Blacklist the reset token for its remaining duration
	if claims.ExpiresAt != nil {
		expTime := claims.ExpiresAt.Time
		remainingTime := time.Until(expTime)
		if remainingTime > 0 {
			_ = uc.tokenBlacklistService.BlacklistTokenWithPrefix(ctx, resetToken, forgotPasswordBlacklistPrefix, remainingTime)
		}
	}

	return true, nil
}

func (uc *PasswordRecoveryUseCase) ResendForgotPasswordOtp(ctx context.Context, input dto.ResendOtpReq) (bool, error) {
	tempUserDataJson, err := uc.otpService.GetData(ctx, input.Email, otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}
	if tempUserDataJson == "" {
		return false, identityerrors.ErrForgotPasswordSessionExpired()
	}

	isCooldown, err := uc.otpService.IsInResendCooldown(ctx, input.Email, otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}
	if isCooldown {
		return false, identityerrors.ErrOtpCooldown()
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, tempUserDataJson, otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendForgotPasswordOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, identityerrors.ErrRecoveryEmailFailed()
	}

	return true, nil
}

var _ uc_interfaces.IPasswordRecoveryUseCase = (*PasswordRecoveryUseCase)(nil)

package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	billing_contract "smart-wardrobe-be/internal/modules/billing/contract"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/communication"
	"smart-wardrobe-be/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	"smart-wardrobe-be/internal/modules/identity/application/vo"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/jwttype"
	"smart-wardrobe-be/internal/shared/domain/constants/otpconstants"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/jwtutils"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepo              repositories.IUserRepository
	refreshTokenRepo      repositories.IRefreshTokenRepository
	otpService            identity.IOtpService
	emailService          communication.IEmailService
	passwordHasher        security.IPasswordHasher
	tokenBlacklistService security.ITokenBlacklistService
	billingContract       billing_contract.IBillingModuleContract
	cfg                   *config.Config
}

func NewAuthUseCase(
	userRepo repositories.IUserRepository,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	otpService identity.IOtpService,
	emailService communication.IEmailService,
	passwordHasher security.IPasswordHasher,
	tokenBlacklistService security.ITokenBlacklistService,
	billingContract billing_contract.IBillingModuleContract,
	cfg *config.Config,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:              userRepo,
		refreshTokenRepo:      refreshTokenRepo,
		otpService:            otpService,
		emailService:          emailService,
		passwordHasher:        passwordHasher,
		tokenBlacklistService: tokenBlacklistService,
		billingContract:       billingContract,
		cfg:                   cfg,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, input dto.RegisterReq) (bool, error) {
	usernameExists, err := uc.userRepo.IsUsernameExists(ctx, input.Username)
	if err != nil {
		return false, err
	}
	if usernameExists {
		return false, errorcode.NewConflict(fmt.Sprintf("Username '%s' đã tồn tại.", input.Username))
	}

	emailExists, err := uc.userRepo.IsEmailExists(ctx, input.Email)
	if err != nil {
		return false, err
	}
	if emailExists {
		return false, errorcode.NewConflict(fmt.Sprintf("Email '%s' đã tồn tại.", input.Email))
	}

	isCooldown, err := uc.otpService.IsInResendCooldown(ctx, input.Email, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}
	if isCooldown {
		return false, errorcode.NewTooManyRequest("Vui lòng đợi 1 phút trước khi yêu cầu OTP mới.")
	}

	hashedPass, err := uc.passwordHasher.HashPassword(input.Password)
	if err != nil {
		return false, err
	}

	var genVal gender.Gender
	if input.Gender != nil {
		genVal = *input.Gender
	}

	cacheModel := vo.TempUserCacheModel{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: hashedPass,
		FirstName:    input.FirstName,
		LastName:     stringutils.GetString(input.LastName),
		Address:      input.Address,
		DateOfBirth:  input.DateOfBirth,
		Gender:       genVal,
	}

	tempUserDataJson, err := json.Marshal(cacheModel)
	if err != nil {
		return false, err
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, string(tempUserDataJson), otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendRegistrationOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (uc *AuthUseCase) ConfirmRegisterOtp(ctx context.Context, input dto.ConfirmRegisterOtpReq) (bool, error) {
	tempUserDataJson, err := uc.otpService.VerifyOtp(ctx, input.Email, input.OtpCode, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	if len(tempUserDataJson) == 0 {
		return false, errors.New("lấy thông tin đăng kí thất bại")
	}

	var registerData vo.TempUserCacheModel
	err = json.Unmarshal([]byte(tempUserDataJson), &registerData)
	if err != nil {
		return false, errorcode.NewBadRequest("Thông tin đăng kí không hợp lệ.")
	}

	usernameExists, err := uc.userRepo.IsUsernameExists(ctx, registerData.Username)
	if err != nil {
		return false, err
	}
	if usernameExists {
		return false, errorcode.NewConflict(fmt.Sprintf("Username '%s' đã tồn tại.", registerData.Username))
	}

	defaultPlanID, err := uc.billingContract.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return false, err
	}

	dob, err := time.Parse("2006-01-02", registerData.DateOfBirth)
	if err != nil {
		return false, errorcode.NewBadRequest("Ngày sinh không hợp lệ. Vui lòng định dạng yyyy-mm-dd.")
	}

	gen := gender.Gender(registerData.Gender)

	newUser := &entities.User{
		Username:           registerData.Username,
		Email:              registerData.Email,
		PasswordHash:       registerData.PasswordHash,
		FirstName:          &registerData.FirstName,
		LastName:           &registerData.LastName,
		DateOfBirth:        &dob,
		Address:            &registerData.Address,
		Gender:             &gen,
		RoleSlug:           "member",
		SubscriptionPlanID: defaultPlanID,
		LastResetDate:      time.Now(),
		Status:             1,
	}
	newUser.ID = uuid.New()
	newUser.IsDeleted = false

	err = uc.userRepo.Create(ctx, newUser)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, input dto.LoginReq) (*dto.TokenRes, error) {
	user, err := uc.userRepo.GetByUsernameOrEmail(ctx, input.LoginName)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errorcode.NewBadRequest("Sai username/email hoặc mật khẩu.")
	}

	isValid := uc.passwordHasher.VerifyPassword(input.Password, user.PasswordHash)
	if !isValid {
		return nil, errorcode.NewBadRequest("Sai username/email hoặc mật khẩu.")
	}

	accessExpiry := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)
	refreshExpiry := time.Hour * 24 * time.Duration(uc.cfg.Jwt.RefreshExpirationDays)

	accessToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, user.RoleSlug,
		jwttype.AccessToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		accessExpiry,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, user.RoleSlug,
		jwttype.RefreshToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		refreshExpiry,
	)
	if err != nil {
		return nil, err
	}

	rt := &entities.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(refreshExpiry),
		IsRevoked: false,
	}
	rt.ID = uuid.New()

	err = uc.refreshTokenRepo.Create(ctx, rt)
	if err != nil {
		return nil, err
	}

	return &dto.TokenRes{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, input dto.RefreshTokenReq) (*dto.TokenRes, error) {
	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), input.OldRefreshToken, jwttype.RefreshToken)
	if err != nil {
		return nil, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	}

	oldExpiresAtUtc := claims.ExpiresAt.Time
	remainingTime := time.Until(oldExpiresAtUtc)
	if remainingTime <= 0 {
		return nil, errorcode.NewUnauthorized("Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	}

	user, err := uc.userRepo.FindByID(ctx, userId)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, errorcode.NewUnauthorized("Không tìm thấy người dùng này.")
	}

	existingToken, err := uc.refreshTokenRepo.FindByToken(ctx, input.OldRefreshToken)
	if err != nil {
		return nil, err
	}
	if existingToken == nil || existingToken.IsRevoked {
		return nil, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	}

	accessExpiry := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)

	newAccessToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, user.RoleSlug,
		jwttype.AccessToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		accessExpiry,
	)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, user.RoleSlug,
		jwttype.RefreshToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		remainingTime,
	)
	if err != nil {
		return nil, err
	}

	err = uc.refreshTokenRepo.RevokeToken(ctx, input.OldRefreshToken)
	if err != nil {
		return nil, err
	}

	rt := &entities.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshToken,
		ExpiresAt: time.Now().Add(remainingTime),
		IsRevoked: false,
	}
	rt.ID = uuid.New()

	err = uc.refreshTokenRepo.Create(ctx, rt)
	if err != nil {
		return nil, err
	}

	return &dto.TokenRes{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, input dto.LogoutReq) (bool, error) {
	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), input.RefreshToken, jwttype.RefreshToken)
	if err != nil {
		return false, errorcode.NewUnauthorized("Invalid token payload.")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return false, errorcode.NewUnauthorized("Invalid token payload.")
	}

	user, err := uc.userRepo.FindByID(ctx, userId)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, errorcode.NewUnauthorized("Không tìm thấy người dùng.")
	}

	err = uc.refreshTokenRepo.RevokeToken(ctx, input.RefreshToken)
	if err != nil {
		return false, err
	}

	// Calculate remaining access token lifetime
	tokenHandler := &jwt.Parser{}
	tokenClaims := &jwtutils.CustomClaims{}
	_, _, err = tokenHandler.ParseUnverified(input.AccessToken, tokenClaims)
	if err == nil && tokenClaims.ExpiresAt != nil {
		expTime := tokenClaims.ExpiresAt.Time
		remainingTime := time.Until(expTime)
		if remainingTime > 0 {
			_ = uc.tokenBlacklistService.BlacklistToken(ctx, input.AccessToken, remainingTime)
		}
	}

	return true, nil
}

func (uc *AuthUseCase) SendForgotPasswordOtp(ctx context.Context, input dto.SendForgotPasswordOtpReq) (bool, error) {
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
		return false, err
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, string(tempUserDataJson), otpconstants.PurposeForgotPassword)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendForgotPasswordOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (uc *AuthUseCase) ConfirmForgotPasswordOtp(ctx context.Context, input dto.ConfirmForgotPasswordOtpReq) (string, error) {
	tempUserDataJson, err := uc.otpService.VerifyOtp(ctx, input.Email, input.OtpCode, otpconstants.PurposeForgotPassword)
	if err != nil {
		return "", err
	}

	if len(tempUserDataJson) == 0 {
		return "", errors.New("dữ liệu xác thực không hợp lệ")
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

	user, err := uc.userRepo.FindByID(ctx, userId)
	if err != nil {
		return "", err
	}
	if user == nil || user.IsDeleted {
		return "", errorcode.NewUnauthorized("Người dùng không tồn tại.")
	}

	resetToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, user.RoleSlug,
		jwttype.ResetPasswordToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		time.Duration(uc.cfg.Jwt.ForgotPasswordExpirationMinutes)*time.Minute,
	)
	if err != nil {
		return "", err
	}

	return resetToken, nil
}

func (uc *AuthUseCase) ResetPassword(ctx context.Context, input dto.ResetPasswordReq, resetToken string) (bool, error) {
	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), resetToken, jwttype.ResetPasswordToken)
	if err != nil {
		return false, errorcode.NewUnauthorized("Token không hợp lệ.")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return false, errorcode.NewUnauthorized("Token không hợp lệ.")
	}

	user, err := uc.userRepo.FindByID(ctx, userId)
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

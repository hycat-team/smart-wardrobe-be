package usecase

import (
	"context"
	"encoding/json"
	"strings"
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
	"smart-wardrobe-be/internal/shared/application/constants/otpconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
)

type RegisterUseCase struct {
	userRepo       repositories.IUserRepository
	otpService     identity.IOtpService
	emailService   communication.IEmailService
	passwordHasher security.IPasswordHasher
	cfg            *config.Config
}

func NewRegisterUseCase(
	userRepo repositories.IUserRepository,
	otpService identity.IOtpService,
	emailService communication.IEmailService,
	passwordHasher security.IPasswordHasher,
	cfg *config.Config,
) uc_interfaces.IRegisterUseCase {
	return &RegisterUseCase{
		userRepo:       userRepo,
		otpService:     otpService,
		emailService:   emailService,
		passwordHasher: passwordHasher,
		cfg:            cfg,
	}
}

func (uc *RegisterUseCase) Register(ctx context.Context, input dto.RegisterReq) (bool, error) {
	if strings.Contains(strings.ToLower(input.Password), strings.ToLower(input.Username)) {
		return false, identityerrors.ErrPasswordContainsUsername()
	}

	usernameExists, err := uc.userRepo.IsUsernameExists(ctx, input.Username)
	if err != nil {
		return false, err
	}
	if usernameExists {
		return false, identityerrors.ErrUsernameExists(input.Username)
	}

	emailExists, err := uc.userRepo.IsEmailExists(ctx, input.Email)
	if err != nil {
		return false, err
	}
	if emailExists {
		return false, identityerrors.ErrEmailExists(input.Email)
	}

	isCooldown, err := uc.otpService.IsInResendCooldown(ctx, input.Email, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}
	if isCooldown {
		return false, identityerrors.ErrOtpCooldown()
	}

	hashedPass, err := uc.passwordHasher.HashPassword(input.Password)
	if err != nil {
		return false, err
	}

	if input.DateOfBirth != "" {
		_, err := time.Parse(time.DateOnly, input.DateOfBirth)
		if err != nil {
			return false, identityerrors.ErrInvalidDob()
		}
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
		Address:      stringutils.GetString(input.Address),
		DateOfBirth:  input.DateOfBirth,
		Gender:       genVal,
	}

	tempUserDataJson, err := json.Marshal(cacheModel)
	if err != nil {
		return false, identityerrors.ErrJsonConvertFailed()
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, string(tempUserDataJson), otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendRegistrationOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, identityerrors.ErrOtpEmailSendFailed()
	}

	return true, nil
}

func (uc *RegisterUseCase) ConfirmRegisterOtp(ctx context.Context, input dto.ConfirmRegisterOtpReq) (bool, error) {
	tempUserDataJson, err := uc.otpService.VerifyOtp(ctx, input.Email, input.OtpCode, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	if len(tempUserDataJson) == 0 {
		return false, identityerrors.ErrOtpVerificationFail()
	}

	var registerData vo.TempUserCacheModel
	err = json.Unmarshal([]byte(tempUserDataJson), &registerData)
	if err != nil {
		return false, identityerrors.ErrOtpDataInvalid()
	}

	dob, err := time.Parse(time.DateOnly, registerData.DateOfBirth)
	if err != nil {
		return false, identityerrors.ErrInvalidDob()
	}

	gen := gender.Gender(registerData.Gender)

	newUser := &entities.User{
		Username:     registerData.Username,
		Email:        registerData.Email,
		PasswordHash: registerData.PasswordHash,
		FirstName:    &registerData.FirstName,
		LastName:     &registerData.LastName,
		DateOfBirth:  &dob,
		Address:      &registerData.Address,
		Gender:       &gen,
		RoleSlug:     roleslug.User,
		Status:       userstatus.Active,
	}
	newUser.ID = uuid.New()
	newUser.IsDeleted = false

	err = uc.userRepo.Create(ctx, newUser)
	if err != nil {
		return false, identityerrors.ErrAccountCreationFailed()
	}

	return true, nil
}

func (uc *RegisterUseCase) ResendRegisterOtp(ctx context.Context, input dto.ResendOtpReq) (bool, error) {
	tempUserDataJson, err := uc.otpService.GetData(ctx, input.Email, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}
	if tempUserDataJson == "" {
		return false, identityerrors.ErrRegistrationSessionExpired()
	}

	isCooldown, err := uc.otpService.IsInResendCooldown(ctx, input.Email, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}
	if isCooldown {
		return false, identityerrors.ErrOtpCooldown()
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, tempUserDataJson, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendRegistrationOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, identityerrors.ErrOtpEmailSendFailed()
	}

	return true, nil
}

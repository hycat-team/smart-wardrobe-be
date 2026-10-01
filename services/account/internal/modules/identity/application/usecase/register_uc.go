package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/dto/identity"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/vo"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/shared/application/constants/otpconstants"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/utils/stringutils"

	"github.com/google/uuid"
)

type RegisterUseCase struct {
	userRepo       repositories.IUserRepository
	otpService     identity.IOtpService
	notifContract  notificationContract.INotificationContract
	passwordHasher security.IPasswordHasher
	cfg            *config.Config
	// 024 T196: see CreateUserByAdmin. The email sign-up door had the same gap
	// as the other two, and it is the busiest one.
	provisioningContract subscription_contract.IUserSubscriptionProvisioningContract
	uow                  shared_repos.IUnitOfWork
}

func NewRegisterUseCase(
	userRepo repositories.IUserRepository,
	otpService identity.IOtpService,
	notifContract notificationContract.INotificationContract,
	passwordHasher security.IPasswordHasher,
	cfg *config.Config,
	provisioningContract subscription_contract.IUserSubscriptionProvisioningContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IRegisterUseCase {
	return &RegisterUseCase{
		userRepo:             userRepo,
		otpService:           otpService,
		notifContract:        notifContract,
		passwordHasher:       passwordHasher,
		cfg:                  cfg,
		provisioningContract: provisioningContract,
		uow:                  uow,
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

	err = uc.notifContract.Send(ctx, notificationContract.ChannelEmail, notificationContract.TemplateOtp, map[string]any{
		"OTPCode":       otpCode,
		"ExpiryMinutes": uc.cfg.Otp.ExpiryMinutes,
	}, input.Email)
	if err != nil {
		sendErr := identityerrors.ErrOtpEmailSendFailed()
		sendErr.Cause = err
		return false, sendErr
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
		Username:      registerData.Username,
		Email:         registerData.Email,
		PasswordHash:  &registerData.PasswordHash,
		Provider:      authprovider.Local,
		EmailVerified: true,
		FirstName:     &registerData.FirstName,
		LastName:      &registerData.LastName,
		DateOfBirth:   &dob,
		Address:       &registerData.Address,
		Gender:        &gen,
		RoleSlug:      roleslug.User,
		Status:        userstatus.Active,
	}
	newUser.ID = uuid.New()
	newUser.IsDeleted = false

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Create(txCtx, newUser); err != nil {
			return err
		}
		return uc.provisioningContract.EnsureDefaultSubscription(txCtx, newUser.ID)
	})
	if err != nil {
		// A concurrent Google login may have created the same email first.
		if emailExists, checkErr := uc.userRepo.IsEmailExists(ctx, registerData.Email); checkErr == nil && emailExists {
			return false, identityerrors.ErrEmailExists(registerData.Email)
		}
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

	err = uc.notifContract.Send(ctx, notificationContract.ChannelEmail, notificationContract.TemplateOtp, map[string]any{
		"OTPCode":       otpCode,
		"ExpiryMinutes": uc.cfg.Otp.ExpiryMinutes,
	}, input.Email)
	if err != nil {
		sendErr := identityerrors.ErrOtpEmailSendFailed()
		sendErr.Cause = err
		return false, sendErr
	}

	return true, nil
}

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
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
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/otpconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
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
	usernameExists, err := uc.userRepo.IsUsernameExists(ctx, input.Username)
	if err != nil {
		return false, err
	}
	if usernameExists {
		return false, errorcode.NewConflict(fmt.Sprintf("Tài khoản '%s' đã tồn tại.", input.Username))
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

	if input.DateOfBirth != "" {
		_, err := time.Parse(time.DateOnly, input.DateOfBirth)
		if err != nil {
			return false, errorcode.NewBadRequest("Ngày sinh không hợp lệ. Vui lòng định dạng yyyy-mm-dd.")
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
		Address:      input.Address,
		DateOfBirth:  input.DateOfBirth,
		Gender:       genVal,
	}

	tempUserDataJson, err := json.Marshal(cacheModel)
	if err != nil {
		return false, errorcode.NewInternalError("Lỗi khi chuyển đổi thông tin người dùng")
	}

	otpCode, err := uc.otpService.GenerateOtp(ctx, input.Email, string(tempUserDataJson), otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	err = uc.emailService.SendRegistrationOtpEmail(ctx, input.Email, otpCode, uc.cfg.Otp.ExpiryMinutes)
	if err != nil {
		return false, errorcode.NewInternalError("Lỗi khi gửi email xác nhận OTP")
	}

	return true, nil
}

func (uc *RegisterUseCase) ConfirmRegisterOtp(ctx context.Context, input dto.ConfirmRegisterOtpReq) (bool, error) {
	tempUserDataJson, err := uc.otpService.VerifyOtp(ctx, input.Email, input.OtpCode, otpconstants.PurposeRegistration)
	if err != nil {
		return false, err
	}

	if len(tempUserDataJson) == 0 {
		return false, errorcode.NewInternalError("Lấy thông tin đăng ký thất bại")
	}

	var registerData vo.TempUserCacheModel
	err = json.Unmarshal([]byte(tempUserDataJson), &registerData)
	if err != nil {
		return false, errorcode.NewInternalError("Thông tin đăng ký không hợp lệ.")
	}

	dob, err := time.Parse(time.DateOnly, registerData.DateOfBirth)
	if err != nil {
		return false, errorcode.NewInternalError("Ngày sinh không hợp lệ.")
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
		RoleSlug:     roleslug.Member,
		Status:       1,
	}
	newUser.ID = uuid.New()
	newUser.IsDeleted = false

	err = uc.userRepo.Create(ctx, newUser)
	if err != nil {
		return false, errorcode.NewInternalError("Lỗi khi khởi tạo tài khoản mới")
	}

	return true, nil
}

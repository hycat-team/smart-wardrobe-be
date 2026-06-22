package usecase

import (
	"context"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/communication"
	"smart-wardrobe-be/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
)

type mockOtpService struct {
	identity.IOtpService
	data           string
	cooldown       bool
	generatedOtp   string
	generatedEmail string
}

func (m *mockOtpService) GetData(ctx context.Context, email string, purpose string) (string, error) {
	return m.data, nil
}

func (m *mockOtpService) IsInResendCooldown(ctx context.Context, email string, purpose string) (bool, error) {
	return m.cooldown, nil
}

func (m *mockOtpService) GenerateOtp(ctx context.Context, email string, tempUserDataJson string, purpose string) (string, error) {
	m.generatedEmail = email
	m.generatedOtp = "123456"
	return m.generatedOtp, nil
}

type mockEmailService struct {
	communication.IEmailService
	sentEmail string
	sentOtp   string
}

func (m *mockEmailService) SendRegistrationOtpEmail(ctx context.Context, email string, otp string, expiry int) error {
	m.sentEmail = email
	m.sentOtp = otp
	return nil
}

func TestResendRegisterOtp_Success(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     `{"username": "testuser"}`,
		cooldown: false,
	}
	emailSvc := &mockEmailService{}

	uc := &RegisterUseCase{
		otpService:   otpSvc,
		emailService: emailSvc,
		cfg: &config.Config{
			Otp: config.Otp{
				ExpiryMinutes: 5,
			},
		},
	}

	ok, err := uc.ResendRegisterOtp(context.Background(), dto.ResendOtpReq{
		Email: "test@example.com",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok to be true")
	}
	if otpSvc.generatedEmail != "test@example.com" {
		t.Errorf("expected generated email to be test@example.com, got %s", otpSvc.generatedEmail)
	}
	if emailSvc.sentEmail != "test@example.com" || emailSvc.sentOtp != "123456" {
		t.Errorf("expected email to be sent to test@example.com with OTP 123456, got %s / %s", emailSvc.sentEmail, emailSvc.sentOtp)
	}
}

type mockUserRepo struct {
	repositories.IUserRepository
}

type mockPasswordHasher struct {
	security.IPasswordHasher
}

func TestResendRegisterOtp_Expired(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     "",
		cooldown: false,
	}

	uc := &RegisterUseCase{
		otpService: otpSvc,
	}

	_, err := uc.ResendRegisterOtp(context.Background(), dto.ResendOtpReq{
		Email: "test@example.com",
	})

	if err == nil {
		t.Fatal("expected error for expired/non-existent session, got nil")
	}
}

func TestResendRegisterOtp_Cooldown(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     `{"username": "testuser"}`,
		cooldown: true,
	}

	uc := &RegisterUseCase{
		otpService: otpSvc,
	}

	_, err := uc.ResendRegisterOtp(context.Background(), dto.ResendOtpReq{
		Email: "test@example.com",
	})

	if err == nil {
		t.Fatal("expected cooldown error, got nil")
	}
}

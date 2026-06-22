package usecase

import (
	"context"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/communication"
)

type mockRecoveryEmailService struct {
	communication.IEmailService
	sentEmail string
	sentOtp   string
}

func (m *mockRecoveryEmailService) SendForgotPasswordOtpEmail(ctx context.Context, email string, otp string, expiry int) error {
	m.sentEmail = email
	m.sentOtp = otp
	return nil
}

func TestResendForgotPasswordOtp_Success(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     `{"UserId": "some-uuid"}`,
		cooldown: false,
	}
	emailSvc := &mockRecoveryEmailService{}

	uc := &PasswordRecoveryUseCase{
		otpService:   otpSvc,
		emailService: emailSvc,
		cfg: &config.Config{
			Otp: config.Otp{
				ExpiryMinutes: 5,
			},
		},
	}

	ok, err := uc.ResendForgotPasswordOtp(context.Background(), dto.ResendOtpReq{
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

func TestResendForgotPasswordOtp_Expired(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     "",
		cooldown: false,
	}

	uc := &PasswordRecoveryUseCase{
		otpService: otpSvc,
	}

	_, err := uc.ResendForgotPasswordOtp(context.Background(), dto.ResendOtpReq{
		Email: "test@example.com",
	})

	if err == nil {
		t.Fatal("expected error for expired/non-existent recovery session, got nil")
	}
}

func TestResendForgotPasswordOtp_Cooldown(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     `{"UserId": "some-uuid"}`,
		cooldown: true,
	}

	uc := &PasswordRecoveryUseCase{
		otpService: otpSvc,
	}

	_, err := uc.ResendForgotPasswordOtp(context.Background(), dto.ResendOtpReq{
		Email: "test@example.com",
	})

	if err == nil {
		t.Fatal("expected cooldown error, got nil")
	}
}

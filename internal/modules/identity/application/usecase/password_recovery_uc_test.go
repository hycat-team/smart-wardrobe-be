package usecase

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
)

type mockRecoveryNotifContract struct {
	notificationContract.INotificationContract
	sentEmail string
	sentOtp   string
}

func (m *mockRecoveryNotifContract) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	m.sentEmail = to
	if otp, ok := params["OTPCode"].(string); ok {
		m.sentOtp = otp
	}
	return nil
}

func TestResendForgotPasswordOtp_Success(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     `{"UserId": "some-uuid"}`,
		cooldown: false,
	}
	notifSvc := &mockRecoveryNotifContract{}

	uc := &PasswordRecoveryUseCase{
		otpService:    otpSvc,
		notifContract: notifSvc,
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
	if notifSvc.sentEmail != "test@example.com" || notifSvc.sentOtp != "123456" {
		t.Errorf("expected email to be sent to test@example.com with OTP 123456, got %s / %s", notifSvc.sentEmail, notifSvc.sentOtp)
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

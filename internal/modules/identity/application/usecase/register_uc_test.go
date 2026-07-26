package usecase

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/identity"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
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

type mockNotifContract struct {
	notificationContract.INotificationContract
	sentEmail string
	sentOtp   string
}

func (m *mockNotifContract) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	m.sentEmail = to
	if otp, ok := params["OTPCode"].(string); ok {
		m.sentOtp = otp
	}
	return nil
}

func TestResendRegisterOtp_Success(t *testing.T) {
	otpSvc := &mockOtpService{
		data:     `{"username": "testuser"}`,
		cooldown: false,
	}
	notifSvc := &mockNotifContract{}

	uc := &RegisterUseCase{
		otpService:    otpSvc,
		notifContract: notifSvc,
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
	if notifSvc.sentEmail != "test@example.com" || notifSvc.sentOtp != "123456" {
		t.Errorf("expected email to be sent to test@example.com with OTP 123456, got %s / %s", notifSvc.sentEmail, notifSvc.sentOtp)
	}
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

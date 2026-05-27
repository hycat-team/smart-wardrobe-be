package identity

import "context"

type IOtpService interface {
	GenerateOtp(ctx context.Context, email string, tempUserDataJson string, purpose string) (string, error)
	VerifyOtp(ctx context.Context, email string, otpCode string, purpose string) (string, error)
	IsInResendCooldown(ctx context.Context, email string, purpose string) (bool, error)
}

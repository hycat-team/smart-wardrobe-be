package communication

import "context"

type IEmailService interface {
	SendForgotPasswordOtpEmail(ctx context.Context, toEmail string, otpCode string, expiryMinutes int) error
	SendRegistrationOtpEmail(ctx context.Context, toEmail string, otpCode string, expiryMinutes int) error
}

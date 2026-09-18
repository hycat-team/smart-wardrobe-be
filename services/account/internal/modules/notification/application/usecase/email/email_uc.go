package email

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
)

// EmailUseCase orchestrate gửi email qua IEmailSender (infrastructure).
// Presentation worker gọi qua interface này — không gọi trực tiếp SmtpSender.
type EmailUseCase struct {
	sender usecase.IEmailSender
}

func NewEmailUseCase(sender usecase.IEmailSender) usecase.IEmailUseCase {
	return &EmailUseCase{sender: sender}
}

func (uc *EmailUseCase) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	return uc.sender.Send(ctx, to, subject, htmlBody)
}

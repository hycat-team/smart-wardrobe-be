package usecase

import "context"

// IEmailSender là contract cho infrastructure gửi email (vd: SMTP).
// Application phụ thuộc interface này, không phụ thuộc concrete sender.
type IEmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// IEmailUseCase là boundary application layer cho email worker gọi.
type IEmailUseCase interface {
	SendEmail(ctx context.Context, to, subject, htmlBody string) error
}

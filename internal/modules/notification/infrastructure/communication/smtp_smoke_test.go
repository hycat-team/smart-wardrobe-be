package communication

import (
	"context"
	"os"
	"testing"

	"smart-wardrobe-be/internal/config"
)

// TestSmtpSendSmoke sends a real email through the configured SMTP relay.
// It is skipped unless SMOKE_EMAIL_TO is set, e.g.:
//
//	SMOKE_EMAIL_TO=you@gmail.com EMAIL_SENDER_EMAIL=no-reply@hycat.online \
//	EMAIL_SMTP_USER=brevo-account@mail.com EMAIL_SMTP_KEY=... \
//	EMAIL_REPLY_TO=you@gmail.com go test -run TestSmtpSendSmoke -v \
//	./internal/modules/notification/infrastructure/communication/
func TestSmtpSendSmoke(t *testing.T) {
	to := os.Getenv("SMOKE_EMAIL_TO")
	if to == "" {
		t.Skip("SMOKE_EMAIL_TO not set — skipping real SMTP smoke test")
	}

	cfg := &config.Config{Email: config.Email{
		Host:        envOr("EMAIL_HOST_OVERRIDE", "smtp-relay.brevo.com"),
		Port:        587,
		SenderName:  "Closy Smoke Test",
		SenderEmail: os.Getenv("EMAIL_SENDER_EMAIL"),
		SmtpUser:    os.Getenv("EMAIL_SMTP_USER"),
		SmtpKey:     os.Getenv("EMAIL_SMTP_KEY"),
		ReplyTo:     os.Getenv("EMAIL_REPLY_TO"),
	}}
	if cfg.Email.SenderEmail == "" || cfg.Email.SmtpUser == "" || cfg.Email.SmtpKey == "" {
		t.Fatal("EMAIL_SENDER_EMAIL / EMAIL_SMTP_USER / EMAIL_SMTP_KEY must be set for the smoke test")
	}

	sender := NewSmtpSender(cfg)
	err := sender.Send(context.Background(), to, "[Closy] SMTP smoke test",
		"<h3>Smoke test thành công</h3><p>Email gửi từ <b>"+cfg.Email.SenderEmail+"</b> qua "+cfg.Email.Host+".</p>"+
			"<p>Kiểm tra header SPF/DKIM/DMARC bằng mail-tester nếu cần.</p>")
	if err != nil {
		t.Fatalf("smtp send failed: %v", err)
	}
	t.Logf("email sent to %s from %s", to, cfg.Email.SenderEmail)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

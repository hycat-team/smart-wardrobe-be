package communication

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/smtp"
	"strings"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/interface/communication"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/communication/templates"
)

type GmailSmtpService struct {
	cfg *config.Config
}

func NewGmailSmtpService(cfg *config.Config) communication.IEmailService {
	return &GmailSmtpService{cfg: cfg}
}

func (s *GmailSmtpService) SendRegistrationOtpEmail(ctx context.Context, toEmail string, otpCode string, expiryMinutes int) error {
	subject := fmt.Sprintf("[%s] Mã xác thực đăng ký tài khoản", s.cfg.Email.SenderName)
	body := strings.ReplaceAll(templates.RegistrationOtpTemplate, "{{OTP_CODE}}", otpCode)
	body = strings.ReplaceAll(body, "{{EXPIRY_MINUTES}}", fmt.Sprintf("%d", expiryMinutes))
	return s.sendEmailInternal(toEmail, subject, body)
}

func (s *GmailSmtpService) SendForgotPasswordOtpEmail(ctx context.Context, toEmail string, otpCode string, expiryMinutes int) error {
	subject := fmt.Sprintf("[%s] Mã xác thực đặt lại mật khẩu", s.cfg.Email.SenderName)
	body := strings.ReplaceAll(templates.ForgotPasswordTemplate, "{{OTP_CODE}}", otpCode)
	body = strings.ReplaceAll(body, "{{EXPIRY_MINUTES}}", fmt.Sprintf("%d", expiryMinutes))
	return s.sendEmailInternal(toEmail, subject, body)
}

func (s *GmailSmtpService) sendEmailInternal(toEmail, subject, htmlBody string) error {
	from := s.cfg.Email.SenderEmail
	password := s.cfg.Email.AppPassword
	smtpHost := s.cfg.Email.Host
	smtpPort := s.cfg.Email.Port

	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	auth := smtp.PlainAuth("", from, password, smtpHost)

	headerSubject := fmt.Sprintf("Subject: %s\r\n", subject)
	headerTo := fmt.Sprintf("To: %s\r\n", toEmail)
	headerFrom := fmt.Sprintf("From: %s <%s>\r\n", s.cfg.Email.SenderName, from)
	headerMime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	msg := []byte(headerFrom + headerTo + headerSubject + headerMime + htmlBody)

	if smtpPort == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         smtpHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial TLS: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, smtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}

		if err = client.Mail(from); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		if err = client.Rcpt(toEmail); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to open data writer: %w", err)
		}
		defer w.Close()

		if _, err = w.Write(msg); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		_ = client.Quit()
		return nil
	}

	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer conn.Close()

	tlsConfig := &tls.Config{
		ServerName: smtpHost,
	}
	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}

	if err = conn.Mail(from); err != nil {
		return fmt.Errorf("SMTP Mail command failed: %w", err)
	}

	if err = conn.Rcpt(toEmail); err != nil {
		return fmt.Errorf("SMTP Rcpt command failed: %w", err)
	}

	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("SMTP Data command failed: %w", err)
	}
	defer w.Close()

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("failed to write SMTP body: %w", err)
	}

	_ = conn.Quit()
	return nil
}

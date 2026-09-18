package communication

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"

	"smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/shared/config"
)

var _ usecase.IEmailSender = (*SmtpSender)(nil)

type SmtpSender struct {
	cfg           *config.Config
	registerTmpl  *template.Template
	forgotPwdTmpl *template.Template
}

func NewSmtpSender(cfg *config.Config) *SmtpSender {
	return &SmtpSender{
		cfg: cfg,
	}
}

func (s *SmtpSender) Send(ctx context.Context, to, subject, htmlBody string) error {
	from := s.cfg.Email.SenderEmail
	// SMTP username: not the sender address — fall back to the sender email for Gmail-style servers.
	user := s.cfg.Email.SmtpUser
	if user == "" {
		user = from
	}
	password := s.cfg.Email.SmtpKey
	smtpHost := s.cfg.Email.Host
	smtpPort := s.cfg.Email.Port

	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	auth := smtp.PlainAuth("", user, password, smtpHost)

	headerSubject := fmt.Sprintf("Subject: %s\r\n", subject)
	headerTo := fmt.Sprintf("To: %s\r\n", to)
	headerFrom := fmt.Sprintf("From: %s <%s>\r\n", s.cfg.Email.SenderName, from)
	headerReplyTo := ""
	if s.cfg.Email.ReplyTo != "" {
		headerReplyTo = fmt.Sprintf("Reply-To: %s\r\n", s.cfg.Email.ReplyTo)
	}
	headerMime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	msg := []byte(headerFrom + headerReplyTo + headerTo + headerSubject + headerMime + htmlBody)

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
		if err = client.Rcpt(to); err != nil {
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

	tlsConfig := &tls.Config{ServerName: smtpHost}
	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}
	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}
	if err = conn.Mail(from); err != nil {
		return fmt.Errorf("SMTP Mail command failed: %w", err)
	}
	if err = conn.Rcpt(to); err != nil {
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

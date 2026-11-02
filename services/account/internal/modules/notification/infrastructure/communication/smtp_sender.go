package communication

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"mime"
	"net/smtp"
	"time"

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

// buildMessage assembles the RFC 5322 message body: headers, then a
// multipart/alternative body with a text/plain part first and the HTML part
// second (the order matters — a client picks the LAST part it can render).
//
// The subject is RFC 2047 encoded. Vietnamese subjects are non-ASCII, and a raw
// 8-bit Subject header is both invalid and a spam signal.
func (s *SmtpSender) buildMessage(to, subject, htmlBody string) []byte {
	boundary := fmt.Sprintf("=_closy_%d", time.Now().UnixNano())

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s <%s>\r\n", s.cfg.Email.SenderName, s.cfg.Email.SenderEmail)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.BEncoding.Encode("UTF-8", subject))
	if s.cfg.Email.ReplyTo != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", s.cfg.Email.ReplyTo)
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(plainTextFromHTML(htmlBody))
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(htmlBody)
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes()
}

func (s *SmtpSender) Send(ctx context.Context, to, subject, htmlBody string) (err error) {
	// 025 T113: while `email.enabled` is false nothing is transmitted, so the
	// function returns before a socket is opened. It returns nil, not an error:
	// the message was handled on purpose, so reporting a failure would make the
	// worker requeue a message that must never be sent. The email worker already
	// skips before it gets here; this guard is the backstop that keeps a future
	// caller from dialling the provider by accident.
	if !s.cfg.Email.Enabled {
		return nil
	}

	// One classification point for every failure below. The worker decides
	// retry-vs-DLQ from this, and a transport error that reaches it unclassified
	// would be retried blindly.
	defer func() { err = classifyDeliveryError(err) }()

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

	msg := s.buildMessage(to, subject, htmlBody)

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

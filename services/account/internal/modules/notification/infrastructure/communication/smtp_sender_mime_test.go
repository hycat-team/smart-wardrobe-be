package communication

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

// TestBuildMessageIsMultipartAlternative pins the shape the customer's client
// sees. Sentinel relied on a single text/html part; a message without a text
// alternative scores worse with spam filters and is unreadable in clients that
// refuse HTML.
func TestBuildMessageIsMultipartAlternative(t *testing.T) {
	s := &SmtpSender{cfg: smtpConfig(2525, true)}
	s.cfg.Email.SenderEmail = "no-reply@closy.vn"
	s.cfg.Email.SenderName = "Closy"
	s.cfg.Email.ReplyTo = "support@closy.vn"

	const htmlBody = `<html><head><title>bỏ qua</title></head><body>` +
		`<h2>Gói Dịch Vụ Đã Hết Hạn</h2>` +
		`<p>Gói <strong>Premium Plan</strong> của bạn đã hết hạn vào ngày 01/11/2026.</p>` +
		`<p>Tài khoản của bạn hiện đã trở về <strong>Free Plan</strong>.</p>` +
		`</body></html>`

	raw := s.buildMessage("khach@example.com", "[Closy] Gói dịch vụ của bạn đã hết hạn", htmlBody)

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("message is not parseable as RFC 5322: %v", err)
	}

	// net/mail keeps the header encoded; a Vietnamese subject must survive the
	// RFC 2047 round trip rather than arriving as =?UTF-8?...?=.
	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if decodedSubject != "[Closy] Gói dịch vụ của bạn đã hết hạn" {
		t.Fatalf("subject did not survive encoding, got %q", decodedSubject)
	}
	if got := msg.Header.Get("To"); got != "khach@example.com" {
		t.Fatalf("unexpected To header %q", got)
	}
	if got := msg.Header.Get("Reply-To"); got != "support@closy.vn" {
		t.Fatalf("unexpected Reply-To header %q", got)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}
	if mediaType != "multipart/alternative" {
		t.Fatalf("expected multipart/alternative, got %q", mediaType)
	}
	boundary := params["boundary"]
	if boundary == "" {
		t.Fatalf("multipart message has no boundary")
	}

	reader := multipart.NewReader(msg.Body, boundary)
	parts := readParts(t, reader)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (text then html), got %d", len(parts))
	}

	// Order matters: a client renders the LAST part it understands, so the HTML
	// has to come second.
	if ct, _, _ := mime.ParseMediaType(parts[0].contentType); ct != "text/plain" {
		t.Fatalf("first part must be text/plain, got %q", parts[0].contentType)
	}
	if ct, _, _ := mime.ParseMediaType(parts[1].contentType); ct != "text/html" {
		t.Fatalf("second part must be text/html, got %q", parts[1].contentType)
	}

	plain := parts[0].body
	if strings.Contains(plain, "<") {
		t.Fatalf("text part still contains markup:\n%s", plain)
	}
	for _, want := range []string{"Gói Dịch Vụ Đã Hết Hạn", "Premium Plan", "01/11/2026", "Free Plan"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("text part is missing %q:\n%s", want, plain)
		}
	}
	// The <title> lives inside <head>, which must not leak into the prose.
	if strings.Contains(plain, "bỏ qua") {
		t.Fatalf("text part leaked the document title:\n%s", plain)
	}
	if strings.TrimRight(parts[1].body, "\r\n") != htmlBody {
		t.Fatalf("html part must be the rendered template verbatim")
	}
}

// TestBuildMessageOmitsEmptyReplyTo keeps the header list honest: an unset
// reply-to must not appear as an empty header.
func TestBuildMessageOmitsEmptyReplyTo(t *testing.T) {
	s := &SmtpSender{cfg: smtpConfig(2525, true)}
	raw := s.buildMessage("khach@example.com", "Chủ đề", "<p>nội dung</p>")

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	if _, ok := msg.Header["Reply-To"]; ok {
		t.Fatalf("an empty Reply-To must not be written as a header")
	}
}

type mimePart struct {
	contentType string
	body        string
}

func readParts(t *testing.T, reader *multipart.Reader) []mimePart {
	t.Helper()
	var parts []mimePart
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return parts
		}
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read part body: %v", err)
		}
		parts = append(parts, mimePart{contentType: part.Header.Get("Content-Type"), body: string(body)})
	}
}

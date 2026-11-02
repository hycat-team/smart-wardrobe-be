package communication

import (
	"strings"
	"testing"
)

func TestPlainTextFromHTMLKeepsProseAndDropsMarkup(t *testing.T) {
	html := `<html><head><title>Tiêu đề tài liệu</title><style>p{color:#333}</style></head>` +
		`<body><div><h2>Gói Dịch Vụ Đã Hết Hạn</h2>` +
		`<p>Gói <strong>Premium Plan</strong> đã hết hạn.</p>` +
		`<p>Số dư: 5.000đ &amp; cần 1.000.000đ</p></div></body></html>`

	text := plainTextFromHTML(html)

	if strings.ContainsAny(text, "<>") {
		t.Fatalf("markup survived the conversion:\n%s", text)
	}
	if strings.Contains(text, "Tiêu đề tài liệu") {
		t.Fatalf("the document title leaked into the prose:\n%s", text)
	}
	if strings.Contains(text, "color:#333") {
		t.Fatalf("CSS leaked into the prose:\n%s", text)
	}
	for _, want := range []string{
		"Gói Dịch Vụ Đã Hết Hạn",
		"Premium Plan",
		"Số dư: 5.000đ & cần 1.000.000đ",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in the plain text:\n%s", want, text)
		}
	}
	// Block boundaries must survive as separate lines, otherwise two paragraphs
	// read as one run-on sentence.
	if !strings.Contains(text, "Đã Hết Hạn\nGói Premium") {
		t.Fatalf("paragraph breaks were lost:\n%q", text)
	}
}

func TestPlainTextFromHTMLCollapsesWhitespaceButKeepsParagraphs(t *testing.T) {
	html := "<p>Một</p><p>Hai</p><p>Ba</p>"
	text := plainTextFromHTML(html)
	if strings.Contains(text, "\n\n") {
		t.Fatalf("blank lines between single paragraphs must be collapsed, got %q", text)
	}
	if text != "Một\nHai\nBa" {
		t.Fatalf("unexpected conversion: %q", text)
	}
}

func TestPlainTextFromHTMLEmptyBody(t *testing.T) {
	if got := plainTextFromHTML(""); got != "" {
		t.Fatalf("empty body must produce empty text, got %q", got)
	}
}

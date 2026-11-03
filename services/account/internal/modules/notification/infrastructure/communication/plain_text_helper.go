package communication

import (
	"html"
	"regexp"
	"strings"
)

// plainTextFromHTML renders a readable text/plain alternative for an email whose
// only source is the HTML template.
//
// The reason it exists rather than being copied from the HTML: a message with no
// text part scores worse with spam filters and is unreadable in clients that
// refuse HTML. The conversion is deliberately lossy - structure (paragraph
// breaks) is kept, styling is not.
var (
	// Dropped whole: their content is not prose. `head` carries the <title>, which
	// would otherwise leak into the visible text.
	nonProseBlock = regexp.MustCompile(`(?is)<(style|head|script)\b[^>]*>.*?</(style|head|script)>`)
	// Block-level boundaries become newlines, so paragraphs do not run together.
	blockBoundary = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr|table|body|html)\s*>|<br\s*/?>`)
	anyTag        = regexp.MustCompile(`<[^>]*>`)
	lineTrailing  = regexp.MustCompile(`[ \t]+\n`)
	blankLines    = regexp.MustCompile(`\n{3,}`)
	spaceRun      = regexp.MustCompile(`[ \t]{2,}`)
)

func plainTextFromHTML(body string) string {
	text := nonProseBlock.ReplaceAllString(body, "")
	text = blockBoundary.ReplaceAllString(text, "\n")
	text = anyTag.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = lineTrailing.ReplaceAllString(text, "\n")
	text = spaceRun.ReplaceAllString(text, " ")
	text = blankLines.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

package synthesis

import (
	"regexp"
	"strings"
)

var (
	// "(A7)", "( a7 )" — xóa kèm ngoặc trước để không còn ngoặc trống.
	parenthesizedAliasPattern = regexp.MustCompile(`(?i)\(\s*A\d{1,3}\s*\)`)
	// Alias trần còn sót: "A7", "a2".
	bareAliasPattern        = regexp.MustCompile(`(?i)\bA\d{1,3}\b`)
	multiSpacePattern       = regexp.MustCompile(`\s{2,}`)
	spaceBeforePunctPattern = regexp.MustCompile(`\s+([,.!?:;)])`)
	emptyParensPattern      = regexp.MustCompile(`\(\s*\)`)
)

// SanitizeOutfitNarrative loại bỏ alias ứng viên nội bộ (A1, A2, ...)
// khỏi văn bản hiển thị cho người dùng (title/explanation).
// Alias chỉ hợp lệ trong primary_id/alternative_ids nên mọi xuất hiện
// trong câu chữ tự nhiên đều là LLM vô tình leak ra.
func SanitizeOutfitNarrative(text string) string {
	if text == "" {
		return ""
	}
	cleaned := parenthesizedAliasPattern.ReplaceAllString(text, "")
	cleaned = bareAliasPattern.ReplaceAllString(cleaned, "")
	cleaned = emptyParensPattern.ReplaceAllString(cleaned, "")
	cleaned = spaceBeforePunctPattern.ReplaceAllString(cleaned, "$1")
	cleaned = multiSpacePattern.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

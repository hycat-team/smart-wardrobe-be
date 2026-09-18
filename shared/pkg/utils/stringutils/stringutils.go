package stringutils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

var vietnameseReplacer *strings.Replacer

func init() {
	var pairs []string
	chars := map[string]struct {
		base  string
		tones string
	}{
		"a": {"a", "áàảãạăắằẳẵặâấầẩẫậ"},
		"e": {"e", "éèẻẽẹêếềểễệ"},
		"i": {"i", "íìỉĩị"},
		"o": {"o", "óòỏõọôốồổỗộơớờởỡợ"},
		"u": {"u", "úùủũụưứừửữự"},
		"y": {"y", "ýỳỷỹỵ"},
	}
	for _, c := range chars {
		for _, r := range c.tones {
			pairs = append(pairs, string(r), c.base)
			pairs = append(pairs, string(unicode.ToUpper(r)), strings.ToUpper(c.base))
		}
	}
	pairs = append(pairs, "đ", "d", "Đ", "D")
	vietnameseReplacer = strings.NewReplacer(pairs...)
}

func RemoveVietnameseSigns(str string) string {
	return vietnameseReplacer.Replace(str)
}

func HashSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func GetString(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}

func ToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TrimmedPtr(in *string) *string {
	if in == nil {
		return nil
	}
	return ToPtr(strings.TrimSpace(*in))
}

func CleanJSONMarkdown(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}
	return strings.TrimSpace(content)
}

// ToNormalizedSet converts a slice of strings to a lookup map with lowercased and trimmed values.
func ToNormalizedSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			set[value] = true
		}
	}
	return set
}

// TruncateLogText cắt ngắn chuỗi nhật ký văn bản theo giới hạn ký tự tối đa để tránh ghi đè bộ nhớ hoặc file log quá lớn.
func TruncateLogText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "...(truncated)"
}

// TruncateRunes cắt ngắn chuỗi theo giới hạn số lượng rune.
func TruncateRunes(value string, limit int) string {
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

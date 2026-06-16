package stringutils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// pascal case / camel case -> snake case
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder
	runes := []rune(s)
	n := len(runes)

	for i := range n {
		b.WriteRune(unicode.ToLower(runes[i]))

		nextIsUpper := i+1 < n && unicode.IsUpper(runes[i+1])
		overIsLowerOrNil := (i+2 >= n && unicode.IsLower(runes[i])) ||
			(i+2 < n && unicode.IsLower(runes[i+2]))
		if nextIsUpper && overIsLowerOrNil {
			b.WriteRune('_')
		}

	}

	return b.String()
}

func HashString(stringutils string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(stringutils))
	return hex.EncodeToString(h.Sum(nil))
}

func GetString(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}

func UuidPtrToStr(in *uuid.UUID) string {
	if in == nil {
		return ""
	}
	return in.String()
}

func UuidPtrToStringPtr(in *uuid.UUID) *string {
	if in == nil || *in == uuid.Nil {
		return nil
	}

	s := in.String()
	return &s
}

func ToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func SanitizeFileName(fileName string, salt *string) string {
	// 1. Only keep the file name, strip directory path (Prevent Path Traversal)
	name := filepath.Base(fileName)

	// 2. Remove the old file extension to process separately
	ext := filepath.Ext(name)
	nameOnly := strings.TrimSuffix(name, ext)

	// 3. Regex: Only keep alphanumeric characters, hyphens, and underscores
	// Replace all special characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)
	safeName := reg.ReplaceAllString(nameOnly, "_")

	// 4. Limit the file name length (avoid OS errors)
	if len(safeName) > 100 {
		safeName = safeName[:100]
	}

	// 5. Combine with Salt or UnixNano
	// If salt is provided, use salt. Otherwise, use current time.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	if salt != nil && *salt != "" {
		suffix = *salt
	}

	return fmt.Sprintf("%s-%s%s", safeName, suffix, ext)
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

package wardrobe

import "strings"

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "safety") ||
		strings.Contains(errStr, "blocked") ||
		strings.Contains(errStr, "invalid image") ||
		strings.Contains(errStr, "unsupported media type") ||
		strings.Contains(errStr, "corrupted") ||
		strings.Contains(errStr, "no fashion item") ||
		strings.Contains(errStr, "not found") {
		return false
	}

	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "eof") {
		return true
	}

	if strings.Contains(errStr, "http 400") ||
		strings.Contains(errStr, "http 401") ||
		strings.Contains(errStr, "http 403") {
		return false
	}

	return true
}

package worker

import (
	"strings"

	"smart-wardrobe-be/shared/application/constants/eventconstants"
)

// analyzeRetryTopic maps a retry attempt to its delay queue. The delays are
// broker-side constants (see messaging/topology.go): attempt 1 waits one
// minute, attempt 2 five minutes, and every later attempt fifteen. A retry count
// above three reuses the last tier, which is the right degradation when
// max_retry_count is raised — a longer wait is always safe, a shorter one is
// what broke production.
func analyzeRetryTopic(retryCount int) string {
	switch {
	case retryCount <= 1:
		return eventconstants.TopicFashionAnalyzeRetry1
	case retryCount == 2:
		return eventconstants.TopicFashionAnalyzeRetry2
	default:
		return eventconstants.TopicFashionAnalyzeRetry3
	}
}

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

package ai

import (
	"net/url"
	"strings"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/pkg/logger"
)

// AttemptTelemetry and its logger live in shared/application/ai so the AI
// application layer can emit them without importing infrastructure. These
// aliases keep the infrastructure call sites and their tests unchanged.
type AttemptTelemetry = app_ai.AttemptTelemetry

type AttemptTelemetryLogger = app_ai.AttemptTelemetryLogger

func NewAttemptTelemetryLogger(log logger.Interface) *AttemptTelemetryLogger {
	return app_ai.NewAttemptTelemetryLogger(log)
}

// RedactEndpoint removes credentials, query values and the fragment from a
// provider endpoint so it can be logged for host diagnostics. It returns an
// empty string for an unparseable value rather than echoing a string that might
// itself be a secret.
func RedactEndpoint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	parsed.User = nil
	if parsed.RawQuery != "" {
		query := parsed.Query()
		for key := range query {
			query.Set(key, "redacted")
		}
		parsed.RawQuery = query.Encode()
	}
	parsed.Fragment = ""
	return parsed.String()
}

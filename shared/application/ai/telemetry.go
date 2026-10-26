package ai

import (
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

// AttemptTelemetry is the only attempt-level payload allowed in structured
// logs. It has no prompt, message, credential or raw-response field, so those
// values cannot be logged by construction.
//
// It lives in the application package because the AI use case emits it; the
// shared infrastructure package re-exports the same type for the text service.
type AttemptTelemetry struct {
	Service         string
	Version         string
	Environment     string
	RequestID       string
	Operation       string
	Capability      string
	AttemptIndex    int
	Provider        string
	Model           string
	BillingClass    string
	Outcome         string
	FallbackReason  string
	HTTPStatus      int
	TTFTMs          int64
	LatencyMs       int64
	PromptTokens    int64
	OutputTokens    int64
	ThinkingTokens  int64
	UsageReported   bool
	CostMicroVND    int64
	InputCharacters int
	MaxOutputTokens int
}

// AttemptTelemetryLogger emits one structured line per provider attempt.
type AttemptTelemetryLogger struct {
	logger logger.Interface
}

func NewAttemptTelemetryLogger(log logger.Interface) *AttemptTelemetryLogger {
	return &AttemptTelemetryLogger{logger: log}
}

// LogAttempt writes the required attempt keys. Timestamp and level are added by
// the logger itself; every other field comes from the typed payload.
func (l *AttemptTelemetryLogger) LogAttempt(entry AttemptTelemetry) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Info("ai provider attempt",
		zap.String("service", entry.Service),
		zap.String("version", entry.Version),
		zap.String("environment", entry.Environment),
		zap.String("request_id", entry.RequestID),
		zap.String("operation", entry.Operation),
		zap.String("capability", entry.Capability),
		zap.Int("attempt_index", entry.AttemptIndex),
		zap.String("provider", entry.Provider),
		zap.String("model", entry.Model),
		zap.String("billing_class", entry.BillingClass),
		zap.String("outcome", entry.Outcome),
		zap.String("fallback_reason", entry.FallbackReason),
		zap.Int("http_status", entry.HTTPStatus),
		zap.Int64("ttft_ms", entry.TTFTMs),
		zap.Int64("latency_ms", entry.LatencyMs),
		zap.Int64("prompt_tokens", entry.PromptTokens),
		zap.Int64("output_tokens", entry.OutputTokens),
		zap.Int64("thinking_tokens", entry.ThinkingTokens),
		zap.Bool("usage_reported", entry.UsageReported),
		zap.Int64("cost_micro_vnd", entry.CostMicroVND),
		zap.Int("input_characters", entry.InputCharacters),
		zap.Int("max_output_tokens", entry.MaxOutputTokens),
	)
}

package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/shared/infrastructure/ai/openai"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// telemetryRecordingLogger captures the fields passed to Info/Warn instead of
// writing to stdout, so the tests can assert the emitted payload without
// depending on the logger's encoder.
type telemetryRecordingLogger struct {
	message string
	fields  []zap.Field
}

func (l *telemetryRecordingLogger) Debug(string, ...zap.Field) {}
func (l *telemetryRecordingLogger) Info(message string, fields ...zap.Field) {
	l.message = message
	l.fields = append([]zap.Field(nil), fields...)
}
func (l *telemetryRecordingLogger) Warn(message string, fields ...zap.Field) {
	l.message = message
	l.fields = append([]zap.Field(nil), fields...)
}
func (l *telemetryRecordingLogger) Error(string, ...zap.Field) {}
func (l *telemetryRecordingLogger) Fatal(string, ...zap.Field) {}

func telemetryField(fields []zap.Field, key string) (zap.Field, bool) {
	for _, field := range fields {
		if field.Key == key {
			return field, true
		}
	}
	return zap.Field{}, false
}

func requiredTelemetryKeys() []string {
	return []string{
		"service",
		"version",
		"environment",
		"request_id",
		"operation",
		"capability",
		"attempt_index",
		"provider",
		"model",
		"billing_class",
		"outcome",
		"http_status",
		"ttft_ms",
		"latency_ms",
		"prompt_tokens",
		"output_tokens",
		"thinking_tokens",
		"usage_reported",
		"cost_micro_vnd",
		"input_characters",
		"max_output_tokens",
	}
}

func TestAttemptTelemetryEmitsRequiredKeysWithoutPromptOrSecret(t *testing.T) {
	recorder := &telemetryRecordingLogger{}
	entry := app_ai.AttemptTelemetry{
		Service:         "ai",
		Version:         "image-tag",
		Environment:     "aws-course",
		RequestID:       "6f6b1e0e-0f2b-4f0e-9f7a-0d2b5c8e1a11",
		Operation:       "chat",
		Capability:      "chat_text",
		AttemptIndex:    1,
		Provider:        "openai",
		Model:           "qwen3:4b",
		BillingClass:    "unbilled",
		Outcome:         "success",
		HTTPStatus:      http.StatusOK,
		TTFTMs:          120,
		LatencyMs:       340,
		PromptTokens:    8,
		OutputTokens:    4,
		ThinkingTokens:  0,
		UsageReported:   true,
		CostMicroVND:    0,
		InputCharacters: 42,
		MaxOutputTokens: 64,
	}

	NewAttemptTelemetryLogger(recorder).LogAttempt(entry)

	if recorder.message != "ai provider attempt" {
		t.Fatalf("message = %q, want the attempt log message", recorder.message)
	}

	emitted := make(map[string]zap.Field, len(recorder.fields))
	for _, field := range recorder.fields {
		emitted[field.Key] = field
	}

	for _, key := range requiredTelemetryKeys() {
		if _, ok := emitted[key]; !ok {
			t.Errorf("missing required telemetry key %q", key)
		}
	}

	// Exact keys only. Names like prompt_tokens are allowed; a bare prompt,
	// message body, Authorization header or credential is not.
	for _, key := range []string{"prompt", "system_prompt", "messages", "content", "authorization", "token", "secret", "api_key"} {
		if _, ok := emitted[key]; ok {
			t.Errorf("forbidden telemetry key %q was emitted", key)
		}
	}

	if field, ok := emitted["provider"]; !ok || field.String != "openai" {
		t.Errorf("provider = %+v, want openai", field)
	}
	if field, ok := emitted["thinking_tokens"]; !ok || field.Integer != 0 {
		t.Errorf("thinking_tokens = %+v, want 0", field)
	}
	if field, ok := emitted["cost_micro_vnd"]; !ok || field.Integer != 0 {
		t.Errorf("cost_micro_vnd = %+v, want 0 for the unbilled trial attempt", field)
	}
}

// TestOpenAITelemetryKeepsThinkingTokensZero proves the openai path reports zero
// thinking tokens even when the provider returns a reasoning-token counter: the
// adapter deliberately does not parse reasoning content, and telemetry must not
// invent a value. A non-zero value is preserved by the logger so a real
// configuration error remains visible.
func TestOpenAITelemetryKeepsThinkingTokensZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"finish_reason":"stop","message":{"content":"ok"}}],
			"usage":{
				"prompt_tokens":7,
				"completion_tokens":3,
				"total_tokens":10,
				"completion_tokens_details":{"reasoning_tokens":99}
			}
		}`))
	}))
	defer server.Close()

	var usage app_ai.TokenUsage
	_, err := openai.CallText(context.Background(), server.Client(), config.APIProviderConfig{
		Provider: "openai",
		ApiKey:   "fake-bearer-that-must-not-be-logged",
		Endpoint: server.URL,
		Model:    "qwen3:4b",
	}, "system prompt", "user prompt that must not be logged", app_ai.TextGenerationOptions{
		MaxOutputTokens: 16,
		OnUsage: func(u app_ai.TokenUsage) {
			usage = u
		},
	})
	if err != nil {
		t.Fatalf("CallText: %v", err)
	}
	if usage.ThinkingTokens != 0 {
		t.Fatalf("openai usage thinking_tokens = %d, want 0", usage.ThinkingTokens)
	}

	capture := &attemptUsageCapture{usage: usage, reported: true}
	result := buildAttemptResult(nil, capture, 10*time.Millisecond, 5*time.Millisecond, false)

	recorder := &telemetryRecordingLogger{}
	NewAttemptTelemetryLogger(recorder).LogAttempt(app_ai.AttemptTelemetry{
		Service:        "ai",
		Provider:       "openai",
		Model:          "qwen3:4b",
		ThinkingTokens: result.Usage.ThinkingTokens,
		PromptTokens:   result.Usage.PromptTokens,
		OutputTokens:   result.Usage.OutputTokens,
		UsageReported:  result.UsageReported,
		Outcome:        result.Outcome,
	})

	field, ok := telemetryField(recorder.fields, "thinking_tokens")
	if !ok || field.Integer != 0 {
		t.Fatalf("thinking_tokens = %+v, want 0 on openai", field)
	}
	for _, field := range recorder.fields {
		if field.Type == zapcore.StringType {
			if strings.Contains(field.String, "fake-bearer") || strings.Contains(field.String, "must not be logged") {
				t.Errorf("telemetry field %q leaked a credential or prompt: %q", field.Key, field.String)
			}
		}
	}
}

func TestAttemptTelemetryRedactEndpoint(t *testing.T) {
	raw := "https://user:pass@ollama.example.com:443/v1/chat/completions?api_key=supersecret&trace=abc#fragment"

	redacted := RedactEndpoint(raw)

	if redacted == "" {
		t.Fatal("redacted endpoint is empty, want a host-only diagnostic value")
	}
	for _, leaked := range []string{"user", "pass", "supersecret", "trace=abc", "fragment"} {
		if strings.Contains(redacted, leaked) {
			t.Errorf("redacted endpoint %q still contains %q", redacted, leaked)
		}
	}
	if !strings.Contains(redacted, "ollama.example.com") {
		t.Errorf("redacted endpoint %q lost the host", redacted)
	}

	if got := RedactEndpoint("://not-a-url-with-secret"); got != "" {
		t.Errorf("unparseable endpoint = %q, want empty rather than echoing a possible secret", got)
	}
}

func TestOpenAIEndpointHostMismatch(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		expected     []string
		wantHost     string
		wantMismatch bool
	}{
		{
			name:     "empty expected list disables the warning",
			endpoint: "https://api.openai.com/v1/chat/completions",
			expected: nil,
		},
		{
			name:     "exact trial host matches",
			endpoint: "https://ollama.example.com/v1/chat/completions",
			expected: []string{"ollama.example.com"},
			wantHost: "ollama.example.com",
		},
		{
			name:     "host match is case insensitive and trims spaces",
			endpoint: "https://OLLAMA.Example.com/v1/chat/completions",
			expected: []string{" ollama.example.com "},
			wantHost: "ollama.example.com",
		},
		{
			name:     "endpoint port is ignored",
			endpoint: "https://ollama.example.com:8443/v1/chat/completions",
			expected: []string{"ollama.example.com"},
			wantHost: "ollama.example.com",
		},
		{
			name:         "different host warns",
			endpoint:     "https://api.openai.com/v1/chat/completions",
			expected:     []string{"ollama.example.com"},
			wantHost:     "api.openai.com",
			wantMismatch: true,
		},
		{
			name:         "unparseable endpoint warns",
			endpoint:     "://not-a-url",
			expected:     []string{"ollama.example.com"},
			wantMismatch: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, mismatch := OpenAIEndpointHostMismatch(test.endpoint, test.expected)
			if host != test.wantHost {
				t.Errorf("host = %q, want %q", host, test.wantHost)
			}
			if mismatch != test.wantMismatch {
				t.Errorf("mismatch = %v, want %v", mismatch, test.wantMismatch)
			}
		})
	}
}

package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
)

// Contract tests for the OpenAI-compatible adapter when it points at the Ollama
// proxy. Source of truth:
// specs/032-ollama-lightsail-trial/contracts/ollama-openai-subset.md.
//
// Every test drives a real httptest server so the request line, headers and SSE
// framing are exercised as the provider sees them, not as the adapter thinks it
// built them.

const (
	trialBearer = "trial-bearer-token"
	trialModel  = "qwen3:4b"
	chatPath    = "/v1/chat/completions"
	trialEffort = "none"
)

// trialProvider builds the provider config the Lightsail env produces during the
// trial: a full URL ending in /v1/chat/completions, a Bearer token and qwen3:4b.
func trialProvider(server *httptest.Server, effort string) config.APIProviderConfig {
	return config.APIProviderConfig{
		Provider:        "openai",
		ApiKey:          trialBearer,
		Endpoint:        server.URL + chatPath,
		Model:           trialModel,
		ReasoningEffort: effort,
	}
}

// readJSONBody decodes a request body into a generic map so individual fields can
// be asserted without a struct that silently swallows unexpected keys. It uses
// Errorf rather than Fatalf because it also runs inside httptest handlers.
func readJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read request body: %v", err)
		return map[string]any{}
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Errorf("request body is not JSON: %v (%s)", err, string(raw))
		return map[string]any{}
	}
	return body
}

// writeSSE writes pre-formatted "data: ..." lines followed by a blank line, the
// framing the OpenAI protocol and Ollama both use.
func writeSSE(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, _ := w.(http.Flusher)
	for _, line := range lines {
		fmt.Fprintf(w, "data: %s\n\n", line)
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// collectStream drains a streaming call and returns the joined text plus the
// first non-nil error.
func collectStream(t *testing.T, text <-chan string, errc <-chan error) (string, error) {
	t.Helper()
	var sb strings.Builder
	var streamErr error
	timeout := time.After(5 * time.Second)
	for text != nil || errc != nil {
		select {
		case chunk, ok := <-text:
			if !ok {
				text = nil
				continue
			}
			sb.WriteString(chunk)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			if err != nil && streamErr == nil {
				streamErr = err
			}
		case <-timeout:
			t.Fatal("timed out draining the stream")
		}
	}
	return sb.String(), streamErr
}

func TestCallTextSendsContractRequestAndHeaders(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotURL    string
		gotAuth   string
		gotType   string
		gotBody   map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotURL = r.URL.String()
		gotAuth = r.Header.Get("Authorization")
		gotType = r.Header.Get("Content-Type")
		gotBody = readJSONBody(t, r)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Xin chào"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`)
	}))
	defer server.Close()

	provider := trialProvider(server, trialEffort)
	var usage app_ai.TokenUsage
	content, err := CallText(context.Background(), server.Client(), provider, "system prompt", "user prompt",
		app_ai.TextGenerationOptions{
			MaxOutputTokens: 128,
			Temperature:     0,
			OnUsage:         func(u app_ai.TokenUsage) { usage = u },
		})
	if err != nil {
		t.Fatalf("CallText returned error: %v", err)
	}

	if content != "Xin chào" {
		t.Errorf("content = %q, want %q", content, "Xin chào")
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	// The adapter posts provider.Endpoint verbatim: it must not append a path.
	if gotPath != chatPath {
		t.Errorf("path = %q, want %q", gotPath, chatPath)
	}
	if gotURL != chatPath {
		t.Errorf("request URL = %q, want the endpoint path %q", gotURL, chatPath)
	}
	if gotAuth != "Bearer "+trialBearer {
		t.Errorf("Authorization = %q, want the Bearer token", gotAuth)
	}
	if gotType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotType)
	}

	if gotBody["model"] != trialModel {
		t.Errorf("model = %v, want %s", gotBody["model"], trialModel)
	}
	if gotBody["stream"] != false {
		t.Errorf("stream = %v, want false (sent explicitly)", gotBody["stream"])
	}
	if _, present := gotBody["temperature"]; !present {
		t.Fatal("temperature must be sent even when it is 0")
	}
	if gotBody["temperature"] != float64(0) {
		t.Errorf("temperature = %v, want 0", gotBody["temperature"])
	}
	if gotBody["max_tokens"] != float64(128) {
		t.Errorf("max_tokens = %v, want 128", gotBody["max_tokens"])
	}
	if gotBody["reasoning_effort"] != trialEffort {
		t.Errorf("reasoning_effort = %v, want %q (thinking off)", gotBody["reasoning_effort"], trialEffort)
	}

	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want system + user", gotBody["messages"])
	}
	if usage.PromptTokens != 11 || usage.OutputTokens != 7 || usage.FinishReason != "stop" {
		t.Errorf("usage = %+v, want 11/7/stop", usage)
	}
	if usage.ThinkingTokens != 0 {
		t.Errorf("thinking tokens = %d, want 0", usage.ThinkingTokens)
	}
}

func TestCallTextOmitsReasoningEffortWhenEmpty(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody = readJSONBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()

	// Empty AI_OPENAI_REASONING_EFFORT means the field must not be sent at all.
	if _, err := CallText(context.Background(), server.Client(), trialProvider(server, ""), "s", "u", app_ai.TextGenerationOptions{}); err != nil {
		t.Fatalf("CallText returned error: %v", err)
	}
	if _, present := gotBody["reasoning_effort"]; present {
		t.Errorf("reasoning_effort must be omitted when unset, body = %v", gotBody)
	}
}

func TestCallTextStreamsChunksInOrderUntilDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+trialBearer {
			t.Errorf("stream Authorization = %q, want the Bearer token", r.Header.Get("Authorization"))
		}
		if r.URL.Path != chatPath {
			t.Errorf("stream path = %q, want %q (endpoint posted verbatim)", r.URL.Path, chatPath)
		}
		body := readJSONBody(t, r)
		if body["stream"] != true {
			t.Errorf("stream = %v, want true", body["stream"])
		}
		if body["temperature"] != float64(0) {
			t.Errorf("stream temperature = %v, want 0 (sent, not skipped)", body["temperature"])
		}
		options, ok := body["stream_options"].(map[string]any)
		if !ok || options["include_usage"] != true {
			t.Errorf("stream_options = %v, want include_usage=true", body["stream_options"])
		}
		writeSSE(w,
			`{"choices":[{"delta":{"content":"Xin"}}]}`,
			`{"choices":[{"delta":{"content":" chào"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`[DONE]`,
		)
	}))
	defer server.Close()

	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
	got, err := collectStream(t, text, errc)
	if err != nil {
		t.Fatalf("stream returned error: %v", err)
	}
	if got != "Xin chào" {
		t.Errorf("streamed text = %q, want %q (in order, no duplication)", got, "Xin chào")
	}
}

// A provider that closes the connection after some content chunks but before
// sending [DONE] has truncated the answer. The contract treats [DONE] as the
// stream terminator, so EOF without it must surface as malformed (fallback
// eligible) rather than a successful, silently cut answer.
func TestCallTextStreamTruncatedWithoutDoneIsMalformed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			`{"choices":[{"delta":{"content":"Xin"}}]}`,
			`{"choices":[{"delta":{"content":" chào"}}]}`,
		)
		// The handler returns without writing [DONE]; the body ends here.
	}))
	defer server.Close()

	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
	got, err := collectStream(t, text, errc)
	if err == nil {
		t.Fatal("a stream that ends without [DONE] must report an error, not done")
	}
	if got != "Xin chào" {
		t.Errorf("streamed text = %q, want the chunks emitted before the truncation", got)
	}

	upstream, ok := AsUpstreamError(err)
	if !ok {
		t.Fatalf("error %T is not a typed UpstreamError: %v", err, err)
	}
	if upstream.Outcome != OutcomeMalformed {
		t.Errorf("outcome = %q, want malformed", upstream.Outcome)
	}
	if !upstream.FallbackEligible() {
		t.Error("a truncated stream must be eligible for the single fallback attempt")
	}
}

func TestCallTextStreamReportsUsageFromFinalChunkOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			`{"choices":[{"delta":{"content":"A"}}]}`,
			`{"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`,
			`[DONE]`,
		)
	}))
	defer server.Close()

	var (
		mu     sync.Mutex
		usages []app_ai.TokenUsage
	)
	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
		app_ai.TextGenerationOptions{OnUsage: func(u app_ai.TokenUsage) {
			mu.Lock()
			usages = append(usages, u)
			mu.Unlock()
		}})
	if _, err := collectStream(t, text, errc); err != nil {
		t.Fatalf("stream returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(usages) != 1 {
		t.Fatalf("usage callbacks = %d, want exactly 1", len(usages))
	}
	if usages[0].PromptTokens != 100 || usages[0].OutputTokens != 50 {
		t.Errorf("usage = %+v, want 100/50", usages[0])
	}
}

func TestCallTextStreamReportsZeroUsageWhenProviderOmitsIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			`{"choices":[{"delta":{"content":"A"}}]}`,
			`{"choices":[{"delta":{"content":"B"},"finish_reason":"stop"}]}`,
			`[DONE]`,
		)
	}))
	defer server.Close()

	var (
		calls  int
		latest app_ai.TokenUsage
	)
	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
		app_ai.TextGenerationOptions{OnUsage: func(u app_ai.TokenUsage) { calls++; latest = u }})
	if _, err := collectStream(t, text, errc); err != nil {
		t.Fatalf("stream returned error: %v", err)
	}

	// Ollama may send no usage at all; the winner stamp still has to fire so the
	// usage record names provider/model and usage_reported stays false.
	if calls != 1 {
		t.Fatalf("usage callbacks = %d, want 1 even without a usage payload", calls)
	}
	if latest.PromptTokens != 0 || latest.OutputTokens != 0 || latest.ThinkingTokens != 0 {
		t.Errorf("usage = %+v, want the zero value", latest)
	}
	if latest.FinishReason != "stop" {
		t.Errorf("finish reason = %q, want stop", latest.FinishReason)
	}
}

func TestCallTextStreamIgnoresReasoningContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			`{"choices":[{"delta":{"reasoning_content":"THINKING-SHOULD-NOT-LEAK","content":"Xin"}}]}`,
			`{"choices":[{"delta":{"thinking":"MORE-THINKING","content":" chào"},"finish_reason":"stop"}]}`,
			`[DONE]`,
		)
	}))
	defer server.Close()

	var usage app_ai.TokenUsage
	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
		app_ai.TextGenerationOptions{OnUsage: func(u app_ai.TokenUsage) { usage = u }})
	got, err := collectStream(t, text, errc)
	if err != nil {
		t.Fatalf("stream returned error: %v", err)
	}

	if got != "Xin chào" {
		t.Errorf("streamed text = %q, want only the content deltas", got)
	}
	if strings.Contains(got, "THINKING") || strings.Contains(got, "MORE-THINKING") {
		t.Errorf("reasoning text leaked into the answer: %q", got)
	}
	if usage.ThinkingTokens != 0 {
		t.Errorf("thinking tokens = %d, want 0 while thinking is off", usage.ThinkingTokens)
	}
}

func TestCallTextThinkingOffReturnsOnlyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"ok\":true}","reasoning_content":"THOUGHTS"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
	}))
	defer server.Close()

	content, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
		app_ai.TextGenerationOptions{ResponseMIMEType: "application/json"})
	if err != nil {
		t.Fatalf("CallText returned error: %v", err)
	}
	if content != `{"ok":true}` {
		t.Errorf("content = %q, want only the answer JSON", content)
	}
	if strings.Contains(content, "THOUGHTS") {
		t.Error("reasoning content must never be returned to the caller")
	}
}

func TestCallTextMaxTokensIsFallbackEligible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"truncated answer"},"finish_reason":"length"}],"usage":{"prompt_tokens":9,"completion_tokens":9,"total_tokens":18}}`)
	}))
	defer server.Close()

	content, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
	if err == nil {
		t.Fatal("finish_reason=length must be an error, not a successful truncated answer")
	}
	if content != "" {
		t.Errorf("content = %q, want empty so the cut answer is never shown", content)
	}

	upstream, ok := AsUpstreamError(err)
	if !ok {
		t.Fatalf("error %T is not a typed UpstreamError: %v", err, err)
	}
	if upstream.Outcome != OutcomeMaxTokens {
		t.Errorf("outcome = %q, want max_tokens", upstream.Outcome)
	}
	if !upstream.FallbackEligible() {
		t.Error("max_tokens must be eligible for the single fallback attempt")
	}
}

func TestCallTextStreamMaxTokensIsFallbackEligible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			`{"choices":[{"delta":{"content":"partial"},"finish_reason":"length"}]}`,
			`[DONE]`,
		)
	}))
	defer server.Close()

	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
	_, err := collectStream(t, text, errc)
	if err == nil {
		t.Fatal("a stream that ends on finish_reason=length must report an error")
	}
	upstream, ok := AsUpstreamError(err)
	if !ok || upstream.Outcome != OutcomeMaxTokens || !upstream.FallbackEligible() {
		t.Fatalf("error = %#v, want an eligible max_tokens outcome", err)
	}
}

func TestCallTextMalformedBodyIsFallbackEligible(t *testing.T) {
	cases := map[string]string{
		"invalid json":  `{"choices": [`,
		"no choices":    `{"choices":[]}`,
		"empty content": `{"choices":[{"message":{"content":""},"finish_reason":"stop"}]}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, body)
			}))
			defer server.Close()

			_, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
			if err == nil {
				t.Fatal("want an error for a malformed body")
			}
			upstream, ok := AsUpstreamError(err)
			if !ok {
				t.Fatalf("error %T is not typed: %v", err, err)
			}
			if upstream.Outcome != OutcomeMalformed {
				t.Errorf("outcome = %q, want malformed", upstream.Outcome)
			}
			if !upstream.FallbackEligible() {
				t.Error("malformed output must be eligible for the single fallback")
			}
		})
	}
}

func TestCallTextHTTPStatusClassification(t *testing.T) {
	cases := []struct {
		status   int
		outcome  Outcome
		eligible bool
	}{
		{http.StatusBadRequest, OutcomeHTTP4xx, false},
		{http.StatusUnauthorized, OutcomeHTTP4xx, false},
		{http.StatusForbidden, OutcomeHTTP4xx, false},
		{http.StatusNotFound, OutcomeHTTP4xx, false},
		{http.StatusUnprocessableEntity, OutcomeHTTP4xx, false},
		{http.StatusTooManyRequests, OutcomeHTTP429, true},
		{http.StatusInternalServerError, OutcomeHTTP5xx, true},
		{http.StatusBadGateway, OutcomeHTTP5xx, true},
		{http.StatusServiceUnavailable, OutcomeHTTP5xx, true},
		{http.StatusGatewayTimeout, OutcomeHTTP5xx, true},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "upstream said no", tc.status)
			}))
			defer server.Close()

			_, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
			if err == nil {
				t.Fatalf("status %d must produce an error", tc.status)
			}
			upstream, ok := AsUpstreamError(err)
			if !ok {
				t.Fatalf("status %d error %T is not typed: %v", tc.status, err, err)
			}
			if upstream.HTTPStatus != tc.status {
				t.Errorf("HTTPStatus = %d, want %d (status must survive the app error)", upstream.HTTPStatus, tc.status)
			}
			if upstream.Outcome != tc.outcome {
				t.Errorf("outcome = %q, want %q", upstream.Outcome, tc.outcome)
			}
			if upstream.FallbackEligible() != tc.eligible {
				t.Errorf("FallbackEligible = %v, want %v", upstream.FallbackEligible(), tc.eligible)
			}
		})
	}
}

func TestCallTextCancellationIsNotAFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := CallText(ctx, server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
	if err == nil {
		t.Fatal("a canceled call must return an error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestCallTextStreamCancellationStopsQuietly(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w, `{"choices":[{"delta":{"content":"Xin"}}]}`)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	text, errc := CallTextStream(ctx, server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})

	// Wait for the first chunk, then cancel mid-stream.
	select {
	case <-text:
	case <-time.After(5 * time.Second):
		t.Fatal("no chunk arrived before cancellation")
	}
	cancel()

	_, err := collectStream(t, text, errc)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want nil or context.Canceled after a client cancel", err)
	}
	// A real cancellation ends without [DONE] too; it must never be reclassified
	// as a malformed/fallback-eligible stream.
	if upstream, ok := AsUpstreamError(err); ok && upstream.Outcome == OutcomeMalformed {
		t.Fatalf("canceled stream was misclassified as malformed: %v", err)
	}
}

func TestCallTextEmptyStreamIsMalformed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w, `[DONE]`)
	}))
	defer server.Close()

	text, errc := CallTextStream(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u", app_ai.TextGenerationOptions{})
	_, err := collectStream(t, text, errc)
	if err == nil {
		t.Fatal("a stream with no content and no usage must be a malformed failure")
	}
	upstream, ok := AsUpstreamError(err)
	if !ok || upstream.Outcome != OutcomeMalformed || !upstream.FallbackEligible() {
		t.Fatalf("error = %#v, want an eligible malformed outcome", err)
	}
}

func TestCallTextStructuredJSONSchemaRequestAndValidation(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"summary", "tags"},
		"properties": map[string]any{
			"summary": map[string]any{"type": "string"},
			"tags":    map[string]any{"type": "array"},
		},
	}

	var gotFormat map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readJSONBody(t, r)
		gotFormat, _ = body["response_format"].(map[string]any)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"summary\":\"ok\",\"tags\":[\"a\"]}"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()

	content, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
		app_ai.TextGenerationOptions{
			ResponseMIMEType: "application/json",
			ResponseSchema:   schema,
		})
	if err != nil {
		t.Fatalf("CallText returned error: %v", err)
	}
	if !json.Valid([]byte(content)) {
		t.Errorf("content = %q, want valid JSON", content)
	}
	if gotFormat["type"] != "json_schema" {
		t.Errorf("response_format.type = %v, want json_schema", gotFormat["type"])
	}
	if gotFormat["json_schema"] == nil {
		t.Error("response_format.json_schema must carry the requested schema")
	}
}

func TestCallTextNormalizesSchemaTypesForOpenAI(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		want   map[string]any
	}{
		{
			name: "uppercase nested schema becomes lowercase",
			schema: map[string]any{
				"type":     "OBJECT",
				"required": []any{"summary", "tags", "meta"},
				"properties": map[string]any{
					"summary": map[string]any{"type": "STRING"},
					"tags": map[string]any{
						"type":  "ARRAY",
						"items": map[string]any{"type": "STRING"},
					},
					"meta": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"score": map[string]any{"type": "NUMBER"},
						},
					},
				},
			},
			want: map[string]any{
				"type":     "object",
				"required": []any{"summary", "tags", "meta"},
				"properties": map[string]any{
					"summary": map[string]any{"type": "string"},
					"tags": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"meta": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"score": map[string]any{"type": "number"},
						},
					},
				},
			},
		},
		{
			name: "lowercase schema is unchanged",
			schema: map[string]any{
				"type":     "object",
				"required": []any{"summary"},
				"properties": map[string]any{
					"summary": map[string]any{"type": "string"},
				},
			},
			want: map[string]any{
				"type":     "object",
				"required": []any{"summary"},
				"properties": map[string]any{
					"summary": map[string]any{"type": "string"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSchema any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := readJSONBody(t, r)
				format, _ := body["response_format"].(map[string]any)
				jsonSchema, _ := format["json_schema"].(map[string]any)
				gotSchema = jsonSchema["schema"]

				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"summary\":\"ok\",\"tags\":[],\"meta\":{\"score\":1}}"},"finish_reason":"stop"}]}`)
			}))
			defer server.Close()

			_, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
				app_ai.TextGenerationOptions{
					ResponseMIMEType: "application/json",
					ResponseSchema:   tt.schema,
				})
			if err != nil {
				t.Fatalf("CallText returned error: %v", err)
			}

			gotJSON, err := json.Marshal(gotSchema)
			if err != nil {
				t.Fatalf("marshal outgoing schema: %v", err)
			}
			wantJSON, err := json.Marshal(tt.want)
			if err != nil {
				t.Fatalf("marshal expected schema: %v", err)
			}
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("outgoing schema = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestCallTextStructuredValidationFailures(t *testing.T) {
	schema := map[string]any{"type": "object", "required": []string{"summary"}}

	cases := map[string]string{
		"not json":        `not-json-at-all`,
		"missing key":     `{"other":"value"}`,
		"json not object": `["summary"]`,
	}

	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			encoded, marshalErr := json.Marshal(map[string]any{
				"choices": []any{map[string]any{
					"message":       map[string]any{"content": payload},
					"finish_reason": "stop",
				}},
			})
			if marshalErr != nil {
				t.Fatalf("marshal fixture: %v", marshalErr)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(encoded)
			}))
			defer server.Close()

			_, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
				app_ai.TextGenerationOptions{ResponseMIMEType: "application/json", ResponseSchema: schema})
			if err == nil {
				t.Fatal("invalid structured output must not be returned as content")
			}
			upstream, ok := AsUpstreamError(err)
			if !ok {
				t.Fatalf("error %T is not typed: %v", err, err)
			}
			if upstream.Outcome != OutcomeSchemaInvalid {
				t.Errorf("outcome = %q, want schema_invalid", upstream.Outcome)
			}
			if !upstream.FallbackEligible() {
				t.Error("schema_invalid must be eligible for the single fallback")
			}
		})
	}
}

func TestCallTextJSONObjectFormatWhenNoSchema(t *testing.T) {
	var gotFormat map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readJSONBody(t, r)
		gotFormat, _ = body["response_format"].(map[string]any)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"ok\":true}"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()

	if _, err := CallText(context.Background(), server.Client(), trialProvider(server, trialEffort), "s", "u",
		app_ai.TextGenerationOptions{ResponseMIMEType: "application/json"}); err != nil {
		t.Fatalf("CallText returned error: %v", err)
	}
	if gotFormat["type"] != "json_object" {
		t.Errorf("response_format.type = %v, want json_object when no schema is requested", gotFormat["type"])
	}
}

func TestSSEParserUsesDataLinesOnly(t *testing.T) {
	stream := strings.Join([]string{
		": keep-alive comment",
		"event: ping",
		`data: {"choices":[{"delta":{"content":"A"}}]}`,
		"",
		`data: [DONE]`,
		"",
	}, "\n")

	var seen []string
	err := parseSSEStream(bufio.NewReader(strings.NewReader(stream)), func(data string) error {
		seen = append(seen, data)
		return nil
	})
	if err != nil {
		t.Fatalf("parseSSEStream returned error: %v", err)
	}
	if len(seen) != 2 || seen[0] != `{"choices":[{"delta":{"content":"A"}}]}` || seen[1] != "[DONE]" {
		t.Errorf("parsed data lines = %v, want only the two data payloads", seen)
	}
}

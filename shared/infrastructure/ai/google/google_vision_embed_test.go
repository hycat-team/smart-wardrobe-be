package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"smart-wardrobe-be/shared/config"
)

// visionFixture serves a tiny image and a staged generateContent response.
func visionFixture(t *testing.T, generateContent any) (imageURL string, provider config.APIProviderConfig, cleanup func()) {
	t.Helper()

	imageSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0x00, 0x01, 0x02})
	}))
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(generateContent)
	}))

	provider = config.APIProviderConfig{
		Endpoint: apiSrv.URL,
		ApiKey:   "test-key",
		Model:    "gemini-2.0-flash",
	}
	return imageSrv.URL + "/img.jpg",
		provider,
		func() { imageSrv.Close(); apiSrv.Close() }
}

func TestCallVisionWithUsageParsesUsageMetadata(t *testing.T) {
	imageURL, provider, cleanup := visionFixture(t, map[string]any{
		"candidates": []map[string]any{
			{
				"finishReason": "STOP",
				"content": map[string]any{
					"parts": []map[string]any{{"text": `{"ok":true}`}},
				},
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     120,
			"candidatesTokenCount": 30,
			"thoughtsTokenCount":   5,
			"totalTokenCount":      155,
		},
	})
	defer cleanup()

	text, usage, err := CallVisionWithUsage(context.Background(), http.DefaultClient, provider, imageURL, "describe")
	if err != nil {
		t.Fatalf("CallVisionWithUsage: %v", err)
	}
	if text != `{"ok":true}` {
		t.Errorf("text = %q, want the candidate text", text)
	}
	if usage.PromptTokens != 120 || usage.OutputTokens != 30 || usage.ThinkingTokens != 5 {
		t.Errorf("usage = %+v, want prompt=120 output=30 thinking=5", usage)
	}
	if usage.FinishReason != "STOP" {
		t.Errorf("finish reason = %q, want STOP", usage.FinishReason)
	}
}

func TestCallVisionWithUsageWithoutMetadataReturnsZeroUsage(t *testing.T) {
	imageURL, provider, cleanup := visionFixture(t, map[string]any{
		"candidates": []map[string]any{
			{
				"finishReason": "STOP",
				"content": map[string]any{
					"parts": []map[string]any{{"text": `{"ok":true}`}},
				},
			},
		},
	})
	defer cleanup()

	text, usage, err := CallVisionWithUsage(context.Background(), http.DefaultClient, provider, imageURL, "describe")
	if err != nil {
		t.Fatalf("CallVisionWithUsage: %v", err)
	}
	if text == "" {
		t.Error("text must still be returned when usage is absent")
	}
	if usage.PromptTokens != 0 || usage.OutputTokens != 0 || usage.ThinkingTokens != 0 {
		t.Errorf("usage without metadata must be zero, got %+v", usage)
	}
}

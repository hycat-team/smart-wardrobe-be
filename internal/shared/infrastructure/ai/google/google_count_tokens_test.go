package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/config"
)

func TestCountGoogleTokensByRequest(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedTokens int64
		expectedError  error
	}{
		{
			name: "Success path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "countTokens") {
					t.Errorf("expected path to contain countTokens, got %s", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"totalTokens": 42,
				})
			},
			expectedTokens: 42,
			expectedError:  nil,
		},
		{
			name: "Bad request - 400",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensInvalidRequest,
		},
		{
			name: "Unauthorized - 401",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensAuthFailed,
		},
		{
			name: "Model not found - 404",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensModelNotFound,
		},
		{
			name: "Rate limit - 429",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensUnavailable,
		},
		{
			name: "Server error - 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensUnavailable,
		},
		{
			name: "Malformed successful response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{invalid-json}"))
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensInvalidResponse,
		},
		{
			name: "Negative totalTokens response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"totalTokens": -5,
				})
			},
			expectedTokens: 0,
			expectedError:  ErrCountTokensInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			provider := config.APIProviderConfig{
				Endpoint: server.URL,
				ApiKey:   "test-key",
				Model:    "gemini-2.0-flash",
			}

			req := PreparedGeminiRequest{
				Model: "gemini-2.0-flash",
				Body:  GeminiGenerateContentBody{},
			}

			tokens, _, err := CountGoogleTokensByRequest(context.Background(), http.DefaultClient, nil, provider, req)
			if tt.expectedError != nil {
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if tokens != tt.expectedTokens {
					t.Errorf("expected %d tokens, got %d", tt.expectedTokens, tokens)
				}
			}
		})
	}
}

func TestCountGoogleTokensByRequest_Retry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests) // 429 (retryable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"totalTokens": 100,
		})
	}))
	defer server.Close()

	provider := config.APIProviderConfig{
		Endpoint: server.URL,
		ApiKey:   "test-key",
		Model:    "gemini-2.0-flash",
	}

	req := PreparedGeminiRequest{
		Model: "gemini-2.0-flash",
		Body:  GeminiGenerateContentBody{},
	}

	tokens, _, err := CountGoogleTokensByRequest(context.Background(), http.DefaultClient, nil, provider, req)
	if err != nil {
		t.Errorf("expected success after retry, got %v", err)
	}
	if tokens != 100 {
		t.Errorf("expected 100 tokens, got %d", tokens)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

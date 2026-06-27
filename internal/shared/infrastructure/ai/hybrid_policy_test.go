package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockLogger struct{}

func (m *mockLogger) Debug(message string, fields ...zap.Field) {}
func (m *mockLogger) Info(message string, fields ...zap.Field)  {}
func (m *mockLogger) Warn(message string, fields ...zap.Field)  {}
func (m *mockLogger) Error(message string, fields ...zap.Field) {}
func (m *mockLogger) Fatal(message string, fields ...zap.Field) {}

var _ logger.Interface = (*mockLogger)(nil)

// MockCostPolicy implements contract.IAICostPolicyContract for integration testing
type MockCostPolicy struct {
	PrepareFunc       func(ctx context.Context, userID uuid.UUID, operation string, meta contract.AITokenEstimationMeta) (*contract.AICostDecision, error)
	PrepareFreeFunc   func(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*contract.AICostDecision, error)
	MarkInFlightFunc  func(ctx context.Context, requestID uuid.UUID) error
	ConfirmFunc       func(ctx context.Context, requestID uuid.UUID, usage contract.AIUsage) error
	ReleaseFunc       func(ctx context.Context, requestID uuid.UUID, reason string) error
	MarkUnknownFunc   func(ctx context.Context, requestID uuid.UUID, reason string) error
	ExpireUnknownFunc func(ctx context.Context, now time.Time, limit int) (int64, error)
}

func (m *MockCostPolicy) Prepare(ctx context.Context, userID uuid.UUID, operation string, meta contract.AITokenEstimationMeta) (*contract.AICostDecision, error) {
	if m.PrepareFunc != nil {
		return m.PrepareFunc(ctx, userID, operation, meta)
	}
	return &contract.AICostDecision{RequestID: uuid.New(), Route: "paid", MaxInputTokens: 12000, MaxOutputTokens: 500, Tracked: true}, nil
}

func (m *MockCostPolicy) PrepareFree(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*contract.AICostDecision, error) {
	if m.PrepareFreeFunc != nil {
		return m.PrepareFreeFunc(ctx, userID, operation, promptTokens, maxOutputTokens)
	}
	return &contract.AICostDecision{RequestID: uuid.New(), Route: "free", MaxInputTokens: 0, MaxOutputTokens: maxOutputTokens, Tracked: false}, nil
}

func (m *MockCostPolicy) MarkInFlight(ctx context.Context, requestID uuid.UUID) error {
	if m.MarkInFlightFunc != nil {
		return m.MarkInFlightFunc(ctx, requestID)
	}
	return nil
}

func (m *MockCostPolicy) Confirm(ctx context.Context, requestID uuid.UUID, usage contract.AIUsage) error {
	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(ctx, requestID, usage)
	}
	return nil
}

func (m *MockCostPolicy) Release(ctx context.Context, requestID uuid.UUID, reason string) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(ctx, requestID, reason)
	}
	return nil
}

func (m *MockCostPolicy) MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error {
	if m.MarkUnknownFunc != nil {
		return m.MarkUnknownFunc(ctx, requestID, reason)
	}
	return nil
}

func (m *MockCostPolicy) ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error) {
	if m.ExpireUnknownFunc != nil {
		return m.ExpireUnknownFunc(ctx, now, limit)
	}
	return 0, nil
}

func TestHybridPolicyIntegration(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIServiceConfig{
			TokenEstimation: config.TokenEstimationConfig{
				CharsPerToken:             4.0,
				LocalSafetyMultiplier:     1.25,
				CountTokensThresholdRatio: 0.70,
				CountTokensTimeout:        500 * time.Millisecond,
			},
		},
	}
	estimator := NewLocalTokenEstimator(4.0, 1.25)
	
	t.Run("Fast path - Under threshold", func(t *testing.T) {
		policy := &MockCostPolicy{
			PrepareFunc: func(ctx context.Context, userID uuid.UUID, operation string, meta contract.AITokenEstimationMeta) (*contract.AICostDecision, error) {
				if meta.TokenEstimationMethod != contract.TokenEstimationLocal {
					t.Errorf("expected LOCAL estimation method, got %s", meta.TokenEstimationMethod)
				}
				if meta.TokenCountLatencyMs != nil {
					t.Errorf("expected latency to be nil on fast path")
				}
				return &contract.AICostDecision{RequestID: uuid.New(), Route: "paid", MaxInputTokens: 12000, MaxOutputTokens: 500, Tracked: true}, nil
			},
		}

		svc := &AIService{
			cfg:            cfg,
			chatTextClient: http.DefaultClient,
			costPolicy:     policy,
			estimator:      estimator,
			logger:         &mockLogger{},
		}

		paid := config.APIProviderConfig{
			Provider: ProviderGemini,
			Model:    "gemini-2.0-flash",
		}

		_, _, _, _, err := svc.prepareText(
			context.Background(),
			paid,
			"System instruction",
			"Short prompt",
			app_ai.TextGenerationOptions{UserID: uuid.New(), Operation: "chat"},
			"chat",
		)
		if err != nil {
			t.Errorf("unexpected error on fast path: %v", err)
		}
	})

	t.Run("Slow path - Above threshold and calls countTokens", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"totalTokens": 9500,
			})
		}))
		defer server.Close()

		paid := config.APIProviderConfig{
			Provider: ProviderGemini,
			Endpoint: server.URL,
			ApiKey:   "test-key",
			Model:    "gemini-2.0-flash",
		}

		policy := &MockCostPolicy{
			PrepareFunc: func(ctx context.Context, userID uuid.UUID, operation string, meta contract.AITokenEstimationMeta) (*contract.AICostDecision, error) {
				if meta.TokenEstimationMethod != contract.TokenEstimationProviderCount {
					t.Errorf("expected PROVIDER_COUNT estimation method, got %s", meta.TokenEstimationMethod)
				}
				if meta.TokenCountLatencyMs == nil {
					t.Errorf("expected non-nil latency on slow path")
				}
				if meta.EstimatedPromptTokens != 9500 {
					t.Errorf("expected 9500 prompt tokens, got %d", meta.EstimatedPromptTokens)
				}
				return &contract.AICostDecision{RequestID: uuid.New(), Route: "paid", MaxInputTokens: 12000, MaxOutputTokens: 500, Tracked: true}, nil
			},
		}

		svc := &AIService{
			cfg:            cfg,
			chatTextClient: http.DefaultClient,
			costPolicy:     policy,
			estimator:      estimator,
			logger:         &mockLogger{},
		}

		// Very long text prompt to exceed 70% threshold (around 8400 tokens / 33600 characters)
		longPrompt := string(make([]rune, 35000))

		_, _, _, _, err := svc.prepareText(
			context.Background(),
			paid,
			"System instruction",
			longPrompt,
			app_ai.TextGenerationOptions{UserID: uuid.New(), Operation: "chat"},
			"chat",
		)
		if err != nil {
			t.Errorf("unexpected error on slow path: %v", err)
		}
	})
}

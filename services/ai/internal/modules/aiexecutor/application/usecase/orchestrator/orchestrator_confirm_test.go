package orchestrator

import (
	"context"
	"testing"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// confirmCaptureCostUC records the AIUsage Confirm receives.
type confirmCaptureCostUC struct {
	uc_interfaces.IAICostUseCase

	confirmUsage uc_interfaces.AIUsage
	confirmCalls int
	confirmErr   error
}

func (f *confirmCaptureCostUC) Prepare(_ context.Context, plan *uc_interfaces.ExecutionPlan) (*uc_interfaces.AICostDecision, error) {
	return &uc_interfaces.AICostDecision{RequestID: plan.RequestID, Route: plan.Route, Tracked: true}, nil
}

func (f *confirmCaptureCostUC) Confirm(_ context.Context, _ uuid.UUID, usage uc_interfaces.AIUsage) error {
	f.confirmUsage = usage
	f.confirmCalls++
	return f.confirmErr
}

func (f *confirmCaptureCostUC) MarkUnknown(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// stubAIService answers vision/embedding calls with staged results.
type stubAIService struct {
	app_ai.IAIService

	embeddings [][]float32
	embedErr   error
	visionText string
	visionUse  app_ai.TokenUsage
	visionErr  error

	analyzeWithUsageCalls int
}

func (f *stubAIService) GenerateEmbeddings(_ context.Context, _ []string) ([][]float32, error) {
	return f.embeddings, f.embedErr
}

func (f *stubAIService) AnalyzeImage(_ context.Context, _ string, _ string) (string, error) {
	return f.visionText, f.visionErr
}

func (f *stubAIService) AnalyzeImageWithUsage(_ context.Context, _ string, _ string) (string, app_ai.TokenUsage, error) {
	f.analyzeWithUsageCalls++
	return f.visionText, f.visionUse, f.visionErr
}

func (f *stubAIService) EstimatePromptTokens(context.Context, string, string, string) (app_ai.TokenEstimationMeta, error) {
	return app_ai.TokenEstimationMeta{}, nil
}

func (f *stubAIService) GenerateChatText(context.Context, string, string, app_ai.TextGenerationOptions) (string, error) {
	return "", nil
}

func (f *stubAIService) GenerateRecommendationText(context.Context, string, string, app_ai.TextGenerationOptions) (string, error) {
	return "", nil
}

func (f *stubAIService) GenerateChatTextStream(context.Context, string, string, app_ai.TextGenerationOptions) (<-chan string, <-chan error) {
	text := make(chan string)
	errs := make(chan error)
	close(text)
	close(errs)
	return text, errs
}

func newEmbeddingUC(ai *stubAIService, cost *confirmCaptureCostUC) *OrchestratorUseCase {
	log := logger.New("test", "", "info", false)
	cfg := &config.Config{}
	cfg.AI.VisionPrimary.Provider = "google"
	cfg.AI.VisionPrimary.Model = "gemini-2.0-flash"
	cfg.AI.EmbeddingPrimary.Provider = "google"
	cfg.AI.EmbeddingPrimary.Model = "gemini-embedding-001"
	cfg.AI.FreeTextPrimary.Provider = "google"
	cfg.AI.FreeTextPrimary.Model = "gemini-2.0-flash"
	cfg.AI.DefaultCostPolicyCode = "free-default"
	// Vision and embedding are always-free operations, so resolveExecutionPlan
	// returns before touching repo/fallback/entitlements; nil is safe here.
	return &OrchestratorUseCase{
		aiService: ai,
		cfg:       cfg,
		costUC:    cost,
		logger:    log,
	}
}

// GenerateEmbeddings has no provider usage signal (spike T001: countTokens 400
// on both embedding models), so it confirms with zero tokens and fails open to
// usage_unreported while the user request still succeeds.
func TestGenerateEmbeddingsConfirmsZeroTokensAndSucceeds(t *testing.T) {
	ai := &stubAIService{embeddings: [][]float32{{0.1, 0.2}}}
	cost := &confirmCaptureCostUC{}
	uc := newEmbeddingUC(ai, cost)

	out, err := uc.GenerateEmbeddings(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("GenerateEmbeddings: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(out))
	}
	if cost.confirmCalls != 1 {
		t.Fatalf("expected exactly 1 Confirm call, got %d", cost.confirmCalls)
	}
	total := cost.confirmUsage.PromptTokens + cost.confirmUsage.OutputTokens + cost.confirmUsage.ThinkingTokens
	if total != 0 {
		t.Errorf("embedding Confirm must carry zero tokens (fail-open usage_unreported), got %+v", cost.confirmUsage)
	}
	if cost.confirmUsage.Provider == "" || cost.confirmUsage.Model == "" {
		t.Errorf("embedding Confirm must still carry provider/model, got %+v", cost.confirmUsage)
	}
}

// AnalyzeImage must confirm with the real provider usage, not zero tokens.
func TestAnalyzeImageConfirmsRealUsage(t *testing.T) {
	ai := &stubAIService{
		visionText: `{"ok":true}`,
		visionUse:  app_ai.TokenUsage{PromptTokens: 120, OutputTokens: 30, ThinkingTokens: 5, FinishReason: "STOP"},
	}
	cost := &confirmCaptureCostUC{}
	uc := newEmbeddingUC(ai, cost)

	text, err := uc.AnalyzeImage(context.Background(), "http://img/x.jpg", "describe")
	if err != nil {
		t.Fatalf("AnalyzeImage: %v", err)
	}
	if text == "" {
		t.Fatal("expected vision text")
	}
	if ai.analyzeWithUsageCalls != 1 {
		t.Fatalf("AnalyzeImage must call AnalyzeImageWithUsage, got %d calls", ai.analyzeWithUsageCalls)
	}
	if cost.confirmUsage.PromptTokens != 120 || cost.confirmUsage.OutputTokens != 30 || cost.confirmUsage.ThinkingTokens != 5 {
		t.Errorf("vision Confirm must carry real usage, got %+v", cost.confirmUsage)
	}
	if cost.confirmUsage.FinishReason != "STOP" {
		t.Errorf("vision Confirm must carry finish reason, got %+v", cost.confirmUsage)
	}
	if cost.confirmUsage.Unbilled {
		t.Errorf("vision must stay billed unless listed in AI_UNBILLED_CAPABILITIES, got %+v", cost.confirmUsage)
	}
}

// Embedding runs on a key the team treats as free, so the billing flag must
// follow the configured capability list rather than the logical route.
func TestGenerateEmbeddingsMarksUnbilledWhenConfigured(t *testing.T) {
	ai := &stubAIService{embeddings: [][]float32{{0.1, 0.2}}}
	cost := &confirmCaptureCostUC{}
	uc := newEmbeddingUC(ai, cost)
	uc.cfg.AI.Billing.UnbilledCapabilities = []string{config.CapabilityEmbedding}

	if _, err := uc.GenerateEmbeddings(context.Background(), []string{"hello"}); err != nil {
		t.Fatalf("GenerateEmbeddings: %v", err)
	}
	if !cost.confirmUsage.Unbilled {
		t.Errorf("embedding must be marked unbilled when configured, got %+v", cost.confirmUsage)
	}
}

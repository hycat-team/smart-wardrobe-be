package cost

import (
	"context"
	"errors"
	"testing"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// confirmCaptureRepo records the arguments Confirm receives.
type confirmCaptureRepo struct {
	repositories.IAICostRepository

	prompt   int64
	output   int64
	thinking int64
	cost     int64
	provider string
	model    string
	finish   string
	errCode  string
	calls    int
}

func (r *confirmCaptureRepo) Confirm(_ context.Context, _ uuid.UUID, prompt, output, thinking, cost int64, provider, model, finish, errorCode string, _ time.Time) error {
	r.prompt = prompt
	r.output = output
	r.thinking = thinking
	r.cost = cost
	r.provider = provider
	r.model = model
	r.finish = finish
	r.errCode = errorCode
	r.calls++
	return nil
}

// stubPricingUC answers GetActivePricingConfig with a staged result.
type stubPricingUC struct {
	uc_interfaces.IPricingUseCase

	cfg *aiexecutor_dto.PricingConfigDTO
	err error

	gotProvider string
	gotModel    string
}

func (s *stubPricingUC) GetActivePricingConfig(_ context.Context, provider, model string) (*aiexecutor_dto.PricingConfigDTO, error) {
	s.gotProvider = provider
	s.gotModel = model
	if s.err != nil {
		return nil, s.err
	}
	return s.cfg, nil
}

func testPricing() *aiexecutor_dto.PricingConfigDTO {
	return &aiexecutor_dto.PricingConfigDTO{
		ID:                  uuid.New(),
		Provider:            "google",
		Model:               "gemini-2.0-flash",
		InputUSDPerMillion:  decimal.NewFromFloat(1),
		OutputUSDPerMillion: decimal.NewFromFloat(2),
		USDToVND:            decimal.NewFromInt(25000),
	}
}

func newConfirmUC(repo *confirmCaptureRepo, pricing *stubPricingUC) *CostUseCase {
	return &CostUseCase{repo: repo, pricingUC: pricing}
}

// (1) Paid request keeps the computed actual cost.
func TestConfirmPaidKeepsComputedCost(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{cfg: testPricing()}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{PromptTokens: 100, OutputTokens: 40, ThinkingTokens: 10, FinishReason: "STOP", Provider: "google", Model: "gemini-2.0-flash"}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	// 100*1*25000 + 50*2*25000 = 5_000_000.
	if repo.cost != 5000000 {
		t.Errorf("paid actual cost = %d, want 5000000", repo.cost)
	}
	if repo.errCode != "" {
		t.Errorf("paid error code = %q, want empty", repo.errCode)
	}
}

// A free route that still runs on a billed key (vision, or a capability left out
// of AI_UNBILLED_CAPABILITIES) keeps recording the real provider cost.
func TestConfirmFreeRouteWithBilledKeyRecordsPositiveActual(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{cfg: testPricing()}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{PromptTokens: 10, OutputTokens: 5, ThinkingTokens: 0, Provider: "google", Model: "gemini-2.0-flash"}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if repo.cost <= 0 {
		t.Errorf("free route billed key actual cost = %d, want > 0", repo.cost)
	}
	if repo.errCode != "" {
		t.Errorf("free route billed key error code = %q, want empty", repo.errCode)
	}
	if pricing.gotProvider != "google" || pricing.gotModel != "gemini-2.0-flash" {
		t.Errorf("pricing lookup got %q/%q, want google/gemini-2.0-flash", pricing.gotProvider, pricing.gotModel)
	}
}

// An unbilled capability records zero cost and never reaches pricing: the free
// key is a separate quota, so a missing price must not fail or misprice it.
func TestConfirmUnbilledSkipsPricingAndRecordsZero(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{err: errors.New("pricing must not be consulted")}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{
		PromptTokens:   10,
		OutputTokens:   5,
		ThinkingTokens: 0,
		FinishReason:   "STOP",
		Provider:       "google",
		Model:          "gemini-3.5-flash-lite",
		Unbilled:       true,
	}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if repo.cost != 0 {
		t.Errorf("unbilled actual cost = %d, want 0", repo.cost)
	}
	if repo.errCode != "" {
		t.Errorf("unbilled error code = %q, want empty", repo.errCode)
	}
	if repo.prompt != 10 || repo.output != 5 {
		t.Errorf("unbilled must keep reported tokens, got prompt=%d output=%d", repo.prompt, repo.output)
	}
	if pricing.gotProvider != "" || pricing.gotModel != "" {
		t.Errorf("unbilled must not hit pricing lookup, got %q/%q", pricing.gotProvider, pricing.gotModel)
	}
}

// provider=openai is always cost 0, even when the caller fails to pass the
// Unbilled flag. The guard is on the provider, not on the flag, so a fallback
// path that forgot to stamp cannot misprice an Ollama call.
func TestConfirmOpenAICostsZeroEvenWhenUnbilledFlagIsFalse(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{err: errors.New("pricing must not be consulted for provider=openai")}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{
		PromptTokens:   100,
		OutputTokens:   40,
		ThinkingTokens: 10,
		FinishReason:   "STOP",
		Provider:       "openai",
		Model:          "qwen3:4b",
		Unbilled:       false,
	}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if repo.cost != 0 {
		t.Errorf("openai actual cost = %d, want 0", repo.cost)
	}
	if repo.errCode != "" {
		t.Errorf("openai error code = %q, want empty", repo.errCode)
	}
	if repo.provider != "openai" || repo.model != "qwen3:4b" {
		t.Errorf("openai confirm got %q/%q, want openai/qwen3:4b", repo.provider, repo.model)
	}
	if pricing.gotProvider != "" || pricing.gotModel != "" {
		t.Errorf("openai must not hit pricing lookup, got %q/%q", pricing.gotProvider, pricing.gotModel)
	}
}

// (3) Non-empty provider/model without pricing row reconciles with a code.
func TestConfirmMissingPricingReportsPricingConfigMissing(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{err: errors.New("no active pricing config")}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{PromptTokens: 10, OutputTokens: 5, ThinkingTokens: 0, FinishReason: "STOP", Provider: "google", Model: "unknown-model"}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if repo.errCode != errorCodePricingConfigMissing {
		t.Errorf("error code = %q, want %q", repo.errCode, errorCodePricingConfigMissing)
	}
	if repo.cost != 0 {
		t.Errorf("missing-pricing cost = %d, want 0", repo.cost)
	}
	if repo.prompt != 10 || repo.output != 5 {
		t.Errorf("missing-pricing must preserve reported tokens, got prompt=%d output=%d", repo.prompt, repo.output)
	}
}

// (4) Empty provider and model reconciles as missing pricing.
func TestConfirmEmptyProviderModelReportsPricingConfigMissing(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{cfg: testPricing()}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{PromptTokens: 10, OutputTokens: 5, Provider: "", Model: ""}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if repo.errCode != errorCodePricingConfigMissing {
		t.Errorf("error code = %q, want %q", repo.errCode, errorCodePricingConfigMissing)
	}
	if repo.cost != 0 {
		t.Errorf("empty provider/model cost = %d, want 0", repo.cost)
	}
	if pricing.gotProvider != "" || pricing.gotModel != "" {
		t.Errorf("empty provider/model must not hit pricing lookup, got %q/%q", pricing.gotProvider, pricing.gotModel)
	}
}

// (5) Priced model with zero tokens reconciles as unreported usage.
func TestConfirmZeroTokensReportsUsageUnreported(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{cfg: testPricing()}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{PromptTokens: 0, OutputTokens: 0, ThinkingTokens: 0, Provider: "google", Model: "gemini-2.0-flash"}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if repo.errCode != errorCodeUsageUnreported {
		t.Errorf("error code = %q, want %q", repo.errCode, errorCodeUsageUnreported)
	}
	if repo.cost != 0 {
		t.Errorf("zero-token cost = %d, want 0", repo.cost)
	}
}

// Confirm always uses the provider/model it is given and never re-resolves
// them (locks the R7 fallback exception: a fallback win must not rewrite the
// price identity of the event).
func TestConfirmUsesPassedProviderModelVerbatim(t *testing.T) {
	repo := &confirmCaptureRepo{}
	pricing := &stubPricingUC{cfg: testPricing()}
	uc := newConfirmUC(repo, pricing)

	usage := uc_interfaces.AIUsage{PromptTokens: 8, OutputTokens: 4, ThinkingTokens: 1, FinishReason: "STOP", Provider: "google", Model: "gemini-embedding-001"}
	if err := uc.Confirm(context.Background(), uuid.New(), usage); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if pricing.gotProvider != "google" || pricing.gotModel != "gemini-embedding-001" {
		t.Errorf("pricing lookup got %q/%q, want google/gemini-embedding-001", pricing.gotProvider, pricing.gotModel)
	}
	if repo.provider != "google" || repo.model != "gemini-embedding-001" {
		t.Errorf("repo confirm got %q/%q, want google/gemini-embedding-001", repo.provider, repo.model)
	}
}

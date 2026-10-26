package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

type trialAIMode string

const (
	trialModeFallback      trialAIMode = "fallback"
	trialModeCancel        trialAIMode = "cancel"
	trialModeHTTP4xx       trialAIMode = "http_4xx"
	trialModeOpenAISuccess trialAIMode = "openai_success"
)

// trialPlanRepo makes resolveExecutionPlan return a paid text plan without a
// database. The grant carries no snapshot on purpose: the reader falls back to
// the live operation row, which is exactly the path those tests do not target.
type trialPlanRepo struct {
	repositories.IAICostRepository
	grant  *aiexecutor_entities.AIPolicyGrant
	policy *aiexecutor_entities.AICostPolicy
	op     *aiexecutor_entities.AICostPolicyOperation
}

func (r *trialPlanRepo) ResolvePolicy(context.Context, uuid.UUID, aioperation.AIOperation, time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, error) {
	return r.grant, r.policy, r.op, nil
}

type trialCostUC struct {
	uc_interfaces.IAICostUseCase

	order *[]string

	admitRoute             airoute.AIRoute
	admitCalls             int
	admitRequestIDs        []uuid.UUID
	prepareBillableCalls   int
	prepareBillableRequest []uuid.UUID
	prepareBillableModel   []string

	releaseCalls int
	confirmCalls int
	confirmUsage uc_interfaces.AIUsage
}

func (f *trialCostUC) Prepare(_ context.Context, plan *uc_interfaces.ExecutionPlan) (*uc_interfaces.AICostDecision, error) {
	return &uc_interfaces.AICostDecision{RequestID: plan.RequestID, Route: plan.Route, Tracked: true}, nil
}

func (f *trialCostUC) MarkInFlight(context.Context, uuid.UUID) error { return nil }

func (f *trialCostUC) AdmitUnbilledAttempt(_ context.Context, admission uc_interfaces.UnbilledAttemptAdmission) (*uc_interfaces.AICostDecision, error) {
	f.admitCalls++
	f.admitRequestIDs = append(f.admitRequestIDs, admission.Plan.RequestID)
	f.appendOrder("admit")
	route := f.admitRoute
	if route == "" {
		route = admission.Plan.Route
	}
	return &uc_interfaces.AICostDecision{
		RequestID: admission.Plan.RequestID,
		Route:     route,
		Tracked:   true,
	}, nil
}

func (f *trialCostUC) PrepareBillableAttempt(_ context.Context, admission uc_interfaces.BillableAttemptAdmission) (*uc_interfaces.AICostDecision, error) {
	f.prepareBillableCalls++
	f.prepareBillableRequest = append(f.prepareBillableRequest, admission.Plan.RequestID)
	f.prepareBillableModel = append(f.prepareBillableModel, admission.Provider+"/"+admission.Model)
	f.appendOrder("prepare_billable")
	return &uc_interfaces.AICostDecision{
		RequestID: admission.Plan.RequestID,
		Route:     admission.Plan.Route,
		Tracked:   true,
	}, nil
}

func (f *trialCostUC) Confirm(_ context.Context, _ uuid.UUID, usage uc_interfaces.AIUsage) error {
	f.confirmCalls++
	f.confirmUsage = usage
	f.appendOrder("confirm")
	return nil
}

func (f *trialCostUC) Release(context.Context, uuid.UUID, string) error {
	f.releaseCalls++
	f.appendOrder("release")
	return nil
}

func (f *trialCostUC) appendOrder(step string) {
	if f.order != nil {
		*f.order = append(*f.order, step)
	}
}

type trialAIService struct {
	app_ai.IAIService

	mode  trialAIMode
	order *[]string
}

func (f *trialAIService) EstimatePromptTokens(context.Context, string, string, string) (app_ai.TokenEstimationMeta, error) {
	return app_ai.TokenEstimationMeta{EstimatedPromptTokens: 100}, nil
}

func (f *trialAIService) GenerateChatText(ctx context.Context, _ string, _ string, options app_ai.TextGenerationOptions) (string, error) {
	if options.Hooks == nil {
		return "", errors.New("trial text must install attempt hooks")
	}

	first := app_ai.AttemptMeta{
		RequestID:    options.RequestID,
		AttemptIndex: 1,
		Provider:     "openai",
		Model:        "qwen3:4b",
		Operation:    string(aioperation.Chat),
		Capability:   config.CapabilityChatText,
		Route:        options.Route,
		BillingClass: options.BillingClass,
	}
	if err := options.Hooks.BeforeAttempt(ctx, first); err != nil {
		return "", err
	}
	f.appendOrder("attempt_1")

	switch f.mode {
	case trialModeCancel:
		options.Hooks.AfterAttempt(ctx, first, app_ai.AttemptResult{Outcome: "canceled", Error: context.Canceled})
		return "", context.Canceled
	case trialModeHTTP4xx:
		options.Hooks.AfterAttempt(ctx, first, app_ai.AttemptResult{Outcome: "http_4xx", HTTPStatus: 400, Error: errors.New("HTTP 400")})
		return "", errors.New("HTTP 400")
	case trialModeOpenAISuccess:
		options.Hooks.AfterAttempt(ctx, first, app_ai.AttemptResult{Outcome: "success", HTTPStatus: 200})
		if options.OnUsage != nil {
			options.OnUsage(app_ai.TokenUsage{Provider: "openai", Model: "qwen3:4b", PromptTokens: 8, OutputTokens: 4, Unbilled: true})
		}
		return "ollama answer", nil
	}

	// Fallback-eligible failure on attempt 1.
	options.Hooks.AfterAttempt(ctx, first, app_ai.AttemptResult{
		Outcome: "timeout", FallbackEligible: true, Error: errors.New("timeout"),
	})

	second := app_ai.AttemptMeta{
		RequestID:    options.RequestID,
		AttemptIndex: 2,
		Provider:     "google",
		Model:        "gemini-2.0-flash",
		Operation:    string(aioperation.Chat),
		Capability:   config.CapabilityChatText,
		Route:        options.Route,
		BillingClass: options.BillingClass,
	}
	if err := options.Hooks.BeforeAttempt(ctx, second); err != nil {
		return "", err
	}
	f.appendOrder("attempt_2")
	options.Hooks.AfterAttempt(ctx, second, app_ai.AttemptResult{Outcome: "success", HTTPStatus: 200})
	if options.OnUsage != nil {
		options.OnUsage(app_ai.TokenUsage{Provider: "google", Model: "gemini-2.0-flash", PromptTokens: 10, OutputTokens: 5, FinishReason: "STOP"})
	}
	return "gemini answer", nil
}

func (f *trialAIService) appendOrder(step string) {
	if f.order != nil {
		*f.order = append(*f.order, step)
	}
}

func newTrialOrchestrator(cost *trialCostUC, ai *trialAIService) *OrchestratorUseCase {
	log := logger.New("test", "", "info", false)

	op := &aiexecutor_entities.AICostPolicyOperation{
		Operation:              aioperation.Chat,
		NormalRoute:            airoute.Paid,
		FreeRoute:              airoute.Free,
		NormalMaxInputTokens:   3000,
		NormalMaxOutputTokens:  500,
		ReducedMaxInputTokens:  2000,
		ReducedMaxOutputTokens: 300,
	}
	policy := &aiexecutor_entities.AICostPolicy{
		EnforcementMode: aienforcementmode.ObserveOnly,
		PeriodDays:      30,
		Operations:      []*aiexecutor_entities.AICostPolicyOperation{op},
	}
	planGrant := &aiexecutor_entities.AIPolicyGrant{UserID: uuid.New()}

	cfg := &config.Config{}
	cfg.Server.Env = "test"
	cfg.AI.ChatTextPrimary = config.APIProviderConfig{Provider: "openai", ApiKey: "trial", Endpoint: "https://ollama.example.com/v1/chat/completions", Model: "qwen3:4b"}
	cfg.AI.ChatTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "gemini", Endpoint: "https://gemini.example.com", Model: "gemini-2.0-flash"}
	cfg.AI.FreeTextPrimary = config.APIProviderConfig{Provider: "openai", ApiKey: "trial", Endpoint: "https://ollama.example.com/v1/chat/completions", Model: "qwen3:4b"}
	cfg.AI.FreeTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "gemini", Endpoint: "https://gemini.example.com", Model: "gemini-2.0-flash"}
	cfg.AI.Billing.UnbilledCapabilities = []string{config.CapabilityFreeText, config.CapabilityEmbedding}

	repo := &trialPlanRepo{grant: planGrant, policy: policy, op: op}
	return &OrchestratorUseCase{
		aiService:    ai,
		repo:         repo,
		cfg:          cfg,
		costUC:       cost,
		entitlements: grant.NewEntitlementReader(log),
		logger:       log,
	}
}

func trialOptions() aiexecutor_dto.OrchestratorOptions {
	return aiexecutor_dto.OrchestratorOptions{Operation: string(aioperation.Chat), UserID: uuid.New()}
}

func indexOf(steps []string, want string) int {
	for i, step := range steps {
		if step == want {
			return i
		}
	}
	return -1
}

// The Gemini fallback must be prepared and reserved on the SAME event BEFORE the
// Gemini attempt starts. One confirm closes the request; no second event and no
// double charge.
func TestTrialFallbackPreparesBillableSameEventBeforeGemini(t *testing.T) {
	var order []string
	cost := &trialCostUC{order: &order}
	ai := &trialAIService{mode: trialModeFallback, order: &order}
	uc := newTrialOrchestrator(cost, ai)

	text, err := uc.GenerateChatText(context.Background(), "system", "user", trialOptions())
	if err != nil {
		t.Fatalf("GenerateChatText: %v", err)
	}
	if text != "gemini answer" {
		t.Errorf("text = %q, want the Gemini answer", text)
	}
	if cost.admitCalls != 1 || cost.prepareBillableCalls != 1 {
		t.Fatalf("admit/prepare calls = %d/%d, want 1/1", cost.admitCalls, cost.prepareBillableCalls)
	}
	if len(cost.admitRequestIDs) != 1 || len(cost.prepareBillableRequest) != 1 {
		t.Fatalf("request ids = %v/%v, want one each", cost.admitRequestIDs, cost.prepareBillableRequest)
	}
	if cost.admitRequestIDs[0] != cost.prepareBillableRequest[0] {
		t.Errorf("PrepareBillableAttempt must upgrade the same event: admit=%s prepare=%s",
			cost.admitRequestIDs[0], cost.prepareBillableRequest[0])
	}
	if idxPrep, idxGemini := indexOf(order, "prepare_billable"), indexOf(order, "attempt_2"); idxPrep < 0 || idxGemini < 0 || idxPrep > idxGemini {
		t.Errorf("order = %v, want prepare_billable before attempt_2", order)
	}
	if cost.confirmCalls != 1 || cost.releaseCalls != 0 {
		t.Errorf("confirm/release calls = %d/%d, want 1/0 (no double charge)", cost.confirmCalls, cost.releaseCalls)
	}
	if cost.confirmUsage.Provider != "google" || cost.confirmUsage.Model != "gemini-2.0-flash" {
		t.Errorf("confirmed winner = %q/%q, want google/gemini-2.0-flash", cost.confirmUsage.Provider, cost.confirmUsage.Model)
	}
	if cost.confirmUsage.Unbilled {
		t.Error("a non-degraded Gemini fallback must stay billed")
	}
}

// Client cancellation is terminal: no fallback, no billable preparation, and the
// event is released.
func TestTrialCancellationDoesNotFallbackOrBill(t *testing.T) {
	var order []string
	cost := &trialCostUC{order: &order}
	ai := &trialAIService{mode: trialModeCancel, order: &order}
	uc := newTrialOrchestrator(cost, ai)

	if _, err := uc.GenerateChatText(context.Background(), "system", "user", trialOptions()); err == nil {
		t.Fatal("expected a cancellation error")
	}
	if cost.prepareBillableCalls != 0 {
		t.Errorf("prepare billable calls = %d, want 0", cost.prepareBillableCalls)
	}
	if cost.confirmCalls != 0 {
		t.Errorf("confirm calls = %d, want 0", cost.confirmCalls)
	}
	if cost.releaseCalls != 1 {
		t.Errorf("release calls = %d, want 1", cost.releaseCalls)
	}
}

// A 4xx other than 429 is not fallback-eligible and must not reserve Gemini.
func TestTrialHTTP4xxDoesNotFallbackOrBill(t *testing.T) {
	var order []string
	cost := &trialCostUC{order: &order}
	ai := &trialAIService{mode: trialModeHTTP4xx, order: &order}
	uc := newTrialOrchestrator(cost, ai)

	if _, err := uc.GenerateChatText(context.Background(), "system", "user", trialOptions()); err == nil {
		t.Fatal("expected an HTTP 400 error")
	}
	if cost.prepareBillableCalls != 0 {
		t.Errorf("prepare billable calls = %d, want 0", cost.prepareBillableCalls)
	}
	if cost.confirmCalls != 0 {
		t.Errorf("confirm calls = %d, want 0", cost.confirmCalls)
	}
}

// When the daily cap degraded the request to the free route, a Gemini fallback
// that wins stays free: no PrepareBillableAttempt and the confirm is unbilled.
func TestDegradedFreeFallbackIsNotBilled(t *testing.T) {
	var order []string
	cost := &trialCostUC{order: &order, admitRoute: airoute.Free}
	ai := &trialAIService{mode: trialModeFallback, order: &order}
	uc := newTrialOrchestrator(cost, ai)

	text, err := uc.GenerateChatText(context.Background(), "system", "user", trialOptions())
	if err != nil {
		t.Fatalf("GenerateChatText: %v", err)
	}
	if text != "gemini answer" {
		t.Errorf("text = %q, want the Gemini answer", text)
	}
	if cost.prepareBillableCalls != 0 {
		t.Errorf("degraded fallback must not prepare a billable attempt, got %d", cost.prepareBillableCalls)
	}
	if cost.confirmCalls != 1 {
		t.Fatalf("confirm calls = %d, want 1", cost.confirmCalls)
	}
	if !cost.confirmUsage.Unbilled {
		t.Errorf("degraded fallback confirm = %+v, want Unbilled=true (Gemini not billed)", cost.confirmUsage)
	}
	if cost.confirmUsage.Provider != "google" || cost.confirmUsage.Model != "gemini-2.0-flash" {
		t.Errorf("degraded winner = %q/%q, want google/gemini-2.0-flash", cost.confirmUsage.Provider, cost.confirmUsage.Model)
	}
}

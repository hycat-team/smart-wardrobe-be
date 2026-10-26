package ai

import (
	"net/http"
	"testing"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
)

func billingTestService() *AIService {
	cfg := &config.Config{}
	cfg.AI.ChatTextPrimary = config.APIProviderConfig{Provider: "google", ApiKey: "paid-chat-key", Endpoint: "chat-endpoint", Model: "paid-chat"}
	cfg.AI.ChatTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "paid-chat-fallback-key", Endpoint: "chat-endpoint", Model: "paid-chat-fallback"}
	cfg.AI.RecommendationTextPrimary = config.APIProviderConfig{Provider: "google", ApiKey: "paid-outfit-key", Endpoint: "outfit-endpoint", Model: "paid-outfit"}
	cfg.AI.RecommendationTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "paid-outfit-fallback-key", Endpoint: "outfit-endpoint", Model: "paid-outfit-fallback"}
	cfg.AI.FreeTextPrimary = config.APIProviderConfig{Provider: "google", ApiKey: "free-key", Endpoint: "free-endpoint", Model: "free-model"}
	cfg.AI.FreeTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "free-fallback-key", Endpoint: "free-endpoint", Model: "free-fallback-model"}
	cfg.AI.Billing.UnbilledCapabilities = []string{config.CapabilityFreeText}

	return &AIService{
		cfg:                       cfg,
		chatTextClient:            &http.Client{},
		recommendationTextClient:  &http.Client{},
		freeTextClient:            &http.Client{},
		chatTextLimiter:           newRPMLimiter(600, 1),
		recommendationTextLimiter: newRPMLimiter(600, 1),
		freeTextLimiter:           newRPMLimiter(600, 1),
	}
}

func TestResolveTextCallPlanFreeUsesFreeConfigAndBucket(t *testing.T) {
	s := billingTestService()

	plan := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierFree}, app_ai.AIOperationChat)

	if plan.primary.provider.ApiKey != "free-key" || plan.primary.capability != config.CapabilityFreeText {
		t.Errorf("free primary = %+v, want the FREE_TEXT primary", plan.primary)
	}
	if plan.primary.client != s.freeTextClient || plan.primary.limiter != s.freeTextLimiter {
		t.Error("free primary must use the free client and limiter")
	}
	if plan.fallback.provider.ApiKey != "free-fallback-key" {
		t.Errorf("free fallback = %+v, want the FREE_TEXT fallback", plan.fallback)
	}
	// Exactly two attempts: no third free failover exists any more (feature 032
	// R5), and a free plan never borrows a paid key.
	attempts := plan.textAttempts()
	if len(attempts) != 2 {
		t.Fatalf("free plan attempts = %d, want 2 (primary + fallback)", len(attempts))
	}
	for _, attempt := range attempts {
		if attempt.provider.ApiKey == "paid-chat-key" || attempt.provider.ApiKey == "paid-chat-fallback-key" {
			t.Errorf("free plan must not use a paid key, got %+v", attempt.provider)
		}
	}
}

func TestResolveTextCallPlanPaidKeepsOperationConfig(t *testing.T) {
	s := billingTestService()

	chat := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}, app_ai.AIOperationChat)
	if chat.primary.provider.ApiKey != "paid-chat-key" || chat.primary.client != s.chatTextClient {
		t.Errorf("chat paid primary = %+v, want the CHAT_TEXT config and client", chat.primary)
	}
	if chat.fallback.provider.ApiKey != "paid-chat-fallback-key" {
		t.Errorf("chat paid fallback = %+v, want the CHAT_TEXT fallback", chat.fallback)
	}
	// The paid plan keeps exactly two attempts even though a FREE_TEXT primary is
	// configured: the synthetic third free attempt was removed.
	attempts := chat.textAttempts()
	if len(attempts) != 2 {
		t.Fatalf("paid plan attempts = %d, want 2 (primary + fallback)", len(attempts))
	}
	for _, attempt := range attempts {
		if attempt.provider.ApiKey == "free-key" || attempt.provider.ApiKey == "free-fallback-key" {
			t.Errorf("paid plan must not fall through to the free key, got %+v", attempt.provider)
		}
	}

	outfit := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}, app_ai.AIOperationOutfit)
	if outfit.primary.provider.ApiKey != "paid-outfit-key" || outfit.primary.client != s.recommendationTextClient {
		t.Errorf("outfit paid primary = %+v, want the RECOMMENDATION_TEXT config and client", outfit.primary)
	}
}

func TestResolveTextCallPlanHonoursExplicitOverrides(t *testing.T) {
	s := billingTestService()
	overrides := app_ai.TextGenerationOptions{
		Tier:     app_ai.TextTierPaid,
		Primary:  &app_ai.TextProviderConfig{Provider: "openai", APIKey: "override-key", Endpoint: "override-endpoint", Model: "override-model", Capability: config.CapabilityChatText},
		Fallback: &app_ai.TextProviderConfig{Provider: "openai", APIKey: "override-fallback-key", Endpoint: "override-endpoint", Model: "override-fallback-model", Capability: config.CapabilityChatText},
	}

	plan := s.resolveTextCallPlan(overrides, app_ai.AIOperationChat)

	if plan.primary.provider.Provider != "openai" || plan.primary.provider.ApiKey != "override-key" {
		t.Errorf("primary = %+v, want the override config", plan.primary.provider)
	}
	if plan.fallback.provider.ApiKey != "override-fallback-key" {
		t.Errorf("fallback = %+v, want the override fallback", plan.fallback.provider)
	}
}

func TestWithUsageStampClassifiesOpenAIAsUnbilledBeforeCapabilityList(t *testing.T) {
	s := billingTestService()

	// chat_text is not in AI_UNBILLED_CAPABILITIES, so only the provider rule can
	// make this unbilled.
	var usage app_ai.TokenUsage
	options := app_ai.TextGenerationOptions{OnUsage: func(u app_ai.TokenUsage) { usage = u }}
	stamped := s.withUsageStamp(options, textCall{
		provider:   config.APIProviderConfig{Provider: "openai", Model: "qwen3:4b"},
		capability: config.CapabilityChatText,
	})
	stamped.OnUsage(app_ai.TokenUsage{PromptTokens: 5})

	if !usage.Unbilled {
		t.Errorf("usage = %+v, want Unbilled=true for provider=openai", usage)
	}
	if usage.Provider != "openai" || usage.Model != "qwen3:4b" {
		t.Errorf("usage = %+v, want provider/model openai/qwen3:4b", usage)
	}
}

func TestWithUsageStampKeepsCapabilityListForGoogle(t *testing.T) {
	s := billingTestService()

	var freeUsage app_ai.TokenUsage
	freeOptions := app_ai.TextGenerationOptions{OnUsage: func(u app_ai.TokenUsage) { freeUsage = u }}
	stamped := s.withUsageStamp(freeOptions, textCall{provider: s.cfg.AI.FreeTextPrimary, capability: config.CapabilityFreeText})
	stamped.OnUsage(app_ai.TokenUsage{})
	if !freeUsage.Unbilled {
		t.Error("google free_text must stay unbilled through AI_UNBILLED_CAPABILITIES")
	}

	var paidUsage app_ai.TokenUsage
	paidOptions := app_ai.TextGenerationOptions{OnUsage: func(u app_ai.TokenUsage) { paidUsage = u }}
	stamped = s.withUsageStamp(paidOptions, textCall{provider: s.cfg.AI.ChatTextPrimary, capability: config.CapabilityChatText})
	stamped.OnUsage(app_ai.TokenUsage{})
	if paidUsage.Unbilled {
		t.Error("google chat_text must stay billed; only provider=openai is unbilled by rule")
	}
}

func TestStampUsageClassifiesOpenAIAsUnbilled(t *testing.T) {
	s := billingTestService()

	usage := s.stampUsage(app_ai.TokenUsage{}, config.APIProviderConfig{Provider: "openai", Model: "qwen3:4b"}, config.CapabilityChatText)
	if !usage.Unbilled {
		t.Errorf("usage = %+v, want Unbilled=true for provider=openai", usage)
	}

	usage = s.stampUsage(app_ai.TokenUsage{}, config.APIProviderConfig{Provider: "google", Model: "gemini"}, config.CapabilityChatText)
	if usage.Unbilled {
		t.Errorf("usage = %+v, want Unbilled=false for google chat_text", usage)
	}
}

func TestOverrideProviderCarriesReasoningEffort(t *testing.T) {
	got := overrideProvider(&app_ai.TextProviderConfig{
		Provider:        "openai",
		APIKey:          "key",
		Endpoint:        "https://ollama.example.com/v1/chat/completions",
		Model:           "qwen3:4b",
		ReasoningEffort: "none",
	}, config.APIProviderConfig{Provider: "google"})

	if got.ReasoningEffort != "none" {
		t.Fatalf("ReasoningEffort = %q, want none", got.ReasoningEffort)
	}
}

func TestResolveTextCallPlanCarriesReasoningEffort(t *testing.T) {
	s := billingTestService()
	s.cfg.AI.ChatTextPrimary.ReasoningEffort = "low"

	plan := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}, app_ai.AIOperationChat)
	if plan.primary.provider.ReasoningEffort != "low" {
		t.Fatalf("primary ReasoningEffort = %q, want low from CHAT_TEXT primary", plan.primary.provider.ReasoningEffort)
	}

	overridePlan := s.resolveTextCallPlan(app_ai.TextGenerationOptions{
		Tier: app_ai.TextTierPaid,
		Primary: &app_ai.TextProviderConfig{
			Provider:        "openai",
			Model:           "qwen3:4b",
			ReasoningEffort: "none",
		},
	}, app_ai.AIOperationChat)
	if overridePlan.primary.provider.ReasoningEffort != "none" {
		t.Fatalf("override ReasoningEffort = %q, want none", overridePlan.primary.provider.ReasoningEffort)
	}
}

func TestWithUsageStampReportsBillingClassOfWinner(t *testing.T) {
	s := billingTestService()

	var freeUsage app_ai.TokenUsage
	freeOptions := app_ai.TextGenerationOptions{OnUsage: func(usage app_ai.TokenUsage) { freeUsage = usage }}
	stamped := s.withUsageStamp(freeOptions, textCall{provider: s.cfg.AI.FreeTextPrimary, capability: config.CapabilityFreeText})
	stamped.OnUsage(app_ai.TokenUsage{PromptTokens: 3})

	if freeUsage.Provider != "google" || freeUsage.Model != "free-model" || !freeUsage.Unbilled {
		t.Errorf("free usage = %+v, want provider/model from FREE_TEXT and Unbilled=true", freeUsage)
	}

	var paidUsage app_ai.TokenUsage
	paidOptions := app_ai.TextGenerationOptions{OnUsage: func(usage app_ai.TokenUsage) { paidUsage = usage }}
	stamped = s.withUsageStamp(paidOptions, textCall{provider: s.cfg.AI.ChatTextPrimary, capability: config.CapabilityChatText})
	stamped.OnUsage(app_ai.TokenUsage{PromptTokens: 3})

	if paidUsage.Model != "paid-chat" || paidUsage.Unbilled {
		t.Errorf("paid usage = %+v, want provider/model from CHAT_TEXT and Unbilled=false", paidUsage)
	}
}

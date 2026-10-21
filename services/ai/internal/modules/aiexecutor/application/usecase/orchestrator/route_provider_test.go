package orchestrator

import (
	"testing"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
)

func routeTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.AI.ChatTextPrimary = config.APIProviderConfig{Provider: "google", ApiKey: "paid-chat-key", Endpoint: "chat-endpoint", Model: "paid-chat"}
	cfg.AI.ChatTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "paid-chat-fallback-key", Endpoint: "chat-endpoint", Model: "paid-chat-fallback"}
	cfg.AI.RecommendationTextPrimary = config.APIProviderConfig{Provider: "google", ApiKey: "paid-outfit-key", Endpoint: "outfit-endpoint", Model: "paid-outfit"}
	cfg.AI.RecommendationTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "paid-outfit-fallback-key", Endpoint: "outfit-endpoint", Model: "paid-outfit-fallback"}
	cfg.AI.FreeTextPrimary = config.APIProviderConfig{Provider: "google", ApiKey: "free-key", Endpoint: "free-endpoint", Model: "free-model"}
	cfg.AI.FreeTextFallback = config.APIProviderConfig{Provider: "google", ApiKey: "free-fallback-key", Endpoint: "free-endpoint", Model: "free-fallback-model"}
	cfg.AI.Billing.UnbilledCapabilities = []string{config.CapabilityFreeText, config.CapabilityEmbedding}
	return cfg
}

func TestRouteProviderConfigsFreeUsesFreeTextForEveryTextOperation(t *testing.T) {
	uc := &OrchestratorUseCase{cfg: routeTestConfig()}

	for _, operation := range []aioperation.AIOperation{aioperation.Chat, aioperation.Summary, aioperation.Rewriter, aioperation.Outfit} {
		primary, fallback := uc.routeProviderConfigs(operation, airoute.Free)
		if primary.APIKey != "free-key" || primary.Model != "free-model" || primary.Capability != config.CapabilityFreeText {
			t.Errorf("operation %s free primary = %+v, want the FREE_TEXT primary", operation, primary)
		}
		if fallback.APIKey != "free-fallback-key" || fallback.Capability != config.CapabilityFreeText {
			t.Errorf("operation %s free fallback = %+v, want the FREE_TEXT fallback", operation, fallback)
		}
	}
}

func TestRouteProviderConfigsPaidUsesTheOperationConfig(t *testing.T) {
	uc := &OrchestratorUseCase{cfg: routeTestConfig()}

	chat, chatFallback := uc.routeProviderConfigs(aioperation.Chat, airoute.Paid)
	if chat.APIKey != "paid-chat-key" || chat.Capability != config.CapabilityChatText {
		t.Errorf("chat paid primary = %+v, want the CHAT_TEXT primary", chat)
	}
	if chatFallback.Model != "paid-chat-fallback" {
		t.Errorf("chat paid fallback = %+v, want the CHAT_TEXT fallback", chatFallback)
	}

	outfit, outfitFallback := uc.routeProviderConfigs(aioperation.Outfit, airoute.Paid)
	if outfit.APIKey != "paid-outfit-key" || outfit.Capability != config.CapabilityRecommendationText {
		t.Errorf("outfit paid primary = %+v, want the RECOMMENDATION_TEXT primary", outfit)
	}
	if outfitFallback.Model != "paid-outfit-fallback" {
		t.Errorf("outfit paid fallback = %+v, want the RECOMMENDATION_TEXT fallback", outfitFallback)
	}
}

func TestApplyTextRouteFreeWritesFreeConfigIntoPlanAndOptions(t *testing.T) {
	uc := &OrchestratorUseCase{cfg: routeTestConfig()}
	plan := &uc_interfaces.ExecutionPlan{Operation: aioperation.Chat, Provider: "google", Model: "paid-chat"}
	options := app_ai.TextGenerationOptions{}

	uc.applyTextRoute(plan, &options, airoute.Free)

	if plan.Model != "free-model" {
		t.Errorf("plan model = %q, want the model that will actually run", plan.Model)
	}
	if options.Tier != app_ai.TextTierFree {
		t.Errorf("tier = %q, want %q", options.Tier, app_ai.TextTierFree)
	}
	if options.Primary == nil || options.Primary.APIKey != "free-key" {
		t.Errorf("primary = %+v, want the FREE_TEXT primary", options.Primary)
	}
	if options.Fallback == nil || options.Fallback.Model != "free-fallback-model" {
		t.Errorf("fallback = %+v, want the FREE_TEXT fallback", options.Fallback)
	}
}

func TestApplyTextRoutePaidKeepsTheOperationConfig(t *testing.T) {
	uc := &OrchestratorUseCase{cfg: routeTestConfig()}
	plan := &uc_interfaces.ExecutionPlan{Operation: aioperation.Outfit, Provider: "google", Model: "free-model"}
	options := app_ai.TextGenerationOptions{}

	uc.applyTextRoute(plan, &options, airoute.Paid)

	if plan.Model != "paid-outfit" {
		t.Errorf("plan model = %q, want the paid outfit model", plan.Model)
	}
	if options.Tier != app_ai.TextTierPaid {
		t.Errorf("tier = %q, want %q", options.Tier, app_ai.TextTierPaid)
	}
	if options.Primary == nil || options.Primary.APIKey != "paid-outfit-key" {
		t.Errorf("primary = %+v, want the RECOMMENDATION_TEXT primary", options.Primary)
	}
}

func TestUsageProviderModelPrefersTheConfigThatActuallyRan(t *testing.T) {
	plan := &uc_interfaces.ExecutionPlan{Provider: "google", Model: "planned-model"}

	provider, model := usageProviderModel(plan, app_ai.TokenUsage{Provider: "google", Model: "free-model"})
	if provider != "google" || model != "free-model" {
		t.Errorf("provider/model = %q/%q, want the usage values", provider, model)
	}

	provider, model = usageProviderModel(plan, app_ai.TokenUsage{})
	if model != "planned-model" {
		t.Errorf("model = %q, want the plan fallback when usage is empty", model)
	}
}

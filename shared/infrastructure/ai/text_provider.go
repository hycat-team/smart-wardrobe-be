package ai

import (
	"net/http"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"

	"golang.org/x/time/rate"
)

// textCall is one resolved provider attempt: the config to call, the capability
// it belongs to (which decides billing), and the client/limiter bucket it runs
// in.
type textCall struct {
	provider   config.APIProviderConfig
	capability string
	client     *http.Client
	limiter    *rate.Limiter
}

// textCallPlan is the ordered set of attempts for one text request. A paid plan
// keeps a free failover as its last resort; a free plan never falls back to a
// paid key, because the free key is a separate quota rather than a cheaper copy.
type textCallPlan struct {
	primary      textCall
	fallback     textCall
	freeFailover *textCall
	tier         string
}

func (s *AIService) resolveTextCallPlan(options app_ai.TextGenerationOptions, operation string) textCallPlan {
	tier := options.Tier
	if tier == "" {
		tier = app_ai.TextTierPaid
	}

	if tier == app_ai.TextTierFree {
		return textCallPlan{
			primary:  s.freeTextCall(options.Primary, s.cfg.AI.FreeTextPrimary),
			fallback: s.freeTextCall(options.Fallback, s.cfg.AI.FreeTextFallback),
			tier:     tier,
		}
	}

	capability := config.CapabilityChatText
	defaultPrimary := s.cfg.AI.ChatTextPrimary
	defaultFallback := s.cfg.AI.ChatTextFallback
	client, limiter := s.chatTextClient, s.chatTextLimiter
	if operation == app_ai.AIOperationOutfit {
		capability = config.CapabilityRecommendationText
		defaultPrimary = s.cfg.AI.RecommendationTextPrimary
		defaultFallback = s.cfg.AI.RecommendationTextFallback
		client, limiter = s.recommendationTextClient, s.recommendationTextLimiter
	}

	plan := textCallPlan{
		primary: textCall{
			provider:   overrideProvider(options.Primary, defaultPrimary),
			capability: overrideCapability(options.Primary, capability),
			client:     client,
			limiter:    limiter,
		},
		fallback: textCall{
			provider:   overrideProvider(options.Fallback, defaultFallback),
			capability: overrideCapability(options.Fallback, capability),
			client:     client,
			limiter:    limiter,
		},
		tier: tier,
	}
	if s.cfg.AI.FreeTextPrimary.Provider != "" {
		free := textCall{
			provider:   s.cfg.AI.FreeTextPrimary,
			capability: config.CapabilityFreeText,
			client:     s.freeTextClient,
			limiter:    s.freeTextLimiter,
		}
		plan.freeFailover = &free
	}
	return plan
}

func (s *AIService) freeTextCall(override *app_ai.TextProviderConfig, defaultCfg config.APIProviderConfig) textCall {
	return textCall{
		provider:   overrideProvider(override, defaultCfg),
		capability: overrideCapability(override, config.CapabilityFreeText),
		client:     s.freeTextClient,
		limiter:    s.freeTextLimiter,
	}
}

// withUsageStamp makes the provider's usage callback report which config
// actually served the call. The orchestrator needs that to record a truthful
// provider/model and to price the request against the right billing class when a
// fallback wins.
func (s *AIService) withUsageStamp(options app_ai.TextGenerationOptions, call textCall) app_ai.TextGenerationOptions {
	original := options.OnUsage
	if original == nil {
		return options
	}
	unbilled := s.cfg.AI.IsUnbilledCapability(call.capability)
	options.OnUsage = func(usage app_ai.TokenUsage) {
		usage.Provider = call.provider.Provider
		usage.Model = call.provider.Model
		usage.Unbilled = unbilled
		original(usage)
	}
	return options
}

// stampUsage fills in the config that actually served a call for direct
// (non-callback) usage results such as vision.
func (s *AIService) stampUsage(usage app_ai.TokenUsage, provider config.APIProviderConfig, capability string) app_ai.TokenUsage {
	usage.Provider = provider.Provider
	usage.Model = provider.Model
	usage.Unbilled = s.cfg.AI.IsUnbilledCapability(capability)
	return usage
}

func overrideProvider(override *app_ai.TextProviderConfig, defaultCfg config.APIProviderConfig) config.APIProviderConfig {
	if override == nil {
		return defaultCfg
	}
	return config.APIProviderConfig{
		Provider: override.Provider,
		ApiKey:   override.APIKey,
		Endpoint: override.Endpoint,
		Model:    override.Model,
	}
}

func overrideCapability(override *app_ai.TextProviderConfig, defaultCapability string) string {
	if override != nil && override.Capability != "" {
		return override.Capability
	}
	return defaultCapability
}

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

// textCallPlan is the ordered set of attempts for one text request: exactly one
// primary and at most one fallback. There is no third "free failover" attempt
// any more (feature 032 FR-023 / research.md R5); a request can never exceed two
// provider calls.
type textCallPlan struct {
	primary  textCall
	fallback textCall
	tier     string
}

// textAttempts returns the attempts to run, in order, capped at two. It is the
// single place that decides how many provider calls a request may make, so the
// "max two attempts" invariant cannot drift between the streaming and
// non-streaming paths.
func (p textCallPlan) textAttempts() []textCall {
	attempts := make([]textCall, 0, 2)
	if p.primary.provider.Provider != "" {
		attempts = append(attempts, p.primary)
	}
	if p.fallback.provider.Provider != "" {
		attempts = append(attempts, p.fallback)
	}
	return attempts
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
// fallback wins. The wrapper is installed even when the caller has no callback,
// because the attempt telemetry reads the stamped values as well.
func (s *AIService) withUsageStamp(options app_ai.TextGenerationOptions, call textCall) app_ai.TextGenerationOptions {
	original := options.OnUsage
	unbilled := s.isUnbilled(call.provider.Provider, call.capability)
	options.OnUsage = func(usage app_ai.TokenUsage) {
		usage.Provider = call.provider.Provider
		usage.Model = call.provider.Model
		usage.Unbilled = unbilled
		if original != nil {
			original(usage)
		}
	}
	return options
}

// stampUsage fills in the config that actually served a call for direct
// (non-callback) usage results such as vision.
func (s *AIService) stampUsage(usage app_ai.TokenUsage, provider config.APIProviderConfig, capability string) app_ai.TokenUsage {
	usage.Provider = provider.Provider
	usage.Model = provider.Model
	usage.Unbilled = s.isUnbilled(provider.Provider, capability)
	return usage
}

// isUnbilled classifies the winning provider BEFORE consulting the capability
// list. provider=openai is the Ollama trial and is never charged, regardless of
// which capability it served. Keeping the capability list for google preserves
// AI_UNBILLED_CAPABILITIES=free_text,embedding and does not turn a Gemini
// fallback on a text capability into free usage.
func (s *AIService) isUnbilled(provider, capability string) bool {
	if provider == ProviderOpenAI {
		return true
	}
	return s.cfg.AI.IsUnbilledCapability(capability)
}

func overrideProvider(override *app_ai.TextProviderConfig, defaultCfg config.APIProviderConfig) config.APIProviderConfig {
	if override == nil {
		return defaultCfg
	}
	return config.APIProviderConfig{
		Provider:        override.Provider,
		ApiKey:          override.APIKey,
		Endpoint:        override.Endpoint,
		Model:           override.Model,
		ReasoningEffort: override.ReasoningEffort,
	}
}

func overrideCapability(override *app_ai.TextProviderConfig, defaultCapability string) string {
	if override != nil && override.Capability != "" {
		return override.Capability
	}
	return defaultCapability
}

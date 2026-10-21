package ai

import (
	"context"

	"smart-wardrobe-be/shared/domain/constants/tokenestimationmethod"

	"github.com/google/uuid"
)

const (
	AIOperationChat     = "chat"
	AIOperationOutfit   = "outfit"
	AIOperationSummary  = "summary"
	AIOperationRewriter = "rewriter"
)

// Text provider tiers. The tier selects which client and rate limiter a text
// call uses, so a free-routed request can never share the paid bucket by
// accident.
const (
	TextTierPaid = "paid"
	TextTierFree = "free"
)

// TextProviderConfig is the resolved provider for one text call. It is built
// by the orchestrator from configuration and passed in-process only: it never
// crosses a transport, so the API key cannot leak into a gRPC message.
type TextProviderConfig struct {
	Provider   string
	APIKey     string
	Endpoint   string
	Model      string
	Capability string
}

type TokenUsage struct {
	PromptTokens   int64
	OutputTokens   int64
	ThinkingTokens int64
	FinishReason   string
	// Provider/Model report the config that actually served the call, which
	// differs from the plan when a fallback or a free failover wins.
	Provider string
	Model    string
	// Unbilled reports that the winning config belongs to a capability the
	// team is not charged for (AI_UNBILLED_CAPABILITIES).
	Unbilled bool
}

type TokenEstimationMeta struct {
	EstimatedPromptTokens int64
	TokenEstimationMethod tokenestimationmethod.Method
	TokenCountLatencyMs   *int64
}

type TextGenerationOptions struct {
	MaxOutputTokens  int
	Temperature      float64
	ResponseMIMEType string
	ResponseSchema   any
	UserID           uuid.UUID
	Operation        string
	RequestID        uuid.UUID
	OnUsage          func(usage TokenUsage)
	// Tier is TextTierPaid or TextTierFree. Empty keeps the pre-existing
	// behaviour (paid primary with the operation default fallback).
	Tier string
	// Primary and Fallback override the operation defaults. Nil means "use the
	// operation's configured primary/fallback", which is what every pre-existing
	// caller gets without changing.
	Primary  *TextProviderConfig
	Fallback *TextProviderConfig
}

type IAIService interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	// AnalyzeImageWithUsage is the additive usage-reporting variant of
	// AnalyzeImage. The original method stays for existing callers.
	AnalyzeImageWithUsage(ctx context.Context, imageUrl string, prompt string) (string, TokenUsage, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (<-chan string, <-chan error)
	EstimatePromptTokens(ctx context.Context, operation string, systemPrompt, userPrompt string) (TokenEstimationMeta, error)
}

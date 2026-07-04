package dto

import (
	"github.com/google/uuid"
)

const (
	AIOperationChat     = "chat"
	AIOperationOutfit   = "outfit"
	AIOperationSummary  = "summary"
	AIOperationRewriter = "rewriter"
	AIRoutePaid         = "paid"
	AIRouteFree         = "free"
	AIRouteLocal        = "local"

	TokenEstimationLocal         = "LOCAL"
	TokenEstimationProviderCount = "PROVIDER_COUNT"
)

type AITokenEstimationMeta struct {
	EstimatedPromptTokens int64
	TokenEstimationMethod string
	TokenCountLatencyMs   *int64
}

type AICostDecision struct {
	RequestID       uuid.UUID
	Route           string
	MaxInputTokens  int
	MaxOutputTokens int
	Tracked         bool
}

type AIUsage struct {
	PromptTokens   int64
	OutputTokens   int64
	ThinkingTokens int64
	FinishReason   string
	Provider       string
	Model          string
}

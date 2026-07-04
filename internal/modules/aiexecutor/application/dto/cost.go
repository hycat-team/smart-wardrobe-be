package dto

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"

	"github.com/google/uuid"
)

const (
	AIOperationChat      = "chat"
	AIOperationOutfit    = "outfit"
	AIOperationSummary   = "summary"
	AIOperationRewriter  = "rewriter"
	AIOperationVision    = "vision"
	AIOperationEmbedding = "embedding"
	AIRoutePaid          = "paid"
	AIRouteFree          = "free"
	AIRouteLocal         = "local"

	TokenEstimationLocal         = "LOCAL"
	TokenEstimationProviderCount = "PROVIDER_COUNT"
)

type AITokenEstimationMeta struct {
	EstimatedPromptTokens int64
	TokenEstimationMethod string
	TokenCountLatencyMs   *int64
}

type ExecutionPlan struct {
	RequestID             uuid.UUID
	UserID                uuid.UUID
	Operation             string
	Provider              aiprovider.Provider
	Model                 aimodel.Model
	Route                 string
	MaxInputTokens        int
	MaxOutputTokens       int
	EstimatedPromptTokens int64
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
	Provider       aiprovider.Provider
	Model          aimodel.Model
}

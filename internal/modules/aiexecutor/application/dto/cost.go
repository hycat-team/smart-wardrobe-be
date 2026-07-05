package dto

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"
	"smart-wardrobe-be/internal/shared/domain/constants/tokenestimationmethod"

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
)

type AITokenEstimationMeta struct {
	EstimatedPromptTokens int64
	TokenEstimationMethod tokenestimationmethod.Method
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

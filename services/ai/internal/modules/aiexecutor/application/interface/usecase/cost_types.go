package usecase

import (
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/shared/domain/constants/tokenestimationmethod"

	"github.com/google/uuid"
)

// Cost-control types internal to the AI pipeline (orchestrator <-> cost use case).
// They never cross a transport or module boundary, so they live next to the use
// case interfaces instead of application/dto (DTO Convention, design-pattern.md).

type ExecutionPlan struct {
	RequestID             uuid.UUID
	UserID                uuid.UUID
	Operation             aioperation.AIOperation
	Provider              string
	Model                 string
	Route                 airoute.AIRoute
	MaxInputTokens        int
	MaxOutputTokens       int
	EstimatedPromptTokens int64
}

type AICostDecision struct {
	RequestID       uuid.UUID
	Route           airoute.AIRoute
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

type AITokenEstimationMeta struct {
	EstimatedPromptTokens int64
	TokenEstimationMethod tokenestimationmethod.Method
	TokenCountLatencyMs   *int64
}

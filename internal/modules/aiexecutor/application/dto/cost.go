package dto

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/internal/shared/domain/constants/tokenestimationmethod"

	"github.com/google/uuid"
)

type AITokenEstimationMeta struct {
	EstimatedPromptTokens int64
	TokenEstimationMethod tokenestimationmethod.Method
	TokenCountLatencyMs   *int64
}

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

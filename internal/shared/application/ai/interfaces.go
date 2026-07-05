package ai

import (
	"context"

	"smart-wardrobe-be/internal/shared/domain/constants/tokenestimationmethod"

	"github.com/google/uuid"
)

const (
	AIOperationChat     = "chat"
	AIOperationOutfit   = "outfit"
	AIOperationSummary  = "summary"
	AIOperationRewriter = "rewriter"
)

type TokenUsage struct {
	PromptTokens   int64
	OutputTokens   int64
	ThinkingTokens int64
	FinishReason   string
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
}

type IAIService interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (<-chan string, <-chan error)
	EstimatePromptTokens(ctx context.Context, operation string, systemPrompt, userPrompt string) (TokenEstimationMeta, error)
}

package contract

import (
	"context"

	"github.com/google/uuid"
)

type OrchestratorOptions struct {
	MaxOutputTokens  int
	Temperature      float64
	ResponseMIMEType string
	ResponseSchema   any
	UserID           uuid.UUID
	Operation        string
}

type IAIOrchestrator interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options OrchestratorOptions) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options OrchestratorOptions) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options OrchestratorOptions) (<-chan string, <-chan error)
}

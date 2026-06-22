package ai

import (
	"context"

	"github.com/google/uuid"
)

type TextGenerationOptions struct {
	MaxOutputTokens  int
	Temperature      float64
	ResponseMIMEType string
	ResponseSchema   any
	UserID           uuid.UUID
	Operation        string
	RequestID        uuid.UUID
}

type IAIService interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options TextGenerationOptions) (<-chan string, <-chan error)
}

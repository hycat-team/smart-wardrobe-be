package ai

import (
	"context"
)

type IAIService interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string) (<-chan string, <-chan error)
}

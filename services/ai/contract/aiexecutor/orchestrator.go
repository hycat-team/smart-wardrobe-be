package contract

import (
	"context"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
)

type IAIOrchestrator interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options dto.OrchestratorOptions) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options dto.OrchestratorOptions) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options dto.OrchestratorOptions) (<-chan string, <-chan error)
}

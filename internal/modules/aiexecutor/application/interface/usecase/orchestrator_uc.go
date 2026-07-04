package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/aiexecutor/contract"
)

type IAIOrchestratorUseCase interface {
	AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options contract.OrchestratorOptions) (string, error)
	GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options contract.OrchestratorOptions) (string, error)
	GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options contract.OrchestratorOptions) (<-chan string, <-chan error)
}

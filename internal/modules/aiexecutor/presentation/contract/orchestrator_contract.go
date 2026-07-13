package contract

import (
	"context"

	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
)

type AIOrchestratorContract struct {
	orchestratorUC uc_interfaces.IAIOrchestratorUseCase
}

func NewAIOrchestratorContract(orchestratorUC uc_interfaces.IAIOrchestratorUseCase) IAIOrchestrator {
	return &AIOrchestratorContract{orchestratorUC: orchestratorUC}
}

func (c *AIOrchestratorContract) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	return c.orchestratorUC.AnalyzeImage(ctx, imageUrl, prompt)
}

func (c *AIOrchestratorContract) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	return c.orchestratorUC.GenerateEmbeddings(ctx, chunks)
}

func (c *AIOrchestratorContract) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options dto.OrchestratorOptions) (string, error) {
	return c.orchestratorUC.GenerateChatText(ctx, systemPrompt, userPrompt, options)
}

func (c *AIOrchestratorContract) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options dto.OrchestratorOptions) (string, error) {
	return c.orchestratorUC.GenerateRecommendationText(ctx, systemPrompt, userPrompt, options)
}

func (c *AIOrchestratorContract) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options dto.OrchestratorOptions) (<-chan string, <-chan error) {
	return c.orchestratorUC.GenerateChatTextStream(ctx, systemPrompt, userPrompt, options)
}

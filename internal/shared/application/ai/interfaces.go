package ai

import (
	"context"
	"smart-wardrobe-be/internal/shared/application/dto"
)

type IAIService interface {
	AnalyzeFashionImage(ctx context.Context, imageUrl string, categories []dto.AICategoryRef) (*dto.FashionMetadataResult, error)
	GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error)
	GenerateText(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

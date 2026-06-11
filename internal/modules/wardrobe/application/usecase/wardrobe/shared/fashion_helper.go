package shared

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/shared/application/ai"
)

// BuildRichTextContext formats a description prompt context of a wardrobe item for AI embedding generation.
func BuildRichTextContext(categoryName, color, style, material, pattern, fit, seasonality, description string) string {
	return fmt.Sprintf(
		"Danh mục trang phục: %s, Thuộc tính màu sắc: %s, Định hình phong cách thiết kế: %s, Chất liệu: %s, Họa tiết: %s, Kiểu dáng: %s, Mùa phù hợp: %s. Mô tả chi tiết: %s",
		categoryName,
		color,
		style,
		material,
		pattern,
		fit,
		seasonality,
		description,
	)
}

// GenerateItemEmbedding calls the AI service to generate a text embedding, validates the response, and returns the embedding.
func GenerateItemEmbedding(ctx context.Context, aiService ai.IAIService, text string) ([]float32, error) {
	embeddings, err := aiService.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned by AI service")
	}
	return embeddings[0], nil
}

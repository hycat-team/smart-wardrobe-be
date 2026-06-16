package shared

import (
	"context"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/shared/application/ai"
)

// RemoveVietnameseSigns strips accents and diacritics from Vietnamese text.
func RemoveVietnameseSigns(str string) string {
	var replacer = strings.NewReplacer(
		"á", "a", "à", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ă", "a", "ắ", "a", "ằ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"â", "a", "ấ", "a", "ầ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"Á", "A", "À", "A", "Ả", "A", "Ã", "A", "Ạ", "A",
		"Ă", "A", "Ắ", "A", "Ằ", "A", "Ẳ", "A", "Ẵ", "A", "Ặ", "A",
		"Â", "A", "Ấ", "A", "Ầ", "A", "Ẩ", "A", "Ẫ", "A", "Ậ", "A",
		"é", "e", "è", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ê", "e", "ế", "e", "ề", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"É", "E", "È", "E", "Ẻ", "E", "Ẽ", "E", "Ẹ", "E",
		"Ê", "E", "Ế", "E", "Ề", "E", "Ể", "E", "Ễ", "E", "Ệ", "E",
		"í", "i", "ì", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"Í", "I", "Ì", "I", "Ỉ", "I", "Ĩ", "I", "Ị", "I",
		"ó", "o", "ò", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ô", "o", "ố", "o", "ồ", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ơ", "o", "ớ", "o", "ờ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"Ó", "O", "Ò", "O", "Ỏ", "O", "Õ", "O", "Ọ", "O",
		"Ô", "O", "Ố", "O", "Ồ", "O", "Ổ", "O", "Ỗ", "O", "Ộ", "O",
		"Ơ", "O", "Ớ", "O", "Ờ", "O", "Ở", "O", "Ỡ", "O", "Ợ", "O",
		"ú", "u", "ù", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ư", "u", "ứ", "u", "ừ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"Ú", "U", "Ù", "U", "Ủ", "U", "Ũ", "U", "Ụ", "U",
		"Ư", "U", "Ứ", "U", "Ừ", "U", "Ử", "U", "Ữ", "U", "Ự", "U",
		"ý", "y", "ỳ", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
		"Ý", "Y", "Ỳ", "Y", "Ỷ", "Y", "Ỹ", "Y", "Ỵ", "Y",
		"đ", "d", "Đ", "D",
	)
	return replacer.Replace(str)
}

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

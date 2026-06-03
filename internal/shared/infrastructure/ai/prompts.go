package ai

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/shared/application/dto"
)

func getVisionSystemPrompt(categories []dto.AICategoryRef) string {
	var catList []string
	for _, cat := range categories {
		catList = append(catList, fmt.Sprintf("- Name: %s, Slug: %s", cat.Name, cat.Slug))
	}
	categoriesStr := strings.Join(catList, "\n")

	return fmt.Sprintf(`Analyze the fashion item image and output a strict JSON payload mapping directly to the following database fields.
All values for the JSON keys (color, style, material, pattern, fit, seasonality, and description) MUST be written entirely in natural, accurate Vietnamese fashion prose.
Do not limit your analysis to the examples provided in the schema; use your broader fashion knowledge to identify the exact attributes.

You MUST classify the item into one of the following categories based on the image. Choose the single most appropriate slug from this list:
%s

If the image does not contain any fashion, apparel, clothing, footwear, or accessory item, you MUST return a JSON containing ONLY:
{
  "error": "No fashion item detected"
}

JSON Schema format:
{
  "category_slug": "The selected category slug from the list (e.g. 'ao', 'quan', etc.)",
  "color": "Color name in natural Vietnamese fashion context, including shades (e.g. 'Đen mun', 'Trắng sữa', 'Xanh olive', etc.)",
  "style": "Fashion style in Vietnamese (e.g. 'Đường phố', 'Tối giản', 'Y2K', 'Cổ điển', etc.)",
  "material": "Material composition in Vietnamese (e.g. 'Vải Denim co giãn', 'Vải thun Cotton dày', 'Len dệt kim', etc.)",
  "pattern": "Pattern details or graphic prints in Vietnamese (e.g. 'In graphic lớn ở lưng', 'Trơn không họa tiết', 'Kẻ sọc ngang', etc.)",
  "fit": "Silhouette or fit type in Vietnamese (e.g. 'Dáng rộng Oversized', 'Dáng ôm Body', 'Dáng đứng Regular', etc.)",
  "seasonality": "Weather context or suitable seasons in Vietnamese (e.g. 'Thích hợp mùa đông', 'Quanh năm', 'Mùa hè và mùa xuân', etc.)",
  "description": "Write a highly descriptive 1-2 sentence paragraph in Vietnamese on a SINGLE LINE (absolutely NO newline characters). Focus on specific visual cues, unique design highlights, printing placements, exact text printed on the garment (keep exact typography), collar types, hardware (zippers/buttons), or fabric texture (e.g. 'Áo thun đen cổ tròn dáng rộng tay lửng bằng chất cotton dày dặn, có hình in họa tiết cyberpunk lớn ở mặt lưng kèm dòng chữ trắng NEON DISTRICT thêu sắc nét.')"
}
Output ONLY the raw JSON object. Do NOT wrap the response in markdown code blocks (e.g. do not use backticks or markdown formatting).`, categoriesStr)
}

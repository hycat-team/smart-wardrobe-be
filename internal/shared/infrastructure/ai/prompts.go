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
All values for color, style, material, pattern, fit, and seasonality MUST be written in natural, accurate Vietnamese fashion context.
Do not limit your analysis to the examples provided in the schema; use your broader fashion knowledge to identify the exact attributes.

You MUST classify the item into one of the following categories based on the image. Choose the single most appropriate slug from this list:
%s

For the "description" field, you must build a comprehensive, high-density English string optimized for vector embeddings. 
Structure the "description" strictly by combining two parts with NO newline characters:
First Part (Structured Tokens): "[CAT:<category>][COL:<color>][STY:<style>][MAT:<material>][PAT:<pattern>][FIT:<fit>][SEA:<seasonality>]" using industry-standard English fashion terms.
Second Part (Free-form Prose): A detailed, 1-2 sentence English paragraph capturing specific visual cues, unique design highlights, printing placements, exact text printed on the garment, collar types, hardware (zippers/buttons), or texture details visible in the image.

If the image does not contain any fashion, apparel, clothing, footwear, or accessory item, you MUST return a JSON containing ONLY:
{
  "error": "No fashion item detected"
}

JSON Schema format:
{
  "category_slug": "The selected category slug from the list (e.g. 'ao', 'quan', etc.)",
  "color": "Tên màu sắc bằng Tiếng Việt tự nhiên bao gồm sắc độ (ví dụ: Rêu quân đội, Trắng kem, ...)",
  "style": "Phong cách thời trang bằng Tiếng Việt (ví dụ: Đường phố, Tối giản, Y2K, Grunge, Cổ điển, ...)",
  "material": "Thành phần chất liệu vải bóc tách được bằng Tiếng Việt (ví dụ: Vải Denim, Vải thun Cotton, Len dệt, ...)",
  "pattern": "Chi tiết họa tiết hoặc bề mặt đồ họa bằng Tiếng Việt (ví dụ: In mặt sau, Trơn, Thêu chữ, Kẻ sọc, ...)",
  "fit": "Thông số form dáng hay cấu trúc silhouette bằng Tiếng Việt (ví dụ: Rộng rãi, Dáng lửng, Vừa vặn, ...)",
  "seasonality": "Bối cảnh thời tiết hoặc mùa phù hợp trong năm bằng Tiếng Việt (ví dụ: Mùa hè/Mùa xuân, Quanh năm, ...)",
  "description": "[CAT:Tops][COL:Black][STY:Streetwear][MAT:HeavyweightCotton][PAT:BackGraphicPrint][FIT:Oversized][SEA:AllSeason] Oversized black crewneck t-shirt featuring a large retro cyberpunk graphic print on the back with explicit white typography reading 'NEON DISTRICT', crafted from thick structured cotton with drop shoulders."
}
Output ONLY the raw JSON object, without any markdown block tags.`, categoriesStr)
}

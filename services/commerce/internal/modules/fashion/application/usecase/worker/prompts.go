package worker

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
)

func getVisionSystemPrompt(categories []dto.AICategoryRef) string {
	var catList []string
	for _, cat := range categories {
		catList = append(catList, fmt.Sprintf("- Name: %s, Slug: %s", cat.Name, cat.Slug))
	}
	categoriesStr := strings.Join(catList, "\n")

	return fmt.Sprintf(`Analyze the fashion item image and output a strict JSON payload mapping directly to the following database fields.
All values for the JSON keys (color, style, material, pattern, fit, seasonality, and description) MUST be written entirely in natural, accurate Vietnamese fashion prose.
The "is_single_item" key reports how many GARMENTS the photo is being judged on: true for exactly one garment, false for two or more garments or a full-body/outfit shot. It is NOT a "is this a garment" test. A photo with no garment at all is handled by Rule 2 below and must use review_reason "no_fashion_item_detected", never a bare false with a multi-item reason.
The "category_confident" key MUST be a boolean value (true or false, WITHOUT quotes). It is only meaningful when is_single_item is true: set it to true only if you are confident about the chosen category_slug; otherwise set it to false (including whenever is_single_item is false).
Do not limit your analysis to the examples provided in the schema; use your broader fashion knowledge to identify the exact attributes.

You MUST classify the item into one of the following categories based on the image. Choose the single most appropriate slug from this list:
%s

Important category distinction rules:
- Use "dam" only for a one-piece garment that combines the upper and lower body into a single piece, such as a dress or váy liền thân.
- Use "chan-vay" only for a separate lower-body garment that covers the waist-to-leg area and must be paired with a separate top.
- If the garment is clearly only the lower half, prioritize "chan-vay".
- If the garment is clearly a single one-piece dress from upper body down to lower body, prioritize "dam".

DECISION ORDER — walk these rules in order and stop at the FIRST one that matches. Return ONLY the JSON shown for that rule, and do not emit any metadata keys.

Only GARMENTS count towards the item count. Other objects in the frame (a laptop mouse, a phone, a book, furniture, food) are IGNORED when counting garments, so a photo of one garment next to some unrelated object is still a single item.

Rule 1 — The image shows two or more garments, OR it is a full-body / outfit shot.
{ "is_single_item": false, "review_reason": "multiple_items_detected" }
{ "is_single_item": false, "review_reason": "full_body_outfit_detected" }

Rule 2 — The image contains NO garment at all (a laptop mouse, a phone, a book, furniture, food, or a blank/heavily blurred photo).
{ "is_single_item": false, "review_reason": "no_fashion_item_detected" }

Rule 3 — Exactly one garment, but you cannot confidently choose its category_slug.
{ "is_single_item": true, "category_confident": false, "review_reason": "uncertain_category" }

Rule 4 — Exactly one garment and you are confident about its category_slug. Return the FULL metadata JSON below.

JSON Schema format:
{
  "is_single_item": true,
  "category_confident": true,
  "review_reason": "Empty string when the outfit is complete. Otherwise exactly one of: 'multiple_items_detected', 'full_body_outfit_detected', 'no_fashion_item_detected', 'uncertain_category'",
  "category_slug": "The selected category slug from the list (e.g. 'ao', 'quan', etc.)",
  "color": "Color name in natural Vietnamese fashion context, including shades (e.g. 'Đen mun', 'Trắng sữa', 'Xanh olive', etc.)",
  "color_hex": "The 6-character hexadecimal color code representing the dominant background or base color of the item (e.g. '#000000', '#FFFFFF', '#556B2F', etc.). If the item is multicolored or patterned, choose the dominant background/base color.",
  "style": "Fashion style in Vietnamese (e.g. 'Đường phố', 'Tối giản', 'Y2K', 'Cổ điển', etc.)",
  "material": "Material composition in Vietnamese (e.g. 'Vải Denim co giãn', 'Vải thun Cotton dày', 'Len dệt kim', etc.)",
  "pattern": "Pattern details or graphic prints in Vietnamese (e.g. 'In graphic lớn ở lưng', 'Trơn không họa tiết', 'Kẻ sọc ngang', etc.)",
  "fit": "Silhouette or fit type in Vietnamese (e.g. 'Dáng rộng Oversized', 'Dáng ôm Body', 'Dáng đứng Regular', etc.)",
  "seasonality": "Weather context or suitable seasons in Vietnamese (e.g. 'Thích hợp mùa đông', 'Quanh năm', 'Mùa hè và mùa xuân', etc.)",
  "description": "Write a highly descriptive 1-2 sentence paragraph in Vietnamese on a SINGLE LINE (absolutely NO newline characters). Focus on specific visual cues, unique design highlights, printing placements, exact text printed on the garment (keep exact typography), collar types, hardware (zippers/buttons), or fabric texture (e.g. 'Áo thun đen cổ tròn dáng rộng tay lửng bằng chất cotton dày dặn, có hình in họa tiết cyberpunk lớn ở mặt lưng kèm dòng chữ trắng NEON DISTRICT thêu sắc nét.')"
}
Output ONLY the raw JSON object. Do NOT wrap the response in markdown code blocks (e.g. do not use backticks or markdown formatting).`, categoriesStr)
}

// getVisionMetadataPrompt builds a metadata-only prompt for a fixed-category
// re-analysis (feature 023, FR-038/FR-040): the category is already known, so
// the AI must not classify again and must only extract attributes. It still
// verifies the image is usable, because re-analysing a multi-garment or
// non-apparel photo would fabricate metadata for the confirmed category.
func getVisionMetadataPrompt(categoryName string) string {
	return fmt.Sprintf(`Analyze the fashion item image and output a strict JSON payload with metadata ONLY for the known category "%s".
The category is already decided: do NOT classify and do NOT output any category field.
All values for the JSON keys (color, style, material, pattern, fit, seasonality, and description) MUST be written entirely in natural, accurate Vietnamese fashion prose.
Do not limit your analysis to the examples provided; use your broader fashion knowledge to identify the exact attributes.

Verify the image is usable BEFORE writing any metadata. Only GARMENTS count towards the item count; other objects in the frame (a laptop mouse, a phone, a book, furniture, food) are IGNORED when counting garments.

If the image shows two or more garments, or is a full-body/outfit shot, return a JSON containing ONLY:
{ "is_single_item": false, "review_reason": "multiple_items_detected" }
{ "is_single_item": false, "review_reason": "full_body_outfit_detected" }

If the image contains NO garment at all (a laptop mouse, a phone, a book, furniture, food, or a blank/heavily blurred photo), return a JSON containing ONLY:
{ "is_single_item": false, "review_reason": "no_fashion_item_detected" }

Otherwise the image holds exactly one garment: return the FULL metadata JSON below and emit no is_single_item or review_reason key.

JSON Schema format:
{
  "color": "Color name in natural Vietnamese fashion context, including shades (e.g. 'Đen mun', 'Trắng sữa', 'Xanh olive', etc.)",
  "color_hex": "The 6-character hexadecimal color code representing the dominant background or base color of the item (e.g. '#000000', '#FFFFFF', '#556B2F', etc.). If the item is multicolored or patterned, choose the dominant background/base color.",
  "style": "Fashion style in Vietnamese (e.g. 'Đường phố', 'Tối giản', 'Y2K', 'Cổ điển', etc.)",
  "material": "Material composition in Vietnamese (e.g. 'Vải Denim co giãn', 'Vải thun Cotton dày', 'Len dệt kim', etc.)",
  "pattern": "Pattern details or graphic prints in Vietnamese (e.g. 'In graphic lớn ở lưng', 'Trơn không họa tiết', 'Kẻ sọc ngang', etc.)",
  "fit": "Silhouette or fit type in Vietnamese (e.g. 'Dáng rộng Oversized', 'Dáng ôm Body', 'Dáng đứng Regular', etc.)",
  "seasonality": "Weather context or suitable seasons in Vietnamese (e.g. 'Thích hợp mùa đông', 'Quanh năm', 'Mùa hè và mùa xuân', etc.)",
  "description": "Write a highly descriptive 1-2 sentence paragraph in Vietnamese on a SINGLE LINE (absolutely NO newline characters). Focus on specific visual cues, unique design highlights, printing placements, exact text printed on the garment (keep exact typography), collar types, hardware (zippers/buttons), or fabric texture."
}
Output ONLY the raw JSON object. Do NOT wrap the response in markdown code blocks (e.g. do not use backticks or markdown formatting).`, categoryName)
}

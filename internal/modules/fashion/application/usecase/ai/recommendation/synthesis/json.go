// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"encoding/json"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

// ParseOutfitRecommendationJSON phân tích cú pháp chuỗi JSON phản hồi của mô hình AI, tự động làm sạch các thẻ markdown block (ví dụ: ```json) và xử lý văn bản thừa xung quanh.
//
// Hành vi:
// 1. Dùng [CleanJSONMarkdown] để xóa các định dạng markdown code block.
// 2. Thử Unmarshal trực tiếp chuỗi đã làm sạch vào [LlmOutfitResponse]. Nếu thành công, xác thực payload qua [ValidateOutfitRecommendationPayload] và trả về.
// 3. Nếu thất bại, sử dụng [ExtractFirstJSONObject] để tìm và cắt ra đối tượng JSON hợp lệ đầu tiên trong chuỗi.
// 4. Unmarshal đối tượng JSON đã cắt được, xác thực tính hợp lệ của dữ liệu và trả về kết quả.
//
// Đầu vào mẫu:
//
//	responseText: "Dưới đây là gợi ý phối đồ:\n```json\n{\"title\": \"Phong cách công sở\", ...}\n```"
//
// Đầu ra mẫu:
//
//	(types.LlmOutfitResponse{Title: "Phong cách công sở", ...}, "{\"title\": \"Phong cách công sở\", ...}", nil)
func ParseOutfitRecommendationJSON(responseText string) (types.LlmOutfitResponse, string, error) {
	cleaned := stringutils.CleanJSONMarkdown(responseText)

	var result types.LlmOutfitResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		return result, cleaned, ValidateOutfitRecommendationPayload(result)
	}

	extracted := ExtractFirstJSONObject(cleaned)
	if extracted == "" {
		return types.LlmOutfitResponse{}, cleaned, fmt.Errorf("could not extract JSON object from response")
	}

	err := json.Unmarshal([]byte(extracted), &result)
	if err != nil {
		return result, extracted, err
	}

	return result, extracted, ValidateOutfitRecommendationPayload(result)
}

// ValidateOutfitRecommendationPayload kiểm tra và từ chối các phản hồi chứa dữ liệu mẫu hoặc placeholder do mô hình AI trả về (như "uuid", "string", v.v.) để đảm bảo tính toàn vẹn dữ liệu.
func ValidateOutfitRecommendationPayload(payload types.LlmOutfitResponse) error {
	if isPlaceholderValue(payload.Title) || isPlaceholderValue(payload.Explanation) {
		return fmt.Errorf("placeholder values detected in recommendation payload")
	}

	for _, item := range payload.Items {
		if isPlaceholderValue(item.Role) || isPlaceholderValue(item.PrimaryID) {
			return fmt.Errorf("placeholder values detected in recommendation items")
		}
		for _, altID := range item.AlternativeIDs {
			if isPlaceholderValue(altID) {
				return fmt.Errorf("placeholder values detected in recommendation alternatives")
			}
		}
	}

	return nil
}

// isPlaceholderValue kiểm tra xem một chuỗi có phải là giá trị placeholder phổ biến mà AI hay trả về khi không có dữ liệu thực tế hay không.
func isPlaceholderValue(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "string", "uuid", "role", "primary_id", "alternative_ids", "title", "explanation":
		return normalized != ""
	}

	return false
}

// ExtractFirstJSONObject tìm kiếm và trích xuất chuỗi đối tượng JSON hợp lệ đầu tiên từ một chuỗi văn bản tự do bằng cách đếm số lượng đóng mở ngoặc nhọn lồng nhau.
//
// Hành vi:
// 1. Tìm vị trí xuất hiện của ký tự '{' đầu tiên. Nếu không có, trả về chuỗi rỗng.
// 2. Duyệt qua từng ký tự tiếp theo, bỏ qua các ký tự nằm trong chuỗi ký tự được bọc bởi dấu nháy kép (inString = true) và các ký tự escaped (ví dụ: \").
// 3. Đếm độ sâu của dấu ngoặc nhọn: tăng [depth] khi gặp '{' và giảm [depth] khi gặp '}'.
// 4. Khi [depth] trở về 0, cắt chuỗi từ vị trí '{' bắt đầu đến vị trí '}' hiện tại và trả về.
//
// Đầu vào mẫu:
//
//	value: "Kết quả: {\"key\": \"value\"} Cảm ơn bạn!"
//
// Đầu ra mẫu:
//
//	"{\"key\": \"value\"}"
func ExtractFirstJSONObject(value string) string {
	start := strings.IndexByte(value, '{')
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(value); i++ {
		ch := value[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(value[start : i+1])
			}
		}
	}

	return ""
}

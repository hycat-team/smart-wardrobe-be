// Package retrieval implements candidate retrieval, taxonomy term expansion, and lexical/semantic query rewriting.
package retrieval

import (
	"sort"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
)

// BuildRecommendationSemanticQuery định dạng các trường thuộc tính ý định đã phân tích thành một chuỗi truy vấn phân tách bằng ký tự "|" để làm đầu vào cho mô hình Vector Embedding.
//
// Hành vi:
// 1. Duyệt qua các trường Occasion, StyleTarget, ColorTone, PositiveConstraints, NegativeConstraints.
// 2. Định dạng từng trường dưới dạng nhãn tương ứng (ví dụ: "occasion: work, casual").
// 3. Ghép nối thông tin chi tiết gốc (original details) nếu có, gắn tiền tố "details context: " hoặc "details: " tùy thuộc vào việc có cấu hình tùy chọn tường minh hay không.
// 4. Ghép toàn bộ các thành phần bằng ký tự " | " và trả về chuỗi kết quả.
//
// Đầu vào mẫu:
//
//	intent: dto.ParsedIntent{Occasion: []string{"work"}, StyleTarget: []string{"minimalist"}}
//	originalDetails: "Đi làm hàng ngày"
//	hasExplicitOptions: false
//
// Đầu ra mẫu:
//
//	"occasion: work | style: minimalist | details: Đi làm hàng ngày"
func BuildRecommendationSemanticQuery(
	intent dto.ParsedIntent,
	originalDetails string,
	hasExplicitOptions bool,
) string {
	var parts []string
	if len(intent.Occasion) > 0 {
		parts = append(parts, "occasion: "+strings.Join(intent.Occasion, ", "))
	}
	if len(intent.StyleTarget) > 0 {
		parts = append(parts, "style: "+strings.Join(intent.StyleTarget, ", "))
	}
	if len(intent.ColorTone) > 0 {
		parts = append(parts, "color tone: "+strings.Join(intent.ColorTone, ", "))
	}
	if len(intent.PositiveConstraints) > 0 {
		parts = append(parts, "constraints: "+strings.Join(intent.PositiveConstraints, ", "))
	}
	if len(intent.NegativeConstraints) > 0 {
		parts = append(parts, "avoid: "+strings.Join(intent.NegativeConstraints, ", "))
	}

	details := strings.TrimSpace(originalDetails)
	if details != "" {
		if hasExplicitOptions {
			parts = append(parts, "details context: "+details)
		} else {
			parts = append(parts, "details: "+details)
		}
	}

	return strings.Join(parts, " | ")
}

// ExtractTermStrings chuyển đổi một lát cắt cấu trúc [RetrievalTerm] thành một lát cắt chuỗi thô để dễ dàng xử lý hoặc hiển thị.
func ExtractTermStrings(terms []types.RetrievalTerm) []string {
	strs := make([]string, len(terms))
	for i, term := range terms {
		strs[i] = term.Value
	}
	return strs
}

// NormalizeTermSet thực hiện loại bỏ trùng lặp, chuẩn hóa chữ thường, lọc bỏ các từ dừng (stopwords) và sắp xếp tăng dần bảng chữ cái cho một lát cắt chuỗi.
func NormalizeTermSet(terms []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || parser.LexicalStopwords[term] {
			continue
		}
		if !seen[term] {
			seen[term] = true
			normalized = append(normalized, term)
		}
	}
	sort.Strings(normalized)
	return normalized
}

// AppendWithoutDuplicate thêm một chuỗi vào lát cắt chuỗi hiện tại nếu nó chưa tồn tại (không phân biệt chữ hoa chữ thường) và sắp xếp lại lát cắt.
func AppendWithoutDuplicate(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}
	values = append(values, value)
	sort.Strings(values)
	return values
}

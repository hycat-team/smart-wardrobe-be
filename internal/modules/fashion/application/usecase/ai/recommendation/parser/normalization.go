package parser

import (
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
)

// NormalizeText thực hiện chuẩn hóa văn bản đầu vào: chuyển thành chữ thường, thay thế từ phủ định đặc biệt,
// chèn khoảng trắng đệm xung quanh các dấu câu, loại bỏ dấu tiếng Việt và loại bỏ các ký tự Telex thừa ở cuối từ.
func NormalizeText(text string) string {
	lowered := strings.ToLower(text)
	lowered = strings.ReplaceAll(lowered, "đừng", "dung_neg")
	lowered = strings.ReplaceAll(lowered, "dungf", "dung_neg")

	padded := lowered
	padded = strings.ReplaceAll(padded, ",", " , ")
	padded = strings.ReplaceAll(padded, ";", " ; ")
	padded = strings.ReplaceAll(padded, "-", " - ")
	padded = strings.ReplaceAll(padded, ".", " . ")

	replaced := shared.RemoveVietnameseSigns(padded)
	return stripOrphanTelex(replaced)
}

// stripOrphanTelex tìm kiếm các từ có đuôi là ký tự Telex (s, f, r, x, j, w) và loại bỏ ký tự đuôi này nếu từ gốc có chứa nguyên âm
// và từ đó không nằm trong danh sách các từ tiếng Anh được bảo vệ [protectedWords].
func stripOrphanTelex(text string) string {
	return reOrphanTelex.ReplaceAllStringFunc(text, func(match string) string {
		if protectedWords[match] {
			return match
		}
		prefix := match[:len(match)-1]
		hasVowel := false
		for i := 0; i < len(prefix); i++ {
			c := prefix[i]
			if c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u' || c == 'y' {
				hasVowel = true
				break
			}
		}
		if hasVowel {
			return prefix
		}
		return match
	})
}

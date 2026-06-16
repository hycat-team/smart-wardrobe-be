package parser

import (
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
)

// NormalizeText applies Telex orphan stripping, punctuation padding, and Vietnamese diacritics removal.
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

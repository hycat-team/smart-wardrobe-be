// Package shared holds helpers used by several personalization use cases
// (style profile, style learning, feedback and outfit completion).
package shared

import (
	"slices"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/retrieval"
	"smart-wardrobe-be/shared/pkg/utils/stringutils"
)

// taxonomyStyleGroup is the existing recommendation taxonomy group that owns the
// canonical style vocabulary. Normalisation reads it instead of keeping a second
// list of synonyms.
const taxonomyStyleGroup = "style"

// styleDescriptions enriches a canonical style label into the Vietnamese
// sentence fragment used to build the style embedding. Values not present here
// are passed through as-is, so an unknown label still reaches the embedding.
var styleDescriptions = map[string]string{
	"minimalist": "tối giản, ít chi tiết, đường nét gọn gàng, tông màu trung tính và dễ phối",
	"vintage":    "cổ điển, hoài niệm, chất liệu và họa tiết mang hơi hướng retro",
	"streetwear": "đường phố, năng động, cá tính, thoải mái theo tinh thần hiphop",
	"preppy":     "học đường, sinh viên, gọn gàng và thanh lịch trẻ trung",
	"elegant":    "thanh lịch, sang trọng, tinh tế, phù hợp với dịp trang trọng",
}

// canonicalStyleKeys keeps iteration deterministic and documents the canonical
// vocabulary mirrored by the taxonomy's style group.
var canonicalStyleKeys = []string{"elegant", "minimalist", "preppy", "streetwear", "vintage"}

// NormalizeStyleLabels maps values that belong to the existing style taxonomy to
// their canonical label (case-insensitive, Vietnamese diacritics ignored) and
// keeps unknown values unchanged after trimming. Duplicates are dropped while
// preserving the input order.
func NormalizeStyleLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(labels))
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		canonical := canonicalStyleLabel(label)
		if canonical == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(canonical))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, canonical)
	}
	return normalized
}

// BuildStyleEmbeddingText builds the Vietnamese text embedded for a style
// profile. Known styles are enriched through the lexicon; unknown values are
// included as-is. Labels are sorted so that reordering the same set produces the
// same text (and therefore the same embedding hash). An empty profile yields an
// empty string, which callers treat as "nothing to embed".
func BuildStyleEmbeddingText(styles, excludedStyles []string) string {
	preferred := sortedNormalizedLabels(styles)
	excluded := sortedNormalizedLabels(excludedStyles)

	parts := make([]string, 0, 2)
	if len(preferred) > 0 {
		parts = append(parts, "Phong cách yêu thích: "+strings.Join(describeStyles(preferred), "; "))
	}
	if len(excluded) > 0 {
		parts = append(parts, "Phong cách muốn tránh: "+strings.Join(describeStyles(excluded), "; "))
	}
	return strings.Join(parts, ". ")
}

func sortedNormalizedLabels(labels []string) []string {
	normalized := NormalizeStyleLabels(labels)
	slices.Sort(normalized)
	return normalized
}

func describeStyles(labels []string) []string {
	descriptions := make([]string, 0, len(labels))
	for _, label := range labels {
		if description, ok := styleDescriptions[label]; ok {
			descriptions = append(descriptions, description)
			continue
		}
		descriptions = append(descriptions, label)
	}
	return descriptions
}

// canonicalStyleLabel resolves a known style value to its canonical label. It
// first matches the canonical label itself, then the Vietnamese/English synonyms
// registered by the recommendation taxonomy.
func canonicalStyleLabel(label string) string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return ""
	}

	lowered := strings.ToLower(trimmed)
	if _, ok := styleDescriptions[lowered]; ok {
		return lowered
	}

	unaccented := strings.ToLower(strings.TrimSpace(stringutils.RemoveVietnameseSigns(trimmed)))
	for _, canonical := range canonicalStyleKeys {
		if unaccented == canonical {
			return canonical
		}
		for _, term := range retrieval.ExpandTaxonomyTermValues(taxonomyStyleGroup, []string{canonical}) {
			if unaccented == term {
				return canonical
			}
		}
	}

	return trimmed
}

package completion

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
)

// completionGapRoles returns the wearing roles a complete outfit still needs.
//
// The core structure is a dress (fullbody) OR a top with a bottom, plus
// footwear; a dress replaces top+bottom exactly like the outfit structure rules.
// Outerwear is an additional, context-driven role: it is only expected when the
// request asks for a layer (cold/cool/rainy weather, autumn/winter season, or an
// explicit layering intent) and the shared fallback decision says so.
func completionGapRoles(present map[string]bool, input dto.RecommendOutfitReq) []string {
	required := make([]string, 0, 4)
	if present["fullbody"] {
		required = append(required, "fullbody")
	} else {
		required = append(required, "top", "bottom")
	}
	required = append(required, "footwear")
	if ranking.ShouldIncludeOuterwear(input) {
		required = append(required, "outerwear")
	}

	gaps := make([]string, 0, len(required))
	for _, role := range required {
		if !present[role] {
			gaps = append(gaps, role)
		}
	}
	if len(gaps) == 0 {
		return nil
	}
	return gaps
}

// completionContext projects the completion request onto the recommendation
// request shape the shared suggestion builder understands. Brand items are
// never suggested for completion, so IncludeBrandItems stays nil.
func completionContext(input dto.CompleteOutfitReq) dto.RecommendOutfitReq {
	return dto.RecommendOutfitReq{
		Occasion:    input.Occasion,
		StyleTarget: input.StyleTarget,
		Season:      input.Season,
		Weather:     input.Weather,
		ColorTone:   input.ColorTone,
		Details:     input.Details,
	}
}

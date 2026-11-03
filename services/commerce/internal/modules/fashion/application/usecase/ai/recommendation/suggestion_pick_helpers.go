package recommendation

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"

	"github.com/google/uuid"
)

// This file holds the part of the catalog suggestion builder that turns
// retrieved candidates into response entries. It is shared by the recommendation
// gap-fill (033 US2) and the outfit-completion endpoint (033 US7), so both
// enforce the same FR-021/FR-023 rules.

// pickCatalogSuggestions applies the FR-023 caps to the retrieved candidates
// and maps them into the response entries. presentRoles carries the roles the
// outfit already covers and only changes the Vietnamese reason wording, never
// the selection.
func (uc *OutfitRecommendationUseCase) pickCatalogSuggestions(
	gapRoles []string,
	candidates []catalogSuggestion,
	presentRoles map[string]bool,
	personalization types.PersonalizationContext,
) []dto.OutfitSuggestionRes {
	maxTotal, maxPerRole := uc.suggestionCaps()
	used := map[uuid.UUID]bool{}
	suggestions := make([]dto.OutfitSuggestionRes, 0, maxTotal)
	for _, role := range gapRoles {
		for picked := 0; picked < maxPerRole && len(suggestions) < maxTotal; picked++ {
			best := bestCatalogSuggestion(candidates, role, used, personalization)
			if best == nil {
				break
			}
			used[best.FashionItem.ID] = true
			suggestions = append(suggestions, mapper.MapOutfitSuggestionRes(
				role,
				suggestionReason(role, presentRoles[role], personalization),
				best.FashionItem,
			))
		}
	}
	return suggestions
}

// suggestionCaps resolves the configured response caps. Non-positive values fall
// back to the shipped defaults and FR-023 keeps the per-role cap at one even
// when the config asks for more.
func (uc *OutfitRecommendationUseCase) suggestionCaps() (int, int) {
	maxTotal := defaultSuggestionMaxTotal
	maxPerRole := defaultSuggestionMaxPerRole
	if uc != nil && uc.cfg != nil {
		if uc.cfg.Personalization.SuggestionMaxTotal > 0 {
			maxTotal = uc.cfg.Personalization.SuggestionMaxTotal
		}
		if uc.cfg.Personalization.SuggestionMaxPerRole > 0 {
			maxPerRole = uc.cfg.Personalization.SuggestionMaxPerRole
		}
	}
	if maxPerRole > 1 {
		maxPerRole = 1
	}
	return maxTotal, maxPerRole
}

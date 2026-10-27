package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

// IOutfitCompletionUseCase answers "what is missing from this set of items I
// already own?" (feature 033, US7; contracts/personalization-api.md §2.3).
type IOutfitCompletionUseCase interface {
	// CompleteOutfit validates ownership of every requested fashion item, works
	// out the wearing roles the set is missing and returns catalog suggestions
	// for those roles. It never runs the LLM, never synthesizes an outfit, never
	// writes anything and never consumes the daily outfit quota (FR-024).
	CompleteOutfit(
		ctx context.Context,
		userID uuid.UUID,
		input dto.CompleteOutfitReq,
	) (*dto.CompleteOutfitRes, error)
}

// IOutfitSuggestionBuilder builds the additive catalog suggestions of feature
// 033 (US2 and US7). It is implemented by the recommendation use case so that
// the completion endpoint reuses exactly the same retrieval query, hard
// filters, personalization blend, ownership exclusion, response caps and
// Vietnamese reason template instead of duplicating them.
type IOutfitSuggestionBuilder interface {
	// BuildOutfitCompletionSuggestions returns at most one catalog item per
	// requested gap role. It is fail-soft: a lookup failure yields no
	// suggestions rather than an error.
	BuildOutfitCompletionSuggestions(
		ctx context.Context,
		userID uuid.UUID,
		input dto.RecommendOutfitReq,
		gapRoles []string,
	) []dto.OutfitSuggestionRes
}

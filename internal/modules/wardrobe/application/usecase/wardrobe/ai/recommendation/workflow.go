package recommendation

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// RecommendOutfit generates an outfit recommendation from the user's active wardrobe items.
func (uc *OutfitRecommendationUseCase) RecommendOutfit(
	ctx context.Context,
	userID uuid.UUID,
	input dto.RecommendOutfitReq,
) (*dto.RecommendedOutfitRes, error) {
	activeItems, quotaDTO, err := uc.validateAndGetActiveItems(ctx, userID)
	if err != nil {
		return nil, err
	}

	candidates, err := uc.filterCandidates(ctx, userID, activeItems, input)
	if err != nil {
		return nil, err
	}

	finalRes, err := uc.generateOutfitRecommendation(ctx, candidates, input)
	if err != nil {
		finalRes = runLocalHSLMatching(candidates, input)
	}

	if err := uc.updateQuotaAndConstructResponse(ctx, userID, finalRes, quotaDTO); err != nil {
		return nil, err
	}

	return finalRes, nil
}

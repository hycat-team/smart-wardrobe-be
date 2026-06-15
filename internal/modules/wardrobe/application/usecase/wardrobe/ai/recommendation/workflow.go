package recommendation

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
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
		failureKind, providerHint, promptLen, responsePreview := classifyFallbackTrace(err)
		uc.logger.Warn("Outfit recommendation AI fallback triggered",
			zap.String("failure_kind", failureKind),
			zap.String("provider_hint", providerHint),
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.Int("candidate_count", len(candidates)),
			zap.Int("prompt_len", promptLen),
			zap.String("response_preview", responsePreview),
		)
		finalRes = runLocalHSLMatching(candidates, input)
	}

	if err := uc.updateQuotaAndConstructResponse(ctx, userID, finalRes, quotaDTO); err != nil {
		return nil, err
	}

	return finalRes, nil
}

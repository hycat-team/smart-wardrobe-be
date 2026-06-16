package recommendation

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/synthesis"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"

	"github.com/google/uuid"
)

func (uc *OutfitRecommendationUseCase) generateOutfitRecommendation(
	ctx context.Context,
	candidates []types.CandidateForPrompt,
	input dto.RecommendOutfitReq,
) (*dto.RecommendedOutfitRes, error) {
	return synthesis.GenerateOutfitRecommendation(ctx, uc.aiService, candidates, input)
}

func (uc *OutfitRecommendationUseCase) updateQuotaAndConstructResponse(
	ctx context.Context,
	userID uuid.UUID,
	finalRes *dto.RecommendedOutfitRes,
	quotaDTO *contract.UserSubscriptionDTO,
) error {
	if err := uc.userQuotaCtr.UpdateOutfitQuota(ctx, userID, 1); err != nil {
		return err
	}

	updatedQuota, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err == nil {
		finalRes.RemainingQuota = updatedQuota.AiOutfitDailyQuota - updatedQuota.OutfitRecommendCount
	} else {
		finalRes.RemainingQuota = quotaDTO.AiOutfitDailyQuota - (quotaDTO.OutfitRecommendCount + 1)
	}

	if finalRes.RemainingQuota < 0 {
		finalRes.RemainingQuota = 0
	}

	return nil
}

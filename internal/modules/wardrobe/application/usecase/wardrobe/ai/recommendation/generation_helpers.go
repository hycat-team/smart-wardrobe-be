package recommendation

import (
	"context"
	"encoding/json"
	"fmt"

	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
)

func (uc *OutfitRecommendationUseCase) generateOutfitRecommendation(
	ctx context.Context,
	candidates []CandidateForPrompt,
	input dto.RecommendOutfitReq,
) (*dto.RecommendedOutfitRes, error) {
	responseText, err := uc.aiService.GenerateText(
		ctx,
		"You are a professional AI fashion stylist. Return ONLY a valid JSON payload for the outfit recommendation. Do not include markdown code block formatting.",
		buildRecommendationPrompt(candidates, input),
	)
	if err != nil {
		return nil, err
	}

	if responseText == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	var llmRes llmOutfitResponse
	if err := json.Unmarshal(
		[]byte(stringutils.CleanJSONMarkdown(responseText)),
		&llmRes,
	); err != nil {
		return nil, err
	}

	validGroups := uc.mapLLMResponseToGroups(candidates, llmRes)
	if len(validGroups) == 0 {
		return nil, wardrobeerrors.ErrInvalidOutfitStructure
	}

	return &dto.RecommendedOutfitRes{
		Title:       llmRes.Title,
		Explanation: llmRes.Explanation,
		Items:       validGroups,
	}, nil
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

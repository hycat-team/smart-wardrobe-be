package recommendation

import (
	"context"
	"sort"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func (uc *OutfitRecommendationUseCase) validateAndGetActiveItems(
	ctx context.Context,
	userID uuid.UUID,
) ([]*entities.WardrobeItem, *contract.UserSubscriptionDTO, error) {
	quotaDTO, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	if quotaDTO.OutfitRecommendCount >= quotaDTO.AiOutfitDailyQuota {
		return nil, nil, subscriptionerrors.ErrAiOutfitQuotaExceeded
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, nil, err
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	items = shared.FilterActiveItems(items, subOverview.MaxWardrobeItems)
	activeItems := make([]*entities.WardrobeItem, 0, len(items))
	for _, item := range items {
		if item.Status == wardrobestatus.InWardrobe && !item.IsDeleted {
			activeItems = append(activeItems, item)
		}
	}

	if len(activeItems) < 5 {
		return nil, nil, wardrobeerrors.ErrMinimumWardrobeItemsRequired
	}

	return activeItems, quotaDTO, nil
}

func (uc *OutfitRecommendationUseCase) filterCandidates(
	ctx context.Context,
	userID uuid.UUID,
	activeItems []*entities.WardrobeItem,
	input dto.RecommendOutfitReq,
) ([]CandidateForPrompt, error) {
	intent := uc.parseRecommendationIntent(input)
	queryVector := uc.buildRecommendationQueryVector(ctx, intent)

	candidates, err := uc.wardrobeRepo.GetHybridCandidates(
		ctx,
		userID,
		queryVector,
		intent.ExactKeywords,
		40,
	)
	if err != nil {
		return nil, err
	}

	candidates = uc.ensureMinimumCandidatePool(candidates, activeItems)

	return uc.rankCandidates(candidates, intent), nil
}

func (uc *OutfitRecommendationUseCase) parseRecommendationIntent(
	input dto.RecommendOutfitReq,
) dto.ParsedIntent {
	freeText := ""
	if input.Details != nil {
		freeText = *input.Details
	}

	intent := uc.nlpParser.Parse(freeText)
	if input.Occasion != nil && *input.Occasion != "" {
		intent.Occasion = *input.Occasion
	}
	if input.ColorTone != nil && *input.ColorTone != "" {
		intent.ColorTone = *input.ColorTone
	}

	return intent
}

func (uc *OutfitRecommendationUseCase) buildRecommendationQueryVector(
	ctx context.Context,
	intent dto.ParsedIntent,
) entities.Vector {
	if intent.SemanticQuery == "" {
		return nil
	}

	embedCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	vecs, err := uc.aiService.GenerateEmbeddings(embedCtx, []string{intent.SemanticQuery})
	if err != nil || len(vecs) == 0 || len(vecs[0]) != 768 {
		return nil
	}

	return entities.Vector(vecs[0])
}

func (uc *OutfitRecommendationUseCase) ensureMinimumCandidatePool(
	candidates, activeItems []*entities.WardrobeItem,
) []*entities.WardrobeItem {
	if len(candidates) >= 20 || len(candidates) >= len(activeItems) {
		return candidates
	}

	taken := map[uuid.UUID]bool{}
	for _, candidate := range candidates {
		taken[candidate.ID] = true
	}

	for _, item := range activeItems {
		if taken[item.ID] {
			continue
		}

		candidates = append(candidates, item)
		taken[item.ID] = true
		if len(candidates) >= 20 {
			break
		}
	}

	return candidates
}

func (uc *OutfitRecommendationUseCase) rankCandidates(
	candidates []*entities.WardrobeItem,
	intent dto.ParsedIntent,
) []CandidateForPrompt {
	scored := make([]RankedCandidate, len(candidates))
	for i, item := range candidates {
		score, tags := scoreCandidateItem(
			item,
			intent,
			uc.cfg.RAG.RecentlyWornPenaltyDays,
			uc.cfg.RAG.LongUnwornBonusDays,
		)
		scored[i] = RankedCandidate{
			Item:  item,
			Score: score,
			Tags:  tags,
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	limit := min(len(scored), 20)
	if limit < 15 && len(scored) >= 15 {
		limit = 15
	}

	final := make([]CandidateForPrompt, 0, limit)
	for i := 0; i < limit; i++ {
		final = append(final, CandidateForPrompt{
			Item: scored[i].Item,
			Tags: scored[i].Tags,
		})
	}

	return final
}

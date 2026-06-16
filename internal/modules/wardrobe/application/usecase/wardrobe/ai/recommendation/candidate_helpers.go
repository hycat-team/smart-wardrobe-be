package recommendation

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/ranking"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/retrieval"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/pkg/utils/sliceutils"
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
		return nil, nil, subscriptionerrors.ErrAiOutfitQuotaExceeded()
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
		return nil, nil, wardrobeerrors.ErrMinimumWardrobeItemsRequired()
	}

	return activeItems, quotaDTO, nil
}

func (uc *OutfitRecommendationUseCase) filterCandidates(
	ctx context.Context,
	userID uuid.UUID,
	activeItems []*entities.WardrobeItem,
	input dto.RecommendOutfitReq,
) ([]types.CandidateForPrompt, error) {
	intent := uc.parseRecommendationIntent(input)
	retrievalQuery := uc.buildRecommendationRetrievalQuery(ctx, intent)
	queryVector := uc.buildRecommendationQueryVector(ctx, retrievalQuery.SemanticQuery)

	candidates, err := uc.wardrobeRepo.GetHybridCandidates(
		ctx,
		userID,
		queryVector,
		retrieval.ExtractTermStrings(retrievalQuery.LexicalTerms),
		retrieval.ExtractTermStrings(retrievalQuery.ExcludedTerms),
		retrievalQuery.HardFilters,
		uc.cfg.RAG.RecommendationCandidateLimit,
		uc.cfg.RAG.RrfKParameter,
	)
	if err != nil {
		return nil, err
	}

	// Transform repositories.HybridCandidate to types.CandidateForRanking
	rankingCandidates := make([]types.CandidateForRanking, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Item == nil {
			continue
		}
		rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
			Item:            candidate.Item,
			Source:          candidate.RetrievalSource,
			VectorScore:     candidate.VectorScore,
			LexicalScore:    candidate.LexicalScore,
			RetrievalScore:  candidate.RetrievalScore,
			RetrievalRank:   candidate.RetrievalRank,
			RetrievalSource: candidate.RetrievalSource,
		})
	}

	rankingCandidates, _ = ranking.EnsureMinimumCandidatePool(
		rankingCandidates,
		activeItems,
		retrievalQuery,
		uc.cfg.RAG.RecommendationMinimumCandidatePool,
	)

	ranked, _ := ranking.RankCandidates(
		rankingCandidates,
		intent,
		retrievalQuery,
		uc.cfg.RAG.RecentlyWornPenaltyDays,
		uc.cfg.RAG.LongUnwornBonusDays,
		uc.cfg.RAG.RecommendationMinimumCandidatePool,
	)

	return ranked, nil
}

func (uc *OutfitRecommendationUseCase) parseRecommendationIntent(
	input dto.RecommendOutfitReq,
) dto.ParsedIntent {
	freeText := ""
	if input.Details != nil {
		freeText = *input.Details
	}

	intent := uc.nlpParser.Parse(freeText)
	hasExplicitOptions := false
	if input.Occasion != nil && *input.Occasion != "" {
		intent.Occasion = []string{strings.TrimSpace(*input.Occasion)}
		hasExplicitOptions = true
	}
	if input.StyleTarget != nil && *input.StyleTarget != "" {
		intent.StyleTarget = []string{strings.TrimSpace(*input.StyleTarget)}
		hasExplicitOptions = true
	}
	if input.ColorTone != nil && *input.ColorTone != "" {
		intent.ColorTone = []string{strings.TrimSpace(*input.ColorTone)}
		hasExplicitOptions = true
	}
	if input.Weather != nil && *input.Weather != "" {
		weather := strings.TrimSpace(*input.Weather)
		intent.PositiveConstraints = sliceutils.AppendUniqueStringCaseInsensitive(intent.PositiveConstraints, weather)
		hasExplicitOptions = true
	}
	if input.Season != nil && *input.Season != "" && *input.Season != dto.SeasonAll {
		intent.PositiveConstraints = sliceutils.AppendUniqueStringCaseInsensitive(intent.PositiveConstraints, string(*input.Season))
		hasExplicitOptions = true
	}
	intent.LexicalTerms = uc.nlpParser.RemoveConflictingLexicalTerms(intent)

	intent.SemanticQuery = retrieval.BuildRecommendationSemanticQuery(intent, freeText, hasExplicitOptions)

	return intent
}

func (uc *OutfitRecommendationUseCase) buildRecommendationQueryVector(
	ctx context.Context,
	semanticQuery string,
) entities.Vector {
	if semanticQuery == "" {
		uc.logger.Warn("Outfit recommendation embedding skipped",
			zap.String("reason", "empty_semantic_query"),
		)
		return nil
	}

	embedCtx, cancel := context.WithTimeout(ctx, time.Duration(uc.cfg.RAG.RecommendationEmbeddingTimeoutSeconds)*time.Second)
	defer cancel()

	vecs, err := uc.aiService.GenerateEmbeddings(embedCtx, []string{semanticQuery})
	if err != nil {
		uc.logger.Warn("Outfit recommendation embedding failed",
			zap.String("reason", "provider_error"),
			zap.Error(err),
		)
		return nil
	}
	if len(vecs) == 0 {
		uc.logger.Warn("Outfit recommendation embedding failed",
			zap.String("reason", "empty_vector_response"),
		)
		return nil
	}
	if len(vecs[0]) != uc.cfg.RAG.RecommendationEmbeddingDimension {
		uc.logger.Warn("Outfit recommendation embedding failed",
			zap.String("reason", "unexpected_dimension"),
			zap.Int("dimension", len(vecs[0])),
			zap.Int("expected_dimension", uc.cfg.RAG.RecommendationEmbeddingDimension),
		)
		return nil
	}

	return entities.Vector(vecs[0])
}

func (uc *OutfitRecommendationUseCase) buildRecommendationRetrievalQuery(
	ctx context.Context,
	intent dto.ParsedIntent,
) types.RecommendationRetrievalQuery {
	local := retrieval.LocalRecommendationQueryRewriter{}
	if uc.cfg == nil || !uc.cfg.RAG.RecommendationLLMRewriterEnabled {
		query, _ := local.Rewrite(ctx, intent)
		return query
	}

	rewriteCtx, cancel := context.WithTimeout(ctx, time.Duration(uc.cfg.RAG.RecommendationLLMRewriterTimeoutSeconds)*time.Second)
	defer cancel()

	query, err := retrieval.NewLLMRecommendationQueryRewriter(uc.aiService, uc.cfg).Rewrite(rewriteCtx, intent)
	if err == nil {
		if uc.logger != nil {
			uc.logger.Info("Outfit recommendation query rewriter completed",
				zap.String("rewriter_source", "llm"),
				zap.Int("lexical_terms_count", len(query.LexicalTerms)),
				zap.Int("excluded_terms_count", len(query.ExcludedTerms)),
			)
		}
		return query
	}

	if uc.logger != nil {
		uc.logger.Warn("Outfit recommendation query rewriter fallback",
			zap.String("rewriter_source", "local"),
			zap.String("fallback_reason", err.Error()),
		)
	}
	query, _ = local.Rewrite(ctx, intent)
	return query
}

func retrievalSourceCount(candidates []repositories.HybridCandidate, source string) int {
	count := 0
	for _, candidate := range candidates {
		if candidate.RetrievalSource == source {
			count++
		}
	}
	return count
}

func recommendationLexicalQueryMode(vector entities.Vector, lexicalTerms []string) string {
	hasVector := len(vector) > 0
	hasLexical := len(lexicalTerms) > 0
	switch {
	case hasVector && hasLexical:
		return "hybrid"
	case hasVector:
		return "vector"
	case hasLexical:
		return "lexical"
	default:
		return "fallback"
	}
}

func missingEmbeddingCount(items []*entities.WardrobeItem) int {
	count := 0
	for _, item := range items {
		if item == nil || len(item.Embedding) == 0 {
			count++
		}
	}
	return count
}

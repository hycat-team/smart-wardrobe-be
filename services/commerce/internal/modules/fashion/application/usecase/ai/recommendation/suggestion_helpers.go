package recommendation

import (
	"context"
	"fmt"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/retrieval"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	personalization_shared "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"
	"smart-wardrobe-be/shared/pkg/utils/stringutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// catalogFashionItemIDLimit caps how many system catalog ids are fetched
	// from the Wardrobe contract. The catalog is only a few thousand items; the
	// cap protects the request from an unexpectedly large table.
	catalogFashionItemIDLimit = 2000
	// catalogSuggestionCandidateLimit is the hybrid retrieval budget for
	// catalog suggestions. Only one item per gap role is returned, but a single
	// query covers every gap role, so the pool must be wide enough for each
	// category to survive the global relevance cut.
	catalogSuggestionCandidateLimit = 300
	// defaultSuggestionMaxTotal / defaultSuggestionMaxPerRole mirror the
	// shipped config values so a zero config still produces useful output.
	defaultSuggestionMaxTotal   = 3
	defaultSuggestionMaxPerRole = 1
	// suggestionPersonalStyleBonus / suggestionLearnedStyleBonus /
	// suggestionAvoidStylePenalty are the soft ordering weights applied when
	// picking a catalog suggestion (033 FR-021, SC-014). They never filter an
	// item out on their own; avoided items only lose to a better alternative.
	suggestionPersonalStyleBonus = 0.20
	suggestionLearnedStyleBonus  = 0.10
	suggestionAvoidStylePenalty  = 0.35
)

// catalogSuggestion is a catalog fashion item retrieved for one gap role.
type catalogSuggestion struct {
	FashionItem *dto.FashionItemContractDTO
	Role        string
	Relevance   float64
}

// buildCatalogSuggestions detects the outfit roles that are missing or only
// hold low-quality candidates, retrieves catalog items for them and returns at
// most one suggestion per role (max SuggestionMaxTotal) (033 US2/T040, FR-020
// to FR-026).
//
// It is deliberately fail-soft: any lookup failure returns no suggestions so
// the recommendation response is never blocked. It never changes quota, never
// touches the outfit items and never requires personalization entitlement.
func (uc *OutfitRecommendationUseCase) buildCatalogSuggestions(
	ctx context.Context,
	userID uuid.UUID,
	finalOutfit *dto.RecommendedOutfitRes,
	filter *candidateFilterResult,
) []dto.OutfitSuggestionRes {
	if uc == nil || uc.cfg == nil || finalOutfit == nil || filter == nil || len(filter.Candidates) == 0 {
		return nil
	}
	gapRoles := detectOutfitGapRoles(
		finalOutfit,
		filter.Pool,
		filter.FashionItemMap,
		uc.cfg.Personalization.GapScoreThreshold,
	)
	if len(gapRoles) == 0 {
		return nil
	}

	candidates := uc.retrieveCatalogCandidates(ctx, userID, gapRoles, filter)
	if len(candidates) == 0 {
		return nil
	}
	return uc.pickCatalogSuggestions(gapRoles, candidates, outfitPresentRoles(finalOutfit), filter.Personalization)
}

// BuildOutfitCompletionSuggestions implements
// usecase.IOutfitSuggestionBuilder for the outfit-completion endpoint (033
// US7/T077).
//
// It rebuilds the retrieval context from the completion request exactly as a
// recommendation does — same intent parser, same style-vector blend, same hard
// filters, same personalization entitlement and same catalog id cache — and then
// reuses [OutfitRecommendationUseCase.pickCatalogSuggestions] so the shipped
// US2 rules apply unchanged: at most one suggestion per role, at most three per
// response and never an item the user already owns.
//
// It never runs the LLM, never synthesizes an outfit and never touches quota.
func (uc *OutfitRecommendationUseCase) BuildOutfitCompletionSuggestions(
	ctx context.Context,
	userID uuid.UUID,
	input dto.RecommendOutfitReq,
	gapRoles []string,
) []dto.OutfitSuggestionRes {
	if uc == nil || uc.cfg == nil || len(gapRoles) == 0 {
		return nil
	}

	intent := uc.parseRecommendationIntent(input)
	personalization := uc.loadPersonalizationContext(ctx, userID, intent)
	intent = mergePersonalizationIntoIntent(intent, personalization)
	retrievalQuery := uc.buildRecommendationRetrievalQuery(ctx, userID, intent)
	// Same soft-preference rule as the outfit pipeline: category is never a hard
	// filter, otherwise a complete outfit could not be assembled.
	retrievalQuery.HardFilters.CategorySlugs = nil
	queryVector := uc.buildRecommendationQueryVector(
		ctx,
		retrievalQuery.SemanticQuery,
		personalization.StyleVector,
		personalization.StyleWeight,
	)

	candidates := uc.retrieveCatalogCandidates(ctx, userID, gapRoles, &candidateFilterResult{
		Personalization: personalization,
		RetrievalQuery:  retrievalQuery,
		QueryVector:     queryVector,
	})
	if len(candidates) == 0 {
		return nil
	}
	return uc.pickCatalogSuggestions(gapRoles, candidates, map[string]bool{}, personalization)
}

// detectOutfitGapRoles returns the structural roles that count as "not
// suitable" for this outfit: the role has no item at all, or every candidate
// for it scored below the configured quality threshold (033 FR-020).
//
// The thresholds are evaluated on the retrieval relevance already computed when
// building the candidate pool; fallback candidates carry 0 and therefore count
// as low quality. Fullbody replaces top+bottom, exactly like the outfit
// structure rules.
func detectOutfitGapRoles(
	finalOutfit *dto.RecommendedOutfitRes,
	pool []types.CandidateForRanking,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
	threshold float64,
) []string {
	present := outfitPresentRoles(finalOutfit)

	requiredRoles := []string{"top", "bottom", "footwear"}
	if present["fullbody"] {
		requiredRoles = []string{"fullbody", "footwear"}
	}

	bestRelevance := roleRelevanceScores(pool, fashionItemMap)
	gaps := make([]string, 0, len(requiredRoles))
	for _, role := range requiredRoles {
		if !present[role] {
			gaps = append(gaps, role)
			continue
		}
		if threshold > 0 && bestRelevance[role] < threshold {
			gaps = append(gaps, role)
		}
	}
	return gaps
}

// roleRelevanceScores returns the best retrieval relevance per wearing role.
// Vector cosine is the primary signal; lexical rank is used when the candidate
// matched on text only. Both are clamped to [0, 1] so the configured threshold
// (0.35 shipped) has a stable meaning.
func roleRelevanceScores(
	pool []types.CandidateForRanking,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) map[string]float64 {
	scores := map[string]float64{}
	for _, candidate := range pool {
		if candidate.Item == nil {
			continue
		}
		role := wearingRoleForFashionItem(fashionItemOf(candidate.Item, fashionItemMap))
		if role == "" {
			continue
		}
		relevance := candidate.VectorScore
		if candidate.LexicalScore > relevance {
			relevance = candidate.LexicalScore
		}
		if relevance > 1 {
			relevance = 1
		}
		if relevance > scores[role] {
			scores[role] = relevance
		}
	}
	return scores
}

// retrieveCatalogCandidates loads the cached catalog id list, drops items the
// user already owns, then retrieves the catalog items of the missing-role
// categories (033 US2/T039).
//
// The outfit query vector and lexical terms describe what the user asked for.
// Reusing them here silently dropped every gap-role item whose catalog wording
// or embedding differs from the request — a "casual" request found no "Sporty"
// footwear, so the gap stayed empty even though the category had items. The
// fallback retrieval therefore keeps the outfit hard filters (season, and the
// avoided terms) but passes no semantic vector and no lexical terms, which makes
// the repository return every catalog item inside the missing-role categories.
func (uc *OutfitRecommendationUseCase) retrieveCatalogCandidates(
	ctx context.Context,
	userID uuid.UUID,
	gapRoles []string,
	filter *candidateFilterResult,
) []catalogSuggestion {
	catalogIDs := uc.getCatalogFashionItemIDs(ctx)
	if len(catalogIDs) == 0 || uc.wardrobeContract == nil || uc.fashionItemRepo == nil {
		return nil
	}

	ownedIDs, err := uc.wardrobeContract.GetOwnedFashionItemIDs(ctx, userID)
	if err != nil {
		// Without a reliable ownership set we cannot guarantee FR-021 (never
		// suggest an owned item), so fail soft to "no suggestions".
		uc.warnSuggestion("owned_items_lookup_error", err)
		return nil
	}
	if len(ownedIDs) > 0 {
		owned := make(map[uuid.UUID]struct{}, len(ownedIDs))
		for _, id := range ownedIDs {
			owned[id] = struct{}{}
		}
		available := make([]uuid.UUID, 0, len(catalogIDs))
		for _, id := range catalogIDs {
			if _, exists := owned[id]; !exists {
				available = append(available, id)
			}
		}
		catalogIDs = available
	}
	if len(catalogIDs) == 0 {
		return nil
	}

	// Copy the hard filters: the category scope below belongs to this catalog
	// lookup only and must never leak back into the outfit retrieval state.
	hardFilters := filter.RetrievalQuery.HardFilters
	hardFilters.CategorySlugs = categorySlugsForRoles(gapRoles)

	limit := min(catalogSuggestionCandidateLimit, len(catalogIDs))
	hybridCandidates, err := uc.fashionItemRepo.GetHybridCandidates(
		ctx,
		sliceutils.UniqueUUIDs(catalogIDs),
		nil,
		nil,
		retrieval.ExtractTermStrings(filter.RetrievalQuery.ExcludedTerms),
		hardFilters,
		limit,
		uc.cfg.RAG.RrfKParameter,
	)
	if err != nil {
		uc.warnSuggestion("catalog_retrieval_error", err)
		return nil
	}
	if len(hybridCandidates) == 0 {
		return nil
	}

	fashionIDs := make([]uuid.UUID, 0, len(hybridCandidates))
	for _, candidate := range hybridCandidates {
		fashionIDs = append(fashionIDs, candidate.ID)
	}
	fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, sliceutils.UniqueUUIDs(fashionIDs))
	if err != nil {
		uc.warnSuggestion("catalog_metadata_error", err)
		return nil
	}
	metadata := make(map[uuid.UUID]*dto.FashionItemContractDTO, len(fashionItems))
	for _, item := range fashionItems {
		metadata[item.ID] = fashionItemEntityToContractDTO(item)
	}

	gapSet := make(map[string]bool, len(gapRoles))
	for _, role := range gapRoles {
		gapSet[role] = true
	}

	suggestions := make([]catalogSuggestion, 0, len(hybridCandidates))
	for _, candidate := range hybridCandidates {
		fi := metadata[candidate.ID]
		if fi == nil || fi.Category == nil {
			continue
		}
		role := mapper.CategorySlugToWearingRole(fi.Category.Slug)
		if !gapSet[role] {
			continue
		}
		relevance := candidate.VectorScore
		if candidate.LexicalScore > relevance {
			relevance = candidate.LexicalScore
		}
		if relevance > 1 {
			relevance = 1
		}
		suggestions = append(suggestions, catalogSuggestion{
			FashionItem: fi,
			Role:        role,
			Relevance:   relevance,
		})
	}
	return suggestions
}

// categorySlugsForRoles maps wearing roles to the catalog category slugs that
// can fill them (033 FR-020/FR-021). It mirrors
// [mapper.CategorySlugToWearingRole] and dedupes while preserving the role
// order, so the retrieval filter is stable. Roles without a catalog category are
// skipped, so an unknown role never widens the retrieval scope.
func categorySlugsForRoles(roles []string) []string {
	slugs := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	add := func(slug string) {
		if _, exists := seen[slug]; exists {
			return
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	for _, role := range roles {
		switch role {
		case "top":
			add("ao")
		case "bottom":
			add("quan")
			add("chan-vay")
		case "fullbody":
			add("dam")
		case "footwear":
			add("giay")
		case "outerwear":
			add("ao-khoac")
		case "headwear":
			add("mu")
		case "accessory":
			add("phu-kien")
		}
	}
	return slugs
}

// getCatalogFashionItemIDs reads the catalog id list through the fail-soft
// cache and falls back to the Wardrobe contract on a miss (033 T039, plan D5).
func (uc *OutfitRecommendationUseCase) getCatalogFashionItemIDs(ctx context.Context) []uuid.UUID {
	if uc.catalogCache != nil {
		if ids, err := uc.catalogCache.Get(ctx); err == nil && len(ids) > 0 {
			return ids
		}
	}
	if uc.wardrobeContract == nil {
		return nil
	}
	ids, err := uc.wardrobeContract.GetSystemCatalogFashionItemIDs(ctx, catalogFashionItemIDLimit)
	if err != nil {
		uc.warnSuggestion("catalog_contract_error", err)
		return nil
	}
	if len(ids) == 0 {
		return nil
	}
	if uc.catalogCache != nil {
		if err := uc.catalogCache.Set(ctx, ids); err != nil {
			uc.warnSuggestion("catalog_cache_set_error", err)
		}
	}
	return ids
}

// bestCatalogSuggestion picks the highest-scoring candidate for a role,
// preferring context match (already encoded in relevance) and the user's
// personal style, penalising avoided styles, and never reusing an item already
// suggested for another role (033 FR-021/FR-023, SC-014).
func bestCatalogSuggestion(
	candidates []catalogSuggestion,
	role string,
	used map[uuid.UUID]bool,
	pc types.PersonalizationContext,
) *catalogSuggestion {
	var best *catalogSuggestion
	bestScore := 0.0
	for i := range candidates {
		candidate := &candidates[i]
		if candidate.Role != role || candidate.FashionItem == nil || used[candidate.FashionItem.ID] {
			continue
		}
		score := candidate.Relevance + suggestionStyleScore(candidate.FashionItem, pc)
		if best == nil || score > bestScore ||
			(score == bestScore && candidate.FashionItem.ID.String() < best.FashionItem.ID.String()) {
			best = candidate
			bestScore = score
		}
	}
	return best
}

// suggestionStyleScore is the soft personal-style component of the suggestion
// ordering. It never filters an item out on its own.
func suggestionStyleScore(fi *dto.FashionItemContractDTO, pc types.PersonalizationContext) float64 {
	if !pc.HasStyleSignals() {
		return 0
	}
	score := 0.0
	if suggestionMatchesStyles(fi, pc.ExplicitStyles) {
		score += suggestionPersonalStyleBonus
	}
	if suggestionMatchesStyles(fi, pc.LearnedStyles) {
		score += suggestionLearnedStyleBonus
	}
	if suggestionMatchesStyles(fi, pc.ExcludedStyles) {
		score -= suggestionAvoidStylePenalty
	}
	return score
}

func suggestionMatchesStyles(fi *dto.FashionItemContractDTO, labels []string) bool {
	if fi == nil || fi.Style == nil || len(labels) == 0 {
		return false
	}
	itemStyle := normalizeStyleText(*fi.Style)
	if itemStyle == "" {
		return false
	}
	for _, label := range personalization_shared.NormalizeStyleLabels(labels) {
		terms := append([]string{label}, retrieval.ExpandTaxonomyTermValues("style", []string{label})...)
		for _, term := range terms {
			normalized := normalizeStyleText(term)
			if normalized == "" {
				continue
			}
			if strings.Contains(itemStyle, normalized) || strings.Contains(normalized, itemStyle) {
				return true
			}
		}
	}
	return false
}

func normalizeStyleText(value string) string {
	return strings.ToLower(stringutils.RemoveVietnameseSigns(strings.TrimSpace(value)))
}

// suggestionReason builds the Vietnamese reason shown next to a suggestion
// (033 FR-023). No LLM is involved: it is a template over the role and the
// gap kind.
func suggestionReason(role string, roleAlreadyPresent bool, pc types.PersonalizationContext) string {
	label := suggestionRoleLabel(role)
	var reason string
	if roleAlreadyPresent {
		reason = fmt.Sprintf("Tủ đồ chưa có %s thật sự hợp với ngữ cảnh của bộ này", label)
	} else {
		reason = fmt.Sprintf("Tủ đồ chưa có %s phù hợp để hoàn thiện bộ này", label)
	}
	if pc.HasStyleSignals() {
		reason += ", gợi ý này được chọn theo gu bạn đã lưu"
	}
	return reason + "."
}

// suggestionRoleLabel maps a wearing role to the Vietnamese noun used in the
// suggestion reason.
func suggestionRoleLabel(role string) string {
	switch role {
	case "top":
		return "áo"
	case "bottom":
		return "quần hoặc chân váy"
	case "fullbody":
		return "đầm"
	case "outerwear":
		return "áo khoác"
	case "footwear":
		return "giày"
	case "headwear":
		return "mũ"
	case "accessory":
		return "phụ kiện"
	default:
		return "món đồ"
	}
}

func outfitPresentRoles(finalOutfit *dto.RecommendedOutfitRes) map[string]bool {
	present := map[string]bool{}
	if finalOutfit == nil {
		return present
	}
	for _, group := range finalOutfit.Items {
		if group == nil || group.Primary == nil {
			continue
		}
		if group.Role != "" {
			present[group.Role] = true
		}
	}
	return present
}

// wearingRoleForFashionItem resolves the wearing role from fashion metadata,
// returning "" when the item has no category.
func wearingRoleForFashionItem(fi *dto.FashionItemContractDTO) string {
	if fi == nil || fi.Category == nil {
		return ""
	}
	return mapper.CategorySlugToWearingRole(fi.Category.Slug)
}

func fashionItemOf(
	item *wardrobeDto.WardrobeItemContractDTO,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) *dto.FashionItemContractDTO {
	if item == nil || item.FashionItemID == nil {
		return nil
	}
	return fashionItemMap[*item.FashionItemID]
}

// warnSuggestion logs a fail-soft suggestion failure reason without leaking
// user style text.
func (uc *OutfitRecommendationUseCase) warnSuggestion(reason string, err error) {
	if uc == nil || uc.logger == nil {
		return
	}
	uc.logger.Warn("Outfit catalog suggestion skipped",
		zap.String("reason", reason),
		zap.Error(err),
	)
}

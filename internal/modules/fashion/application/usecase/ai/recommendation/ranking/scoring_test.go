package ranking

import (
	"testing"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

func TestEnsureMinimumCandidatePoolTracksSource(t *testing.T) {
	activeItems := []*wardrobeDto.WardrobeItemRes{
		newTestItem(),
		newTestItem(),
		newTestItem(),
	}

	retrievalCandidates := []types.CandidateForRanking{
		{
			Item:            activeItems[0],
			RetrievalScore:  0.8,
			RetrievalRank:   1,
			RetrievalSource: "lexical",
			Source:          "lexical",
		},
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(retrievalCandidates, activeItems, types.RecommendationRetrievalQuery{}, 3, nil)
	if fallbackCounts.Total() != 2 {
		t.Fatalf("expected two fallback candidates, got %d", fallbackCounts.Total())
	}
	if rankingCandidates[0].Source != "lexical" ||
		rankingCandidates[1].Source != types.CandidateSourceStrictFallback ||
		rankingCandidates[2].Source != types.CandidateSourceStrictFallback {
		t.Fatalf("unexpected candidate sources: %+v", rankingCandidates)
	}
	if rankingCandidates[0].RetrievalScore != 0.8 || rankingCandidates[0].RetrievalRank != 1 {
		t.Fatalf("expected retrieval metadata to be preserved, got %+v", rankingCandidates[0])
	}
	if rankingCandidates[1].RetrievalRank != 2 || rankingCandidates[1].RetrievalScore != 0 {
		t.Fatalf("expected fallback metadata after retrieval candidates, got %+v", rankingCandidates[1])
	}
}

func TestRankCandidatesAddsSourceTags(t *testing.T) {
	itemID := uuid.New()
	candidates := []types.CandidateForRanking{
		{
			Item:   &wardrobeDto.WardrobeItemRes{ID: itemID},
			Source: types.CandidateSourceStrictFallback,
		},
	}

	ranked, _ := RankCandidates(candidates, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, 2, nil)
	if len(ranked) != 1 {
		t.Fatalf("expected one ranked candidate, got %d", len(ranked))
	}
	if !containsString(ranked[0].Tags, "candidate-source:strict-fallback") {
		t.Fatalf("expected fallback source tag, got %v", ranked[0].Tags)
	}
}

func TestEnsureMinimumCandidatePoolDoesNotFallbackHardFilteredItems(t *testing.T) {
	winter := "winter"
	summer := "summer"
	item1, f1 := newTestItemWithFashion()
	f1.Seasonality = &summer
	item2, f2 := newTestItemWithFashion()
	f2.Seasonality = &winter
	activeItems := []*wardrobeDto.WardrobeItemRes{item1, item2}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
		item1.FashionItem.ID: f1,
		item2.FashionItem.ID: f2,
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(
		nil,
		activeItems,
		types.RecommendationRetrievalQuery{
			HardFilters: repositories.RecommendationHardFilters{Seasonality: []string{"winter"}},
		},
		2,
		fashionItemMap,
	)

	if fallbackCounts.Total() != 1 {
		t.Fatalf("expected one fallback candidate, got %d", fallbackCounts.Total())
	}
	if len(rankingCandidates) != 1 || rankingCandidates[0].Item.ID != activeItems[1].ID {
		t.Fatalf("expected only winter item fallback, got %+v", rankingCandidates)
	}
}

func TestFallbackSeasonalityFilterAllowsAllSeasonAndMissingMetadata(t *testing.T) {
	for _, seasonality := range []string{"", "Bốn mùa", "Quanh năm", "all-season"} {
		if !MatchesFallbackSeasonalityFilter(seasonality, []string{"winter"}) {
			t.Fatalf("expected seasonality %q to pass winter hard filter", seasonality)
		}
	}
	if !MatchesFallbackSeasonalityFilter("Thích hợp mùa đông", []string{"winter"}) {
		t.Fatal("expected Vietnamese winter metadata to pass winter hard filter")
	}
	if MatchesFallbackSeasonalityFilter("Mùa hè", []string{"winter"}) {
		t.Fatal("expected summer-only metadata to fail winter hard filter")
	}
}

func TestEnsureMinimumCandidatePoolDoesNotFallbackExcludedTerms(t *testing.T) {
	black := "black"
	white := "white"
	item1, f1 := newTestItemWithFashion()
	f1.Color = &black
	item2, f2 := newTestItemWithFashion()
	f2.Color = &white
	activeItems := []*wardrobeDto.WardrobeItemRes{item1, item2}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
		item1.FashionItem.ID: f1,
		item2.FashionItem.ID: f2,
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(
		nil,
		activeItems,
		types.RecommendationRetrievalQuery{
			ExcludedTerms: []types.RetrievalTerm{{Value: "black", Source: types.RetrievalTermSourceRaw}},
		},
		2,
		fashionItemMap,
	)

	if fallbackCounts.Total() != 1 {
		t.Fatalf("expected one fallback candidate, got %d", fallbackCounts.Total())
	}
	if len(rankingCandidates) != 1 || rankingCandidates[0].Item.ID != activeItems[1].ID {
		t.Fatalf("expected only non-excluded item fallback, got %+v", rankingCandidates)
	}
}

func TestEnsureMinimumCandidatePoolUsesRelaxedFallbackWhenStrictLexicalDoesNotMatch(t *testing.T) {
	white := "white"
	item1, f1 := newTestItemWithFashion()
	f1.Color = &white
	activeItems := []*wardrobeDto.WardrobeItemRes{item1}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
		item1.FashionItem.ID: f1,
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(
		nil,
		activeItems,
		types.RecommendationRetrievalQuery{
			LexicalTerms: []types.RetrievalTerm{{Value: "cashmere", Source: types.RetrievalTermSourceRaw}},
		},
		1,
		fashionItemMap,
	)

	if fallbackCounts.Strict != 0 || fallbackCounts.Relaxed != 1 {
		t.Fatalf("expected relaxed fallback only, got %+v", fallbackCounts)
	}
	if rankingCandidates[0].Source != types.CandidateSourceRelaxedFallback {
		t.Fatalf("expected relaxed fallback source, got %q", rankingCandidates[0].Source)
	}
}

func TestFallbackSourcePenaltyIsFixedBySource(t *testing.T) {
	if FallbackSourcePenalty(types.CandidateSourceStrictFallback) <= 0 {
		t.Fatal("expected strict fallback penalty")
	}
	if FallbackSourcePenalty(types.CandidateSourceRelaxedFallback) <= FallbackSourcePenalty(types.CandidateSourceStrictFallback) {
		t.Fatal("expected relaxed fallback penalty to exceed strict fallback")
	}
	if FallbackSourcePenalty(types.CandidateSourceGeneralFallback) <= FallbackSourcePenalty(types.CandidateSourceRelaxedFallback) {
		t.Fatal("expected general fallback penalty to exceed relaxed fallback")
	}
	if FallbackSourcePenalty("lexical") != 0 {
		t.Fatal("expected retrieval sources to have no fallback penalty")
	}
}

func TestRankCandidatesUsesRetrievalRelevanceWhenRuleScoresAreClose(t *testing.T) {
	lowRetrievalID := uuid.New()
	highRetrievalID := uuid.New()
	candidates := []types.CandidateForRanking{
		{
			Item:            &wardrobeDto.WardrobeItemRes{ID: lowRetrievalID},
			Source:          "lexical",
			RetrievalScore:  0.1,
			RetrievalRank:   2,
			RetrievalSource: "lexical",
		},
		{
			Item:            &wardrobeDto.WardrobeItemRes{ID: highRetrievalID},
			Source:          "lexical",
			RetrievalScore:  0.9,
			RetrievalRank:   1,
			RetrievalSource: "lexical",
		},
	}

	ranked, stats := RankCandidates(candidates, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, 2, nil)
	if len(ranked) != 2 {
		t.Fatalf("expected two ranked candidates, got %d", len(ranked))
	}
	if stats.MaxScore <= stats.MinScore || stats.AvgScore == 0 {
		t.Fatalf("expected populated rerank stats, got %+v", stats)
	}
	if ranked[0].Item.ID != highRetrievalID {
		t.Fatalf("expected higher retrieval relevance candidate first, got %s", ranked[0].Item.ID)
	}
}

func TestScoreCandidateItemDoesNotPenalizeOuterwearWhenWeatherUnspecified(t *testing.T) {
	item, fi := newTestItemWithFashion()
	catName := "ao-khoac"
	fi.CategoryName = &catName
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if score != 1.1 {
		t.Fatalf("expected neutral outerwear baseline without weather penalty, got %.2f tags=%v", score, tags)
	}
	if containsString(tags, "weather-mismatch:heavy-outerwear") {
		t.Fatalf("did not expect weather mismatch tag without weather, got %v", tags)
	}
}

func TestScoreCandidateItemBoostsColdLikeOuterwear(t *testing.T) {
	item, fi := newTestItemWithFashion()
	catName := "ao-khoac"
	fi.CategoryName = &catName
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"winter"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if score <= 1.1 {
		t.Fatalf("expected cold-like outerwear boost, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "weather-appropriate:outerwear") {
		t.Fatalf("expected cold-like outerwear tag, got %v", tags)
	}
}

func TestScoreCandidateItemPenalizesHotLikeOuterwear(t *testing.T) {
	item, fi := newTestItemWithFashion()
	catName := "ao-khoac"
	fi.CategoryName = &catName
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"summer"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if score >= 1.1 {
		t.Fatalf("expected hot-like outerwear penalty, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "weather-mismatch:heavy-outerwear") {
		t.Fatalf("expected hot-like outerwear tag, got %v", tags)
	}
}

func TestScoreCandidateItemDoesNotTreatRainyAsCold(t *testing.T) {
	item, fi := newTestItemWithFashion()
	catName := "ao-khoac"
	fi.CategoryName = &catName
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	_, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"rainy"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if containsString(tags, "weather-appropriate:outerwear") {
		t.Fatalf("did not expect rainy to use cold-like outerwear boost, got %v", tags)
	}
	if containsString(tags, "weather-appropriate:rainy") {
		t.Fatalf("did not expect generic outerwear category to get rainy boost, got %v", tags)
	}
}

func TestScoreCandidateItemBoostsExplicitRainySignal(t *testing.T) {
	item, fi := newTestItemWithFashion()
	desc := "waterproof rain jacket"
	fi.Description = &desc
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	_, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"rainy"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if !containsString(tags, "weather-appropriate:rainy") {
		t.Fatalf("expected explicit rainy signal tag, got %v", tags)
	}
}

func TestScoreCandidateItemAppliesExcludedWeatherPenalty(t *testing.T) {
	item, fi := newTestItemWithFashion()
	catName := "ao-khoac"
	fi.CategoryName = &catName
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{ExcludedWeather: []string{"cold"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if score >= 1.1 {
		t.Fatalf("expected excluded cold-like weather penalty, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "avoid-weather:cold-like") {
		t.Fatalf("expected excluded weather tag, got %v", tags)
	}
}

func TestScoreCandidateItemMatchesOccasionTaxonomy(t *testing.T) {
	item, fi := newTestItemWithFashion()
	style := "office"
	fi.Style = &style
	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{item.FashionItem.ID: fi}

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{Occasion: []string{"work"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if score <= 1.1 {
		t.Fatalf("expected work occasion to match office taxonomy, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "occasion-match:taxonomy:work") {
		t.Fatalf("expected occasion taxonomy tag, got %v", tags)
	}
}

func TestRankedCandidateLessUsesRetrievalRankAndIDTieBreakers(t *testing.T) {
	idA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	idB := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	itemA := &wardrobeDto.WardrobeItemRes{ID: idA}
	itemB := &wardrobeDto.WardrobeItemRes{ID: idB}

	if !RankedCandidateLess(
		types.RankedCandidate{Item: itemB, Score: 2, RetrievalRank: 1},
		types.RankedCandidate{Item: itemA, Score: 2, RetrievalRank: 2},
	) {
		t.Fatal("expected lower retrieval rank to win when score ties")
	}
	if !RankedCandidateLess(
		types.RankedCandidate{Item: itemA, Score: 2, RetrievalRank: 0},
		types.RankedCandidate{Item: itemB, Score: 2, RetrievalRank: 0},
	) {
		t.Fatal("expected item ID to break ties when score and rank tie")
	}
}

func TestDiversifyRankedCandidates(t *testing.T) {
	item1, f1 := newTestItemWithFashion()
	cn1 := "ao"
	f1.CategoryName = &cn1
	item2, f2 := newTestItemWithFashion()
	cn2 := "ao"
	f2.CategoryName = &cn2
	item3, f3 := newTestItemWithFashion()
	cn3 := "ao"
	f3.CategoryName = &cn3
	item4, f4 := newTestItemWithFashion()
	cn4 := "ao"
	f4.CategoryName = &cn4
	item5, f5 := newTestItemWithFashion()
	cn5 := "quan"
	f5.CategoryName = &cn5

	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
		item1.FashionItem.ID: f1,
		item2.FashionItem.ID: f2,
		item3.FashionItem.ID: f3,
		item4.FashionItem.ID: f4,
		item5.FashionItem.ID: f5,
	}

	items := []types.RankedCandidate{
		{Item: item1, Score: 3.0},
		{Item: item2, Score: 2.8},
		{Item: item3, Score: 2.6},
		{Item: item4, Score: 2.4},
		{Item: item5, Score: 2.2},
	}

	diversified := DiversifyRankedCandidates(items, 4, fashionItemMap)
	if len(diversified) != 4 {
		t.Fatalf("expected 4 diversified candidates, got %d", len(diversified))
	}

	categoryCounts := map[string]int{}
	for _, cand := range diversified {
		cat := "?"
		if cand.Item.FashionItem != nil {
			if fi := fashionItemMap[cand.Item.FashionItem.ID]; fi != nil && fi.CategoryName != nil {
				cat = *fi.CategoryName
			}
		}
		categoryCounts[cat]++
	}
	if categoryCounts["ao"] != 3 || categoryCounts["quan"] != 1 {
		t.Fatalf("expected 3 ao and 1 quan, got %+v", categoryCounts)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func newTestItem() *wardrobeDto.WardrobeItemRes {
	return &wardrobeDto.WardrobeItemRes{
		ID: uuid.New(),
	}
}

func newTestItemWithFashion() (*wardrobeDto.WardrobeItemRes, *dto.FashionItemRes) {
	fashionID := uuid.New()
	item := &wardrobeDto.WardrobeItemRes{
		ID: uuid.New(),
		FashionItem: &wardrobeDto.FashionItemRes{
			ID: fashionID,
		},
	}
	res := &dto.FashionItemRes{ID: fashionID}
	return item, res
}

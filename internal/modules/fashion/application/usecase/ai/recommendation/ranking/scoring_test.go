package ranking

import (
	"testing"

	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func TestEnsureMinimumCandidatePoolTracksSource(t *testing.T) {
	activeItems := []*entities.WardrobeItem{
		{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}},
		{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}},
		{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}},
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

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(retrievalCandidates, activeItems, types.RecommendationRetrievalQuery{}, 3)
	if fallbackCounts.Total() != 2 {
		t.Fatalf("expected two fallback candidates, got %d", fallbackCounts.Total())
	}
	if rankingCandidates[0].Source != "lexical" ||
		rankingCandidates[1].Source != candidateSourceStrictFallback ||
		rankingCandidates[2].Source != candidateSourceStrictFallback {
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
			Item:   &entities.WardrobeItem{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: itemID}}},
			Source: candidateSourceStrictFallback,
		},
	}

	ranked, _ := RankCandidates(candidates, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, 2)
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
	activeItems := []*entities.WardrobeItem{
		wardrobeItemWithFashion(&entities.FashionItem{Seasonality: &summer}),
		wardrobeItemWithFashion(&entities.FashionItem{Seasonality: &winter}),
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(
		nil,
		activeItems,
		types.RecommendationRetrievalQuery{
			HardFilters: repositories.RecommendationHardFilters{Seasonality: []string{"winter"}},
		},
		2,
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
	activeItems := []*entities.WardrobeItem{
		wardrobeItemWithFashion(&entities.FashionItem{Color: &black}),
		wardrobeItemWithFashion(&entities.FashionItem{Color: &white}),
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(
		nil,
		activeItems,
		types.RecommendationRetrievalQuery{
			ExcludedTerms: []types.RetrievalTerm{{Value: "black", Source: types.RetrievalTermSourceRaw}},
		},
		2,
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
	activeItems := []*entities.WardrobeItem{
		wardrobeItemWithFashion(&entities.FashionItem{Color: &white}),
	}

	rankingCandidates, fallbackCounts := EnsureMinimumCandidatePool(
		nil,
		activeItems,
		types.RecommendationRetrievalQuery{
			LexicalTerms: []types.RetrievalTerm{{Value: "cashmere", Source: types.RetrievalTermSourceRaw}},
		},
		1,
	)

	if fallbackCounts.Strict != 0 || fallbackCounts.Relaxed != 1 {
		t.Fatalf("expected relaxed fallback only, got %+v", fallbackCounts)
	}
	if rankingCandidates[0].Source != candidateSourceRelaxedFallback {
		t.Fatalf("expected relaxed fallback source, got %q", rankingCandidates[0].Source)
	}
}

func TestFallbackSourcePenaltyIsFixedBySource(t *testing.T) {
	if FallbackSourcePenalty(candidateSourceStrictFallback) <= 0 {
		t.Fatal("expected strict fallback penalty")
	}
	if FallbackSourcePenalty(candidateSourceRelaxedFallback) <= FallbackSourcePenalty(candidateSourceStrictFallback) {
		t.Fatal("expected relaxed fallback penalty to exceed strict fallback")
	}
	if FallbackSourcePenalty(candidateSourceGeneralFallback) <= FallbackSourcePenalty(candidateSourceRelaxedFallback) {
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
			Item:            &entities.WardrobeItem{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: lowRetrievalID}}},
			Source:          "lexical",
			RetrievalScore:  0.1,
			RetrievalRank:   2,
			RetrievalSource: "lexical",
		},
		{
			Item:            &entities.WardrobeItem{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: highRetrievalID}}},
			Source:          "lexical",
			RetrievalScore:  0.9,
			RetrievalRank:   1,
			RetrievalSource: "lexical",
		},
	}

	ranked, stats := RankCandidates(candidates, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, 2)
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
	item := wardrobeItemWithCategory("ao-khoac", "Ao khoac")

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14)

	if score != 1.1 {
		t.Fatalf("expected neutral outerwear baseline without weather penalty, got %.2f tags=%v", score, tags)
	}
	if containsString(tags, "weather-mismatch:heavy-outerwear") {
		t.Fatalf("did not expect weather mismatch tag without weather, got %v", tags)
	}
}

func TestScoreCandidateItemBoostsColdLikeOuterwear(t *testing.T) {
	item := wardrobeItemWithCategory("ao-khoac", "Ao khoac")

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"winter"}}, types.RecommendationRetrievalQuery{}, 3, 14)

	if score <= 1.1 {
		t.Fatalf("expected cold-like outerwear boost, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "weather-appropriate:outerwear") {
		t.Fatalf("expected cold-like outerwear tag, got %v", tags)
	}
}

func TestScoreCandidateItemPenalizesHotLikeOuterwear(t *testing.T) {
	item := wardrobeItemWithCategory("ao-khoac", "Ao khoac")

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"summer"}}, types.RecommendationRetrievalQuery{}, 3, 14)

	if score >= 1.1 {
		t.Fatalf("expected hot-like outerwear penalty, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "weather-mismatch:heavy-outerwear") {
		t.Fatalf("expected hot-like outerwear tag, got %v", tags)
	}
}

func TestScoreCandidateItemDoesNotTreatRainyAsCold(t *testing.T) {
	item := wardrobeItemWithCategory("ao-khoac", "Ao khoac")

	_, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"rainy"}}, types.RecommendationRetrievalQuery{}, 3, 14)

	if containsString(tags, "weather-appropriate:outerwear") {
		t.Fatalf("did not expect rainy to use cold-like outerwear boost, got %v", tags)
	}
	if containsString(tags, "weather-appropriate:rainy") {
		t.Fatalf("did not expect generic outerwear category to get rainy boost, got %v", tags)
	}
}

func TestScoreCandidateItemBoostsExplicitRainySignal(t *testing.T) {
	description := "waterproof rain jacket"
	item := wardrobeItemWithFashion(&entities.FashionItem{Description: &description})

	_, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"rainy"}}, types.RecommendationRetrievalQuery{}, 3, 14)

	if !containsString(tags, "weather-appropriate:rainy") {
		t.Fatalf("expected explicit rainy signal tag, got %v", tags)
	}
}

func TestScoreCandidateItemAppliesExcludedWeatherPenalty(t *testing.T) {
	item := wardrobeItemWithCategory("ao-khoac", "Ao khoac")

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{ExcludedWeather: []string{"cold"}}, types.RecommendationRetrievalQuery{}, 3, 14)

	if score >= 1.1 {
		t.Fatalf("expected excluded cold-like weather penalty, got %.2f tags=%v", score, tags)
	}
	if !containsString(tags, "avoid-weather:cold-like") {
		t.Fatalf("expected excluded weather tag, got %v", tags)
	}
}

func TestScoreCandidateItemMatchesOccasionTaxonomy(t *testing.T) {
	style := "office"
	item := wardrobeItemWithFashion(&entities.FashionItem{Style: &style})

	score, tags := ScoreCandidateItem(item, dto.ParsedIntent{Occasion: []string{"work"}}, types.RecommendationRetrievalQuery{}, 3, 14)

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
	itemA := &entities.WardrobeItem{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: idA}}}
	itemB := &entities.WardrobeItem{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: idB}}}

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
	items := []types.RankedCandidate{
		{Item: wardrobeItemWithCategory("ao", "ao 1"), Score: 3.0},
		{Item: wardrobeItemWithCategory("ao", "ao 2"), Score: 2.8},
		{Item: wardrobeItemWithCategory("ao", "ao 3"), Score: 2.6},
		{Item: wardrobeItemWithCategory("ao", "ao 4"), Score: 2.4},
		{Item: wardrobeItemWithCategory("quan", "quan 1"), Score: 2.2},
	}

	diversified := DiversifyRankedCandidates(items, 4)
	if len(diversified) != 4 {
		t.Fatalf("expected 4 diversified candidates, got %d", len(diversified))
	}

	// Verify that the fourth item in diversified is "quan 1" instead of "ao 4" because of the cap of 3
	categoryCounts := map[string]int{}
	for _, cand := range diversified {
		categoryCounts[cand.Item.FashionCategory().Slug]++
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

func wardrobeItemWithCategory(slug, name string) *entities.WardrobeItem {
	return wardrobeItemWithFashion(&entities.FashionItem{Category: &entities.Category{Slug: slug, Name: name}})
}

func wardrobeItemWithFashion(fashion *entities.FashionItem) *entities.WardrobeItem {
	return &entities.WardrobeItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		FashionItem:     fashion,
	}
}

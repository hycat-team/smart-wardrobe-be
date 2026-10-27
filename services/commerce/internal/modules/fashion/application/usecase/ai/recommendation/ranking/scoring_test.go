package ranking

import (
	"math"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

func TestEnsureMinimumCandidatePoolTracksSource(t *testing.T) {
	activeItems := []*wardrobeDto.WardrobeItemContractDTO{
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

func TestEnsureMinimumCandidatePoolGuaranteesStructureWhenPoolAlreadyFull(t *testing.T) {
	top1, fTop1 := newTestItemWithFashion()
	fTop1.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Áo", Slug: "ao"}
	top2, fTop2 := newTestItemWithFashion()
	fTop2.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Áo", Slug: "ao"}
	bottom, fBottom := newTestItemWithFashion()
	fBottom.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Quần", Slug: "quan"}

	activeItems := []*wardrobeDto.WardrobeItemContractDTO{top1, top2, bottom}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top1.FashionItemID:   fTop1,
		*top2.FashionItemID:   fTop2,
		*bottom.FashionItemID: fBottom,
	}

	// Pool đã đầy (2 = minimumPool) nhưng chỉ toàn "ao" và thiếu bottom.
	candidates := []types.CandidateForRanking{
		{Item: top1, Source: "lexical"},
		{Item: top2, Source: "lexical"},
	}

	pool, _ := EnsureMinimumCandidatePool(candidates, activeItems, types.RecommendationRetrievalQuery{}, 2, fashionItemMap)
	if !HasValidOutfitStructure(pool, fashionItemMap) {
		t.Fatalf("expected structure guarantee to add a bottom, got %d candidates", len(pool))
	}
	found := false
	for _, c := range pool {
		if c.Item != nil && c.Item.ID == bottom.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected bottom item to be added by structure guarantee")
	}
}

func TestEnsureOutfitStructureAddsOnlyMissingRoles(t *testing.T) {
	top1, fTop1 := newTestItemWithFashion()
	fTop1.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Áo", Slug: "ao"}
	top2, fTop2 := newTestItemWithFashion()
	fTop2.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Áo", Slug: "ao"}
	bottom, fBottom := newTestItemWithFashion()
	fBottom.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Quần", Slug: "quan"}

	activeItems := []*wardrobeDto.WardrobeItemContractDTO{top1, top2, bottom}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top1.FashionItemID:   fTop1,
		*top2.FashionItemID:   fTop2,
		*bottom.FashionItemID: fBottom,
	}

	candidates := []types.CandidateForRanking{{Item: top1, Source: "lexical"}}
	pool, _ := EnsureMinimumCandidatePool(candidates, activeItems, types.RecommendationRetrievalQuery{}, 2, fashionItemMap)

	if len(pool) != 3 {
		t.Fatalf("expected exactly 3 candidates (2 tops + 1 bottom), got %d", len(pool))
	}
}

func TestRankCandidatesKeepsOutfitStructureAfterDiversify(t *testing.T) {
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{}
	candidates := make([]types.CandidateForRanking, 0, 5)
	for i := 0; i < 4; i++ {
		item, fi := newTestItemWithFashion()
		fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Áo", Slug: "ao"}
		fashionItemMap[*item.FashionItemID] = fi
		candidates = append(candidates, types.CandidateForRanking{
			Item: item, Source: "lexical", RetrievalScore: 0.9, RetrievalRank: i + 1,
		})
	}
	bottom, fBottom := newTestItemWithFashion()
	fBottom.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Quần", Slug: "quan"}
	fashionItemMap[*bottom.FashionItemID] = fBottom
	candidates = append(candidates, types.CandidateForRanking{
		Item: bottom, Source: "lexical", RetrievalScore: 0.1, RetrievalRank: 5,
	})

	ranked, _ := RankCandidates(candidates, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, 3, fashionItemMap)

	promptCandidates := make([]types.CandidateForPrompt, 0, len(ranked))
	for _, c := range ranked {
		promptCandidates = append(promptCandidates, types.CandidateForPrompt{Item: c.Item})
	}
	if !HasValidOutfitStructurePrompts(promptCandidates, fashionItemMap) {
		t.Fatalf("expected ranked candidates to keep a valid outfit structure, got %d", len(ranked))
	}
}

func TestRankCandidatesAddsSourceTags(t *testing.T) {
	itemID := uuid.New()
	candidates := []types.CandidateForRanking{
		{
			Item:   &wardrobeDto.WardrobeItemContractDTO{ID: itemID},
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
	activeItems := []*wardrobeDto.WardrobeItemContractDTO{item1, item2}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*item1.FashionItemID: f1,
		*item2.FashionItemID: f2,
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
	activeItems := []*wardrobeDto.WardrobeItemContractDTO{item1, item2}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*item1.FashionItemID: f1,
		*item2.FashionItemID: f2,
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
	activeItems := []*wardrobeDto.WardrobeItemContractDTO{item1}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*item1.FashionItemID: f1,
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
			Item:            &wardrobeDto.WardrobeItemContractDTO{ID: lowRetrievalID},
			Source:          "lexical",
			RetrievalScore:  0.1,
			RetrievalRank:   2,
			RetrievalSource: "lexical",
		},
		{
			Item:            &wardrobeDto.WardrobeItemContractDTO{ID: highRetrievalID},
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
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao-khoac", Slug: "ao-khoac"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

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
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao-khoac", Slug: "ao-khoac"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

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
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao-khoac", Slug: "ao-khoac"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

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
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao-khoac", Slug: "ao-khoac"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

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
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

	_, tags := ScoreCandidateItem(item, dto.ParsedIntent{PositiveConstraints: []string{"rainy"}}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if !containsString(tags, "weather-appropriate:rainy") {
		t.Fatalf("expected explicit rainy signal tag, got %v", tags)
	}
}

func TestScoreCandidateItemAppliesExcludedWeatherPenalty(t *testing.T) {
	item, fi := newTestItemWithFashion()
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao-khoac", Slug: "ao-khoac"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

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
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

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
	itemA := &wardrobeDto.WardrobeItemContractDTO{ID: idA}
	itemB := &wardrobeDto.WardrobeItemContractDTO{ID: idB}

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
	f1.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao", Slug: "ao"}
	item2, f2 := newTestItemWithFashion()
	f2.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao", Slug: "ao"}
	item3, f3 := newTestItemWithFashion()
	f3.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao", Slug: "ao"}
	item4, f4 := newTestItemWithFashion()
	f4.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao", Slug: "ao"}
	item5, f5 := newTestItemWithFashion()
	f5.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "quan", Slug: "quan"}

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*item1.FashionItemID: f1,
		*item2.FashionItemID: f2,
		*item3.FashionItemID: f3,
		*item4.FashionItemID: f4,
		*item5.FashionItemID: f5,
	}

	items := []types.RankedCandidate{
		{Item: item1, Score: 2.0},
		{Item: item2, Score: 1.9},
		{Item: item3, Score: 1.8},
		{Item: item4, Score: 1.7},
		{Item: item5, Score: 1.6},
	}

	diversified := DiversifyRankedCandidates(items, 4, fashionItemMap)
	if len(diversified) != 4 {
		t.Fatalf("expected 4 candidates, got %d", len(diversified))
	}

	categoryCounts := map[string]int{}
	for _, cand := range diversified {
		cat := "?"
		if cand.Item.FashionItemID != nil {
			if fi := fashionItemMap[*cand.Item.FashionItemID]; fi != nil && fi.Category != nil {
				cat = fi.Category.Name
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

func newTestItem() *wardrobeDto.WardrobeItemContractDTO {
	return &wardrobeDto.WardrobeItemContractDTO{
		ID: uuid.New(),
	}
}

func newTestItemWithFashion() (*wardrobeDto.WardrobeItemContractDTO, *dto.FashionItemContractDTO) {
	fashionID := uuid.New()
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionID,
	}
	res := &dto.FashionItemContractDTO{ID: fashionID}
	return item, res
}

func personalisedTestItem(style string) (*wardrobeDto.WardrobeItemContractDTO, map[uuid.UUID]*dto.FashionItemContractDTO) {
	item, fi := newTestItemWithFashion()
	itemStyle := style
	fi.Style = &itemStyle
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Ao", Slug: "ao"}
	return item, map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}
}

func TestScoreCandidateItemPersonalStyleBonusDependsOnRequestStyle(t *testing.T) {
	cases := []struct {
		name         string
		requestStyle []string
		want         float64
	}{
		{name: "profile is the only style signal", requestStyle: nil, want: 0.25},
		{name: "request style weakens the profile bonus", requestStyle: []string{"streetwear"}, want: 0.10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item, fashionItemMap := personalisedTestItem("minimalist")
			baseIntent := dto.ParsedIntent{StyleTarget: tc.requestStyle}
			withProfile := baseIntent
			withProfile.PersonalStyle = []string{"minimalist"}

			base, _ := ScoreCandidateItem(item, baseIntent, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)
			matched, tags := ScoreCandidateItem(item, withProfile, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

			if delta := matched - base; math.Abs(delta-tc.want) > 1e-9 {
				t.Fatalf("expected a %.2f personal-style bonus, got %.4f (%.4f vs %.4f)", tc.want, delta, matched, base)
			}
			if !containsString(tags, "personal-style-match") {
				t.Fatalf("expected the personal-style-match tag, got %v", tags)
			}
		})
	}
}

func TestScoreCandidateItemLearnedStyleBonusUsesConfiguredWeight(t *testing.T) {
	cases := []struct {
		name   string
		weight float64
		want   float64
	}{
		{name: "unconfigured keeps the default", weight: 0, want: 0.10},
		{name: "configured weight is applied", weight: 0.25, want: 0.25},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item, fashionItemMap := personalisedTestItem("minimalist")
			baseIntent := dto.ParsedIntent{}
			withLearned := dto.ParsedIntent{
				LearnedStyle:       []string{"minimalist"},
				LearnedStyleWeight: tc.weight,
			}

			base, _ := ScoreCandidateItem(item, baseIntent, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)
			matched, tags := ScoreCandidateItem(item, withLearned, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

			if delta := matched - base; math.Abs(delta-tc.want) > 1e-9 {
				t.Fatalf("expected a %.2f learned-style bonus, got %.4f", tc.want, delta)
			}
			if !containsString(tags, "learned-style-match") {
				t.Fatalf("expected the learned-style-match tag, got %v", tags)
			}
		})
	}
}

func TestScoreCandidateItemPenalizesAvoidedStyleWithoutFiltering(t *testing.T) {
	item, fashionItemMap := personalisedTestItem("minimalist")
	avoidIntent := dto.ParsedIntent{AvoidStyle: []string{"minimalist"}}

	base, _ := ScoreCandidateItem(item, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)
	avoided, tags := ScoreCandidateItem(item, avoidIntent, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	if delta := avoided - base; math.Abs(delta-(-0.30)) > 1e-9 {
		t.Fatalf("expected a %.2f avoid-style penalty, got %.4f", -0.30, delta)
	}
	if !containsString(tags, "avoid-style") {
		t.Fatalf("expected the avoid-style tag, got %v", tags)
	}
	if avoided <= 0 {
		t.Fatalf("an avoided style must never zero out the candidate, got %.4f", avoided)
	}

	ranked, _ := RankCandidates(
		[]types.CandidateForRanking{{Item: item, Source: "lexical"}},
		avoidIntent,
		types.RecommendationRetrievalQuery{},
		3,
		14,
		5,
		fashionItemMap,
	)
	if len(ranked) != 1 || ranked[0].Item.ID != item.ID {
		t.Fatalf("expected the avoided item to stay in the ranked pool, got %d candidates", len(ranked))
	}
}

func TestScoreCandidateItemPrefersDeclaredOverLearnedStyle(t *testing.T) {
	item, fashionItemMap := personalisedTestItem("minimalist")
	intent := dto.ParsedIntent{
		PersonalStyle:      []string{"minimalist"},
		LearnedStyle:       []string{"minimalist"},
		LearnedStyleWeight: 0.10,
	}

	base, _ := ScoreCandidateItem(item, dto.ParsedIntent{}, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)
	stacked, tags := ScoreCandidateItem(item, intent, types.RecommendationRetrievalQuery{}, 3, 14, fashionItemMap)

	// The declared bonus (0.25) must outrank the learned bonus (0.10); both are
	// additive soft signals, so the profile contribution is 0.35 in total.
	if delta := stacked - base; math.Abs(delta-0.35) > 1e-9 {
		t.Fatalf("expected declared and learned bonuses to stack to 0.35, got %.4f tags=%v", delta, tags)
	}
	if !containsString(tags, "personal-style-match") || !containsString(tags, "learned-style-match") {
		t.Fatalf("expected both personal and learned tags, got %v", tags)
	}
}

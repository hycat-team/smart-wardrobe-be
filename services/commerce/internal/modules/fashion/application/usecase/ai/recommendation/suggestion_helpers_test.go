package recommendation

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// outfitWithRoles builds a minimal final outfit response with the given roles.
func outfitWithRoles(roles ...string) *dto.RecommendedOutfitRes {
	groups := make([]*dto.RecommendedItemGroup, 0, len(roles))
	for _, role := range roles {
		groups = append(groups, &dto.RecommendedItemGroup{
			Role:    role,
			Primary: &dto.RecommendedItemRes{ID: uuid.New()},
		})
	}
	return &dto.RecommendedOutfitRes{Items: groups}
}

// poolCandidate builds one ranked pool entry for a category slug with the given
// retrieval relevance.
func poolCandidate(slug string, relevance float64) (types.CandidateForRanking, map[uuid.UUID]*dto.FashionItemContractDTO) {
	fashionID := uuid.New()
	categoryID := uuid.New()
	itemID := uuid.New()
	item := &wardrobeDto.WardrobeItemContractDTO{ID: itemID, FashionItemID: &fashionID}
	fashionMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fashionID: {
			ID:       fashionID,
			Category: &dto.CategoryBriefRes{ID: categoryID, Slug: slug},
		},
	}
	return types.CandidateForRanking{
		Item:        item,
		VectorScore: relevance,
	}, fashionMap
}

func TestDetectOutfitGapRolesMissingRole(t *testing.T) {
	topCandidate, fashionMap := poolCandidate("ao", 0.9)
	bottomCandidate, bottomMap := poolCandidate("quan", 0.9)
	for id, fi := range bottomMap {
		fashionMap[id] = fi
	}

	gaps := detectOutfitGapRoles(
		outfitWithRoles("top", "bottom"),
		[]types.CandidateForRanking{topCandidate, bottomCandidate},
		fashionMap,
		0.35,
	)
	if len(gaps) != 1 || gaps[0] != "footwear" {
		t.Fatalf("expected only footwear gap, got %v", gaps)
	}
}

func TestDetectOutfitGapRolesLowQualityRole(t *testing.T) {
	topCandidate, fashionMap := poolCandidate("ao", 0.9)
	bottomCandidate, bottomMap := poolCandidate("quan", 0.9)
	shoesCandidate, shoesMap := poolCandidate("giay", 0.1)
	for id, fi := range bottomMap {
		fashionMap[id] = fi
	}
	for id, fi := range shoesMap {
		fashionMap[id] = fi
	}

	gaps := detectOutfitGapRoles(
		outfitWithRoles("top", "bottom", "footwear"),
		[]types.CandidateForRanking{topCandidate, bottomCandidate, shoesCandidate},
		fashionMap,
		0.35,
	)
	if len(gaps) != 1 || gaps[0] != "footwear" {
		t.Fatalf("expected footwear to be flagged as low quality, got %v", gaps)
	}
}

func TestDetectOutfitGapRolesFullbodySatisfiesStructure(t *testing.T) {
	dressCandidate, fashionMap := poolCandidate("dam", 0.9)
	shoesCandidate, shoesMap := poolCandidate("giay", 0.9)
	for id, fi := range shoesMap {
		fashionMap[id] = fi
	}

	gaps := detectOutfitGapRoles(
		outfitWithRoles("fullbody", "footwear"),
		[]types.CandidateForRanking{dressCandidate, shoesCandidate},
		fashionMap,
		0.35,
	)
	if len(gaps) != 0 {
		t.Fatalf("expected no gap for a fullbody outfit with footwear, got %v", gaps)
	}
}

func TestDetectOutfitGapRolesFallbackCandidateCountsAsLowQuality(t *testing.T) {
	topCandidate, fashionMap := poolCandidate("ao", 0.9)
	bottomCandidate, bottomMap := poolCandidate("quan", 0.9)
	// A fallback candidate carries relevance 0 (merged by EnsureMinimumCandidatePool).
	shoesCandidate, shoesMap := poolCandidate("giay", 0)
	for id, fi := range bottomMap {
		fashionMap[id] = fi
	}
	for id, fi := range shoesMap {
		fashionMap[id] = fi
	}

	gaps := detectOutfitGapRoles(
		outfitWithRoles("top", "bottom", "footwear"),
		[]types.CandidateForRanking{topCandidate, bottomCandidate, shoesCandidate},
		fashionMap,
		0.35,
	)
	if len(gaps) != 1 || gaps[0] != "footwear" {
		t.Fatalf("expected low-quality footwear gap, got %v", gaps)
	}
}

func TestSuggestionReasonIsVietnamese(t *testing.T) {
	reason := suggestionReason("footwear", false, types.PersonalizationContext{})
	if reason == "" {
		t.Fatal("expected a reason")
	}
	if got := suggestionRoleLabel("footwear"); got != "giày" {
		t.Fatalf("expected Vietnamese role label, got %q", got)
	}
}

// --- catalog suggestion builder (033 US2/T039-T040) -------------------------

// fakeSuggestionCatalogCache fakes IPersonalizationCatalogCache in memory.
type fakeSuggestionCatalogCache struct {
	ids    []uuid.UUID
	getErr error
	setErr error
	sets   int
	setIDs []uuid.UUID
}

func (c *fakeSuggestionCatalogCache) Get(context.Context) ([]uuid.UUID, error) {
	return c.ids, c.getErr
}

func (c *fakeSuggestionCatalogCache) Set(_ context.Context, ids []uuid.UUID) error {
	c.sets++
	c.setIDs = ids
	if c.setErr != nil {
		return c.setErr
	}
	c.ids = ids
	return nil
}

// fakeSuggestionWardrobeContract serves the two wardrobe contract reads the
// suggestion builder needs (catalog ids and ownership).
type fakeSuggestionWardrobeContract struct {
	wardrobe_contract.IWardrobeContract
	catalogIDs   []uuid.UUID
	ownedIDs     []uuid.UUID
	catalogErr   error
	ownedErr     error
	catalogCalls int
	ownedCalls   int
}

func (f *fakeSuggestionWardrobeContract) GetSystemCatalogFashionItemIDs(context.Context, int) ([]uuid.UUID, error) {
	f.catalogCalls++
	return f.catalogIDs, f.catalogErr
}

func (f *fakeSuggestionWardrobeContract) GetOwnedFashionItemIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	f.ownedCalls++
	return f.ownedIDs, f.ownedErr
}

// fakeSuggestionFashionRepo fakes IFashionItemRepository and records the exact
// retrieval arguments so the test can prove hard filters and the query vector
// are reused unchanged (SC-014).
type fakeSuggestionFashionRepo struct {
	repositories.IFashionItemRepository
	items       map[uuid.UUID]*entities.FashionItem
	hybrid      []repositories.HybridCandidate
	hybridErr   error
	itemsErr    error
	hybridCalls int
	gotIDs      []uuid.UUID
	gotVector   []float32
	gotLexical  []string
	gotExcluded []string
	gotFilters  repositories.RecommendationHardFilters
	gotLimit    int
	gotRRF      int
}

func (f *fakeSuggestionFashionRepo) GetHybridCandidates(
	_ context.Context,
	fashionItemIDs []uuid.UUID,
	semanticVector []float32,
	lexicalTerms []string,
	excludedTerms []string,
	hardFilters repositories.RecommendationHardFilters,
	limit int,
	rrfK int,
) ([]repositories.HybridCandidate, error) {
	f.hybridCalls++
	f.gotIDs = fashionItemIDs
	f.gotVector = semanticVector
	f.gotLexical = lexicalTerms
	f.gotExcluded = excludedTerms
	f.gotFilters = hardFilters
	f.gotLimit = limit
	f.gotRRF = rrfK
	if f.hybridErr != nil {
		return nil, f.hybridErr
	}
	// Model the SQL query: only rows inside the requested id set come back, so
	// the ownership filter applied before retrieval is actually exercised.
	wanted := make(map[uuid.UUID]struct{}, len(fashionItemIDs))
	for _, id := range fashionItemIDs {
		wanted[id] = struct{}{}
	}
	filtered := make([]repositories.HybridCandidate, 0, len(f.hybrid))
	for _, candidate := range f.hybrid {
		if _, ok := wanted[candidate.ID]; ok {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func (f *fakeSuggestionFashionRepo) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	if f.itemsErr != nil {
		return nil, f.itemsErr
	}
	items := make([]*entities.FashionItem, 0, len(ids))
	for _, id := range ids {
		if item := f.items[id]; item != nil {
			items = append(items, item)
		}
	}
	return items, nil
}

// suggestionCatalogItem builds a system catalog fashion entity in one category.
func suggestionCatalogItem(slug, style string) *entities.FashionItem {
	item := &entities.FashionItem{Category: &entities.Category{Slug: slug}}
	item.ID = uuid.New()
	if style != "" {
		value := style
		item.Style = &value
	}
	return item
}

func suggestionHybrid(item *entities.FashionItem, relevance float64) repositories.HybridCandidate {
	return repositories.HybridCandidate{
		ID:             item.ID,
		FashionItemID:  &item.ID,
		VectorScore:    relevance,
		RetrievalScore: relevance,
	}
}

func suggestionItemIndex(items ...*entities.FashionItem) map[uuid.UUID]*entities.FashionItem {
	index := make(map[uuid.UUID]*entities.FashionItem, len(items))
	for _, item := range items {
		index[item.ID] = item
	}
	return index
}

// suggestionTestConfig mirrors the shipped personalization block.
func suggestionTestConfig() *config.Config {
	return &config.Config{
		Personalization: config.Personalization{
			Enabled:              true,
			GapScoreThreshold:    0.35,
			SuggestionMaxPerRole: 1,
			SuggestionMaxTotal:   3,
		},
		RAG: config.RAG{RrfKParameter: 60},
	}
}

// suggestionTestFilter builds the candidateFilterResult the workflow hands to
// the suggestion builder, including the retrieval context that must be reused.
func suggestionTestFilter(personalization types.PersonalizationContext, pool []types.CandidateForRanking) *candidateFilterResult {
	return &candidateFilterResult{
		Candidates: []types.CandidateForPrompt{{
			Item: &wardrobeDto.WardrobeItemContractDTO{ID: uuid.New()},
		}},
		FashionItemMap:  map[uuid.UUID]*dto.FashionItemContractDTO{},
		Personalization: personalization,
		Pool:            pool,
		RetrievalQuery: types.RecommendationRetrievalQuery{
			SemanticQuery:  "office winter outfit",
			LexicalTerms:   []types.RetrievalTerm{{Value: "office"}},
			ExcludedTerms:  []types.RetrievalTerm{{Value: "dark"}},
			HardFilters:    repositories.RecommendationHardFilters{Seasonality: []string{"winter"}},
			RewriterSource: "local",
		},
		QueryVector: entities.Vector{0.1, 0.2},
	}
}

func suggestionTestUseCase(
	cfg *config.Config,
	repo *fakeSuggestionFashionRepo,
	contract *fakeSuggestionWardrobeContract,
	cache *fakeSuggestionCatalogCache,
) *OutfitRecommendationUseCase {
	uc := &OutfitRecommendationUseCase{
		cfg:              cfg,
		logger:           nopRecommendationLogger{},
		wardrobeContract: contract,
		fashionItemRepo:  repo,
	}
	if cache != nil {
		uc.catalogCache = cache
	}
	return uc
}

func TestBuildCatalogSuggestionsOnePerRoleAndDistinctItems(t *testing.T) {
	topPrimary := suggestionCatalogItem("ao", "")
	topSecondary := suggestionCatalogItem("ao", "")
	bottom := suggestionCatalogItem("quan", "")
	shoes := suggestionCatalogItem("giay", "")

	repo := &fakeSuggestionFashionRepo{
		items: suggestionItemIndex(topPrimary, topSecondary, bottom, shoes),
		hybrid: []repositories.HybridCandidate{
			suggestionHybrid(topPrimary, 0.9),
			suggestionHybrid(topSecondary, 0.8),
			suggestionHybrid(bottom, 0.7),
			suggestionHybrid(shoes, 0.6),
		},
	}
	cfg := suggestionTestConfig()
	// A larger per-role limit must be ignored: one suggestion per role (FR-023).
	cfg.Personalization.SuggestionMaxPerRole = 5
	contract := &fakeSuggestionWardrobeContract{
		catalogIDs: []uuid.UUID{topPrimary.ID, topSecondary.ID, bottom.ID, shoes.ID},
	}
	uc := suggestionTestUseCase(cfg, repo, contract, &fakeSuggestionCatalogCache{})

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles(),
		suggestionTestFilter(types.PersonalizationContext{}, nil),
	)

	if len(got) != 3 {
		t.Fatalf("suggestions = %d (%v), want 3 (one per gap role)", len(got), suggestionRoles(got))
	}
	if want := []string{"top", "bottom", "footwear"}; !reflect.DeepEqual(suggestionRoles(got), want) {
		t.Fatalf("roles = %v, want %v", suggestionRoles(got), want)
	}
	if got[0].FashionItem == nil || got[0].FashionItem.ID != topPrimary.ID {
		t.Fatalf("top suggestion = %+v, want the highest relevance item", got[0].FashionItem)
	}
	seen := map[uuid.UUID]bool{}
	for _, suggestion := range got {
		if suggestion.FashionItem == nil {
			t.Fatal("suggestion without fashion item")
		}
		if seen[suggestion.FashionItem.ID] {
			t.Fatalf("fashion item %s suggested twice", suggestion.FashionItem.ID)
		}
		seen[suggestion.FashionItem.ID] = true
		if suggestion.Reason == "" {
			t.Fatalf("suggestion for role %q has no reason", suggestion.Role)
		}
	}
}

func TestBuildCatalogSuggestionsCapsTotal(t *testing.T) {
	top := suggestionCatalogItem("ao", "")
	bottom := suggestionCatalogItem("quan", "")
	shoes := suggestionCatalogItem("giay", "")

	repo := &fakeSuggestionFashionRepo{
		items: suggestionItemIndex(top, bottom, shoes),
		hybrid: []repositories.HybridCandidate{
			suggestionHybrid(top, 0.9),
			suggestionHybrid(bottom, 0.8),
			suggestionHybrid(shoes, 0.7),
		},
	}
	cfg := suggestionTestConfig()
	cfg.Personalization.SuggestionMaxTotal = 2
	contract := &fakeSuggestionWardrobeContract{catalogIDs: []uuid.UUID{top.ID, bottom.ID, shoes.ID}}
	uc := suggestionTestUseCase(cfg, repo, contract, &fakeSuggestionCatalogCache{})

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles(),
		suggestionTestFilter(types.PersonalizationContext{}, nil),
	)

	if len(got) != 2 {
		t.Fatalf("suggestions = %d (%v), want 2 (SuggestionMaxTotal)", len(got), suggestionRoles(got))
	}
}

func TestBuildCatalogSuggestionsReturnsNoneWhenOutfitIsComplete(t *testing.T) {
	pool := make([]types.CandidateForRanking, 0, 3)
	fashionMap := map[uuid.UUID]*dto.FashionItemContractDTO{}
	for _, slug := range []string{"ao", "quan", "giay"} {
		candidate, itemMap := poolCandidate(slug, 0.9)
		pool = append(pool, candidate)
		for id, item := range itemMap {
			fashionMap[id] = item
		}
	}

	repo := &fakeSuggestionFashionRepo{}
	filter := suggestionTestFilter(types.PersonalizationContext{}, pool)
	filter.FashionItemMap = fashionMap
	uc := suggestionTestUseCase(suggestionTestConfig(), repo, &fakeSuggestionWardrobeContract{}, &fakeSuggestionCatalogCache{})

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom", "footwear"),
		filter,
	)

	if len(got) != 0 {
		t.Fatalf("suggestions = %v, want none for a complete outfit", suggestionRoles(got))
	}
	if repo.hybridCalls != 0 {
		t.Fatalf("catalog retrieval ran %d times for a complete outfit, want 0", repo.hybridCalls)
	}
}

func TestBuildCatalogSuggestionsSkipsOwnedFashionItems(t *testing.T) {
	ownedShoes := suggestionCatalogItem("giay", "")
	otherShoes := suggestionCatalogItem("giay", "")

	repo := &fakeSuggestionFashionRepo{
		items: suggestionItemIndex(ownedShoes, otherShoes),
		hybrid: []repositories.HybridCandidate{
			suggestionHybrid(ownedShoes, 0.9),
			suggestionHybrid(otherShoes, 0.6),
		},
	}
	contract := &fakeSuggestionWardrobeContract{
		catalogIDs: []uuid.UUID{ownedShoes.ID, otherShoes.ID},
		ownedIDs:   []uuid.UUID{ownedShoes.ID},
	}
	uc := suggestionTestUseCase(suggestionTestConfig(), repo, contract, &fakeSuggestionCatalogCache{})

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom"),
		suggestionTestFilter(types.PersonalizationContext{}, nil),
	)

	if len(got) != 1 {
		t.Fatalf("suggestions = %d (%v), want 1", len(got), suggestionRoles(got))
	}
	if got[0].FashionItem == nil || got[0].FashionItem.ID != otherShoes.ID {
		t.Fatalf("suggested %+v, want the unowned item", got[0].FashionItem)
	}
	if len(repo.gotIDs) != 1 || repo.gotIDs[0] != otherShoes.ID {
		t.Fatalf("retrieval ids = %v, want only the unowned item", repo.gotIDs)
	}
}

func TestBuildCatalogSuggestionsReusesOutfitRetrievalContext(t *testing.T) {
	shoes := suggestionCatalogItem("giay", "")
	repo := &fakeSuggestionFashionRepo{
		items:  suggestionItemIndex(shoes),
		hybrid: []repositories.HybridCandidate{suggestionHybrid(shoes, 0.9)},
	}
	contract := &fakeSuggestionWardrobeContract{catalogIDs: []uuid.UUID{shoes.ID}}
	uc := suggestionTestUseCase(suggestionTestConfig(), repo, contract, &fakeSuggestionCatalogCache{})
	filter := suggestionTestFilter(types.PersonalizationContext{}, nil)

	uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom"),
		filter,
	)

	if !reflect.DeepEqual(repo.gotFilters, filter.RetrievalQuery.HardFilters) {
		t.Fatalf("hard filters = %+v, want the outfit filters %+v", repo.gotFilters, filter.RetrievalQuery.HardFilters)
	}
	if !reflect.DeepEqual(repo.gotVector, []float32(filter.QueryVector)) {
		t.Fatalf("query vector = %v, want %v", repo.gotVector, filter.QueryVector)
	}
	if !reflect.DeepEqual(repo.gotLexical, []string{"office"}) {
		t.Fatalf("lexical terms = %v, want [office]", repo.gotLexical)
	}
	if !reflect.DeepEqual(repo.gotExcluded, []string{"dark"}) {
		t.Fatalf("excluded terms = %v, want [dark]", repo.gotExcluded)
	}
	if repo.gotLimit != 1 || repo.gotRRF != 60 {
		t.Fatalf("limit/rrf = %d/%d, want 1/60", repo.gotLimit, repo.gotRRF)
	}
}

func TestBuildCatalogSuggestionsPrefersPersonalStyleOnTie(t *testing.T) {
	minimalist := suggestionCatalogItem("giay", "minimalist")
	streetwear := suggestionCatalogItem("giay", "streetwear")

	repo := &fakeSuggestionFashionRepo{
		items: suggestionItemIndex(minimalist, streetwear),
		hybrid: []repositories.HybridCandidate{
			suggestionHybrid(minimalist, 0.8),
			suggestionHybrid(streetwear, 0.8),
		},
	}
	contract := &fakeSuggestionWardrobeContract{catalogIDs: []uuid.UUID{minimalist.ID, streetwear.ID}}
	uc := suggestionTestUseCase(suggestionTestConfig(), repo, contract, &fakeSuggestionCatalogCache{})
	personalization := types.PersonalizationContext{
		Active:         true,
		ExplicitStyles: []string{"minimalist"},
	}

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom"),
		suggestionTestFilter(personalization, nil),
	)

	if len(got) != 1 || got[0].FashionItem == nil {
		t.Fatalf("suggestions = %v, want exactly one footwear suggestion", got)
	}
	if got[0].FashionItem.ID != minimalist.ID {
		t.Fatalf("suggested %s, want the personal-style match %s (SC-014)", got[0].FashionItem.ID, minimalist.ID)
	}
}

func TestBuildCatalogSuggestionsDropsAvoidedStyleOnTie(t *testing.T) {
	avoided := suggestionCatalogItem("giay", "minimalist")
	neutral := suggestionCatalogItem("giay", "streetwear")

	repo := &fakeSuggestionFashionRepo{
		items: suggestionItemIndex(avoided, neutral),
		hybrid: []repositories.HybridCandidate{
			suggestionHybrid(avoided, 0.8),
			suggestionHybrid(neutral, 0.8),
		},
	}
	contract := &fakeSuggestionWardrobeContract{catalogIDs: []uuid.UUID{avoided.ID, neutral.ID}}
	uc := suggestionTestUseCase(suggestionTestConfig(), repo, contract, &fakeSuggestionCatalogCache{})
	personalization := types.PersonalizationContext{
		Active:         true,
		ExcludedStyles: []string{"minimalist"},
	}

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom"),
		suggestionTestFilter(personalization, nil),
	)

	if len(got) != 1 || got[0].FashionItem == nil {
		t.Fatalf("suggestions = %v, want exactly one footwear suggestion", got)
	}
	if got[0].FashionItem.ID != neutral.ID {
		t.Fatalf("suggested %s, want the non-avoided item %s", got[0].FashionItem.ID, neutral.ID)
	}
}

func TestBuildCatalogSuggestionsAvailableWithoutPersonalizationEntitlement(t *testing.T) {
	shoes := suggestionCatalogItem("giay", "")
	repo := &fakeSuggestionFashionRepo{
		items:  suggestionItemIndex(shoes),
		hybrid: []repositories.HybridCandidate{suggestionHybrid(shoes, 0.9)},
	}
	contract := &fakeSuggestionWardrobeContract{catalogIDs: []uuid.UUID{shoes.ID}}
	cfg := suggestionTestConfig()
	// Free user / kill switch off: the suggestion pass must still run, because
	// catalog gap-fill is not a personalization entitlement (FR-026).
	cfg.Personalization.Enabled = false
	uc := suggestionTestUseCase(cfg, repo, contract, &fakeSuggestionCatalogCache{})

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom"),
		suggestionTestFilter(types.PersonalizationContext{}, nil),
	)

	if len(got) != 1 || got[0].Role != "footwear" {
		t.Fatalf("suggestions = %v, want one footwear suggestion for a free user", suggestionRoles(got))
	}
}

func TestBuildCatalogSuggestionsFailsSoftWithoutOwnershipData(t *testing.T) {
	shoes := suggestionCatalogItem("giay", "")
	repo := &fakeSuggestionFashionRepo{
		items:  suggestionItemIndex(shoes),
		hybrid: []repositories.HybridCandidate{suggestionHybrid(shoes, 0.9)},
	}
	contract := &fakeSuggestionWardrobeContract{
		catalogIDs: []uuid.UUID{shoes.ID},
		ownedErr:   errors.New("wardrobe contract unavailable"),
	}
	uc := suggestionTestUseCase(suggestionTestConfig(), repo, contract, &fakeSuggestionCatalogCache{})

	got := uc.buildCatalogSuggestions(
		context.Background(),
		uuid.New(),
		outfitWithRoles("top", "bottom"),
		suggestionTestFilter(types.PersonalizationContext{}, nil),
	)

	if len(got) != 0 {
		t.Fatalf("suggestions = %v, want none when ownership is unknown (FR-021)", suggestionRoles(got))
	}
	if repo.hybridCalls != 0 {
		t.Fatalf("retrieval ran %d times without ownership data, want 0", repo.hybridCalls)
	}
}

func TestBestCatalogSuggestionNeverReusesAFashionItem(t *testing.T) {
	shared := &dto.FashionItemContractDTO{ID: uuid.New()}
	// The same fashion item listed under two roles is artificial, but it is the
	// exact shape the used-set guard defends against.
	candidates := []catalogSuggestion{
		{FashionItem: shared, Role: "footwear", Relevance: 0.9},
		{FashionItem: shared, Role: "accessory", Relevance: 0.9},
	}
	used := map[uuid.UUID]bool{}

	picked := bestCatalogSuggestion(candidates, "footwear", used, types.PersonalizationContext{})
	if picked == nil || picked.FashionItem.ID != shared.ID {
		t.Fatalf("first pick = %+v, want the shared item", picked)
	}
	used[picked.FashionItem.ID] = true

	if again := bestCatalogSuggestion(candidates, "accessory", used, types.PersonalizationContext{}); again != nil {
		t.Fatalf("second pick = %+v, want none: the item is already suggested", again)
	}
}

func TestGetCatalogFashionItemIDsFillsCacheOnMissAndSkipsContractOnHit(t *testing.T) {
	catalogIDs := []uuid.UUID{uuid.New(), uuid.New()}

	t.Run("miss fills the cache", func(t *testing.T) {
		contract := &fakeSuggestionWardrobeContract{catalogIDs: catalogIDs}
		cache := &fakeSuggestionCatalogCache{}
		uc := suggestionTestUseCase(suggestionTestConfig(), &fakeSuggestionFashionRepo{}, contract, cache)

		got := uc.getCatalogFashionItemIDs(context.Background())
		if !reflect.DeepEqual(got, catalogIDs) {
			t.Fatalf("ids = %v, want %v", got, catalogIDs)
		}
		if contract.catalogCalls != 1 || cache.sets != 1 {
			t.Fatalf("contract calls/cache sets = %d/%d, want 1/1", contract.catalogCalls, cache.sets)
		}
		if !reflect.DeepEqual(cache.setIDs, catalogIDs) {
			t.Fatalf("cached ids = %v, want %v", cache.setIDs, catalogIDs)
		}
	})

	t.Run("hit skips the contract", func(t *testing.T) {
		contract := &fakeSuggestionWardrobeContract{catalogIDs: []uuid.UUID{uuid.New()}}
		cache := &fakeSuggestionCatalogCache{ids: catalogIDs}
		uc := suggestionTestUseCase(suggestionTestConfig(), &fakeSuggestionFashionRepo{}, contract, cache)

		got := uc.getCatalogFashionItemIDs(context.Background())
		if !reflect.DeepEqual(got, catalogIDs) {
			t.Fatalf("ids = %v, want the cached %v", got, catalogIDs)
		}
		if contract.catalogCalls != 0 {
			t.Fatalf("contract called %d times on a cache hit, want 0", contract.catalogCalls)
		}
	})

	t.Run("contract error degrades to no ids", func(t *testing.T) {
		contract := &fakeSuggestionWardrobeContract{catalogErr: errors.New("contract down")}
		cache := &fakeSuggestionCatalogCache{}
		uc := suggestionTestUseCase(suggestionTestConfig(), &fakeSuggestionFashionRepo{}, contract, cache)

		if got := uc.getCatalogFashionItemIDs(context.Background()); len(got) != 0 {
			t.Fatalf("ids = %v, want none on a contract error", got)
		}
		if cache.sets != 0 {
			t.Fatalf("cache sets = %d, want 0", cache.sets)
		}
	})
}

func suggestionRoles(suggestions []dto.OutfitSuggestionRes) []string {
	roles := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		roles = append(roles, suggestion.Role)
	}
	return roles
}

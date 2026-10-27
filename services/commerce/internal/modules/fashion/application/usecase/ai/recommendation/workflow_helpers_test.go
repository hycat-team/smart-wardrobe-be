package recommendation

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// fakeRotationWardrobeContract serves the two wardrobe contract calls the
// rotation needs: the stale item read and the last_recommended_at write.
type fakeRotationWardrobeContract struct {
	wardrobe_contract.IWardrobeContract
	staleItems []*wardrobeDto.WardrobeItemContractDTO
	staleErr   error
	gotLimit   int
	staleCalls int
	touchedIDs []uuid.UUID
	touchAt    time.Time
	touchErr   error
	touchCalls int
}

func (f *fakeRotationWardrobeContract) GetStaleActiveWardrobeItemsByUserID(_ context.Context, _ uuid.UUID, limit int) ([]*wardrobeDto.WardrobeItemContractDTO, error) {
	f.staleCalls++
	f.gotLimit = limit
	return f.staleItems, f.staleErr
}

func (f *fakeRotationWardrobeContract) TouchLastRecommendedAt(_ context.Context, wardrobeItemIDs []uuid.UUID, at time.Time) error {
	f.touchCalls++
	f.touchedIDs = wardrobeItemIDs
	f.touchAt = at
	return f.touchErr
}

// rotationTestItem builds one wardrobe row plus its fashion metadata.
func rotationTestItem(slug string, style string, lastUsedAt, lastRecommendedAt *time.Time) (*wardrobeDto.WardrobeItemContractDTO, *dto.FashionItemContractDTO) {
	fashionID := uuid.New()
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:                uuid.New(),
		UserID:            uuid.New(),
		FashionItemID:     &fashionID,
		LastUsedAt:        lastUsedAt,
		LastRecommendedAt: lastRecommendedAt,
		CreatedAt:         time.Now().AddDate(0, 0, -60),
	}
	fashion := &dto.FashionItemContractDTO{
		ID:       fashionID,
		Category: &dto.CategoryBriefRes{ID: uuid.New(), Name: slug, Slug: slug},
	}
	if style != "" {
		value := style
		fashion.Style = &value
	}
	return item, fashion
}

// rotationTestSettings mirrors the shipped rag rotation block but keeps the
// slot count small so a handful of fixtures is enough.
func rotationTestSettings() rotationSettings {
	return rotationSettings{
		recency: ranking.RecencyWindow{
			RecentlyWornDays:        7,
			RecentlyRecommendedDays: 3,
			LongUnwornDays:          14,
		},
		staleEligibleDays:   30,
		staleCandidateSlots: 2,
		cooldownTolerance:   0.15,
	}
}

func rotationTestUseCase(contract *fakeRotationWardrobeContract) *OutfitRecommendationUseCase {
	return &OutfitRecommendationUseCase{
		cfg: &config.Config{RAG: config.RAG{
			RecentlyWornPenaltyDays:        7,
			RecentlyRecommendedPenaltyDays: 3,
			LongUnwornBonusDays:            14,
			StaleCandidateSlots:            2,
			StaleEligibleDays:              30,
			CooldownScoreTolerance:         0.15,
		}},
		logger:           nopRecommendationLogger{},
		wardrobeContract: contract,
	}
}

func fashionIndex(items ...*dto.FashionItemContractDTO) map[uuid.UUID]*dto.FashionItemContractDTO {
	index := make(map[uuid.UUID]*dto.FashionItemContractDTO, len(items))
	for _, item := range items {
		index[item.ID] = item
	}
	return index
}

// TestRotationSettingsFallBackToShippedDefaults makes sure a missing rag block
// still rotates with the documented 4/30/0.15 values.
func TestRotationSettingsFallBackToShippedDefaults(t *testing.T) {
	uc := &OutfitRecommendationUseCase{cfg: &config.Config{}}
	settings := uc.rotationSettings()

	if settings.staleCandidateSlots != defaultStaleCandidateSlots ||
		settings.staleEligibleDays != defaultStaleEligibleDays ||
		settings.cooldownTolerance != defaultCooldownTolerance {
		t.Fatalf("unexpected rotation defaults: %+v", settings)
	}
	if settings.recency.RecentlyWornDays != 7 || settings.recency.RecentlyRecommendedDays != 3 || settings.recency.LongUnwornDays != 14 {
		t.Fatalf("unexpected recency defaults: %+v", settings.recency)
	}
}

func TestDedupeWardrobeItemsByFashionItemIDKeepsLatestSignals(t *testing.T) {
	twentyDaysAgo := time.Now().AddDate(0, 0, -20)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	fiveDaysAgo := time.Now().AddDate(0, 0, -5)
	yesterday := time.Now().AddDate(0, 0, -1)

	older, fashion := rotationTestItem("ao", "", &twentyDaysAgo, &fiveDaysAgo)
	newer := *older
	newer.ID = uuid.New()
	newer.LastUsedAt = &twoDaysAgo
	newer.LastRecommendedAt = &yesterday

	// A row without metadata must survive untouched.
	orphan := &wardrobeDto.WardrobeItemContractDTO{ID: uuid.New()}

	got := dedupeWardrobeItemsByFashionItemID([]*wardrobeDto.WardrobeItemContractDTO{older, &newer, orphan})

	if len(got) != 2 {
		t.Fatalf("deduped = %d rows, want 2 (one copy + the orphan)", len(got))
	}
	if got[0].FashionItemID == nil || *got[0].FashionItemID != fashion.ID {
		t.Fatalf("first row = %+v, want the merged copy of %s", got[0], fashion.ID)
	}
	if got[0].LastUsedAt == nil || !got[0].LastUsedAt.Equal(twoDaysAgo) {
		t.Fatalf("last_used_at = %v, want the max (2 days ago)", got[0].LastUsedAt)
	}
	if got[0].LastRecommendedAt == nil || !got[0].LastRecommendedAt.Equal(yesterday) {
		t.Fatalf("last_recommended_at = %v, want the max (yesterday)", got[0].LastRecommendedAt)
	}
	if got[1].ID != orphan.ID {
		t.Fatalf("second row = %s, want the orphan %s", got[1].ID, orphan.ID)
	}
}

func TestFetchStaleCandidatesReservesSlotsAndKeepsEligibleSuperset(t *testing.T) {
	fortyFiveDaysAgo := time.Now().AddDate(0, 0, -45)
	fortyDaysAgo := time.Now().AddDate(0, 0, -40)
	thirtyFiveDaysAgo := time.Now().AddDate(0, 0, -35)
	tenDaysAgo := time.Now().AddDate(0, 0, -10)
	yesterday := time.Now().AddDate(0, 0, -1)

	neverUsed, neverUsedFashion := rotationTestItem("ao", "", nil, nil)
	oldest, oldestFashion := rotationTestItem("quan", "", &fortyDaysAgo, nil)
	recentRecommended, recentRecommendedFashion := rotationTestItem("giay", "", &fortyFiveDaysAgo, &yesterday)
	recentlyUsed, recentlyUsedFashion := rotationTestItem("mu", "", &yesterday, nil)
	fourth, fourthFashion := rotationTestItem("phu-kien", "", &tenDaysAgo, nil)
	fifth, fifthFashion := rotationTestItem("ao-khoac", "", &thirtyFiveDaysAgo, nil)
	// Recently used, so not stale on its own copy.
	notStale, notStaleFashion := rotationTestItem("ao", "", &tenDaysAgo, nil)

	contract := &fakeRotationWardrobeContract{
		staleItems: []*wardrobeDto.WardrobeItemContractDTO{notStale, recentlyUsed, fifth, fourth, recentRecommended, oldest, neverUsed},
	}
	uc := rotationTestUseCase(contract)
	settings := rotationTestSettings()

	got := uc.fetchStaleCandidates(
		context.Background(),
		uuid.New(),
		200,
		types.RecommendationRetrievalQuery{},
		fashionIndex(neverUsedFashion, oldestFashion, recentRecommendedFashion, recentlyUsedFashion, fourthFashion, fifthFashion, notStaleFashion),
		settings,
		time.Now(),
	)

	if contract.gotLimit != 200 {
		t.Fatalf("stale limit = %d, want the wide plan maximum 200", contract.gotLimit)
	}
	if len(got.Candidates) != settings.staleCandidateSlots {
		t.Fatalf("slots = %d, want %d", len(got.Candidates), settings.staleCandidateSlots)
	}
	if len(got.EligibleIDs) != 4 {
		t.Fatalf("eligible = %v, want all 4 stale-eligible items (not only the 2 slots)", got.EligibleIDs)
	}
	wantOrder := []string{neverUsedFashion.ID.String(), recentRecommendedFashion.ID.String()}
	gotOrder := []string{got.SlotIDs[0], got.SlotIDs[1]}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("slot order = %v, want %v", gotOrder, wantOrder)
	}
	for _, slotID := range got.SlotIDs {
		if !containsString(got.EligibleIDs, slotID) {
			t.Fatalf("slot %s is not in stale_eligible_item_ids %v", slotID, got.EligibleIDs)
		}
	}
	if containsString(got.EligibleIDs, recentlyUsedFashion.ID.String()) || containsString(got.EligibleIDs, notStaleFashion.ID.String()) {
		t.Fatalf("recently used items leaked into stale eligibility: %v", got.EligibleIDs)
	}
	for _, candidate := range got.Candidates {
		if candidate.Source != types.CandidateSourceStale {
			t.Fatalf("candidate source = %q, want stale", candidate.Source)
		}
		if candidate.ItemContext != outfititemcontext.UserWardrobe {
			t.Fatalf("candidate context = %q, want user wardrobe", candidate.ItemContext)
		}
	}
}

func TestFetchStaleCandidatesSharesUseSignalAcrossCopies(t *testing.T) {
	fortyDaysAgo := time.Now().AddDate(0, 0, -40)
	yesterday := time.Now().AddDate(0, 0, -1)

	staleCopy, fashion := rotationTestItem("ao", "", &fortyDaysAgo, nil)
	usedCopy := *staleCopy
	usedCopy.ID = uuid.New()
	usedCopy.LastUsedAt = &yesterday

	contract := &fakeRotationWardrobeContract{
		staleItems: []*wardrobeDto.WardrobeItemContractDTO{staleCopy, &usedCopy},
	}
	uc := rotationTestUseCase(contract)

	got := uc.fetchStaleCandidates(
		context.Background(),
		uuid.New(),
		200,
		types.RecommendationRetrievalQuery{},
		fashionIndex(fashion),
		rotationTestSettings(),
		time.Now(),
	)

	if len(got.Candidates) != 0 || len(got.EligibleIDs) != 0 {
		t.Fatalf("a copy used yesterday must make the item ineligible, got slots=%v eligible=%v", got.SlotIDs, got.EligibleIDs)
	}
}

func TestFetchStaleCandidatesAppliesHardFilters(t *testing.T) {
	fortyDaysAgo := time.Now().AddDate(0, 0, -40)
	winter := "Mùa đông"
	summer := "Mùa hè"

	winterItem, winterFashion := rotationTestItem("ao", "", &fortyDaysAgo, nil)
	winterFashion.Seasonality = &winter
	summerItem, summerFashion := rotationTestItem("quan", "", &fortyDaysAgo, nil)
	summerFashion.Seasonality = &summer

	contract := &fakeRotationWardrobeContract{
		staleItems: []*wardrobeDto.WardrobeItemContractDTO{winterItem, summerItem},
	}
	uc := rotationTestUseCase(contract)

	got := uc.fetchStaleCandidates(
		context.Background(),
		uuid.New(),
		200,
		types.RecommendationRetrievalQuery{
			HardFilters: repositories.RecommendationHardFilters{Seasonality: []string{"winter"}},
		},
		fashionIndex(winterFashion, summerFashion),
		rotationTestSettings(),
		time.Now(),
	)

	if len(got.EligibleIDs) != 1 || got.EligibleIDs[0] != winterFashion.ID.String() {
		t.Fatalf("eligible = %v, want only the winter item", got.EligibleIDs)
	}
}

func TestFetchStaleCandidatesFailsSoftOnContractError(t *testing.T) {
	contract := &fakeRotationWardrobeContract{staleErr: errors.New("contract down")}
	uc := rotationTestUseCase(contract)

	got := uc.fetchStaleCandidates(
		context.Background(),
		uuid.New(),
		200,
		types.RecommendationRetrievalQuery{},
		map[uuid.UUID]*dto.FashionItemContractDTO{},
		rotationTestSettings(),
		time.Now(),
	)

	if len(got.Candidates) != 0 || len(got.SlotIDs) != 0 || len(got.EligibleIDs) != 0 {
		t.Fatalf("a contract error must yield no stale slots, got %+v", got)
	}
}

// rotationObservabilityFilter builds a two-item prompt pool: a recently worn
// item matching the saved style and a stale item.
func rotationObservabilityFilter() (*candidateFilterResult, *wardrobeDto.WardrobeItemContractDTO, *wardrobeDto.WardrobeItemContractDTO) {
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	fortyDaysAgo := time.Now().AddDate(0, 0, -40)
	yesterday := time.Now().AddDate(0, 0, -1)

	recent, recentFashion := rotationTestItem("ao", "minimalist", &twoDaysAgo, &yesterday)
	stale, staleFashion := rotationTestItem("quan", "", &fortyDaysAgo, nil)

	return &candidateFilterResult{
		Candidates: []types.CandidateForPrompt{
			{Item: recent, ItemContext: outfititemcontext.UserWardrobe},
			{Item: stale, ItemContext: outfititemcontext.UserWardrobe},
		},
		FashionItemMap: fashionIndex(recentFashion, staleFashion),
		Personalization: types.PersonalizationContext{
			Active:         true,
			ExplicitStyles: []string{"minimalist"},
		},
		Stats: types.RerankStats{
			StaleSlotsAdded:       1,
			CooldownSubstitutions: 2,
			StaleItemIDs:          []string{staleFashion.ID.String()},
			StaleEligibleItemIDs:  []string{staleFashion.ID.String()},
		},
	}, recent, stale
}

func TestRecordRotationObservabilityComputesCountersAndTouches(t *testing.T) {
	filter, recent, stale := rotationObservabilityFilter()
	finalRes := &dto.RecommendedOutfitRes{
		Items: []*dto.RecommendedItemGroup{
			{
				Role:         "top",
				Primary:      &dto.RecommendedItemRes{ID: recent.ID, ItemContext: string(outfititemcontext.UserWardrobe)},
				Alternatives: []*dto.RecommendedItemRes{{ID: stale.ID, ItemContext: string(outfititemcontext.UserWardrobe)}},
			},
		},
	}
	contract := &fakeRotationWardrobeContract{}
	uc := rotationTestUseCase(contract)

	stats := uc.recordRotationObservability(context.Background(), uuid.New(), uuid.New(), finalRes, filter, rotationTestSettings(), time.Now())

	recentFashionID := recent.FashionItemID.String()
	staleFashionID := stale.FashionItemID.String()
	if want := []string{recentFashionID, staleFashionID}; !reflect.DeepEqual(stats.ReturnedItemIDs, want) {
		t.Fatalf("returned ids = %v, want %v", stats.ReturnedItemIDs, want)
	}
	if !reflect.DeepEqual(stats.StaleItemIDs, []string{staleFashionID}) {
		t.Fatalf("stale ids = %v, want the returned stale item only", stats.StaleItemIDs)
	}
	if !reflect.DeepEqual(stats.CooldownItemIDs, []string{recentFashionID}) {
		t.Fatalf("cooldown ids = %v, want the recently worn item only", stats.CooldownItemIDs)
	}
	if stats.RecentlyUsedRepeatRatio != 0.5 {
		t.Fatalf("recently used ratio = %.2f, want 0.5", stats.RecentlyUsedRepeatRatio)
	}
	if stats.RecentlyRecommendedRepeatRatio != 0.5 {
		t.Fatalf("recently recommended ratio = %.2f, want 0.5", stats.RecentlyRecommendedRepeatRatio)
	}
	if stats.PersonalStyleMatchRatio != 1 {
		t.Fatalf("personal style match ratio = %.2f, want 1 (the top matches the saved style)", stats.PersonalStyleMatchRatio)
	}
	if stats.StaleSlotsAdded != 1 || stats.CooldownSubstitutions != 2 {
		t.Fatalf("counters = %d/%d, want the ranking counters carried through", stats.StaleSlotsAdded, stats.CooldownSubstitutions)
	}
	if !reflect.DeepEqual(stats.StaleEligibleItemIDs, []string{staleFashionID}) {
		t.Fatalf("stale eligible ids = %v, want the denominator kept", stats.StaleEligibleItemIDs)
	}
	if contract.touchCalls != 1 {
		t.Fatalf("TouchLastRecommendedAt calls = %d, want 1", contract.touchCalls)
	}
	wantTouched := map[uuid.UUID]bool{recent.ID: true, stale.ID: true}
	if len(contract.touchedIDs) != 2 {
		t.Fatalf("touched ids = %v, want the primary and the alternative", contract.touchedIDs)
	}
	for _, id := range contract.touchedIDs {
		if !wantTouched[id] {
			t.Fatalf("unexpected touched wardrobe item %s", id)
		}
	}
}

func TestRecordRotationObservabilitySkipsBrandItems(t *testing.T) {
	brandFashionID := uuid.New()
	brandCandidate := types.CandidateForPrompt{
		Item:        &wardrobeDto.WardrobeItemContractDTO{ID: brandFashionID, FashionItemID: &brandFashionID},
		ItemContext: outfititemcontext.BrandItem,
	}
	filter := &candidateFilterResult{
		Candidates:     []types.CandidateForPrompt{brandCandidate},
		FashionItemMap: map[uuid.UUID]*dto.FashionItemContractDTO{},
	}
	finalRes := &dto.RecommendedOutfitRes{
		Items: []*dto.RecommendedItemGroup{
			{
				Role:    "top",
				Primary: &dto.RecommendedItemRes{ID: brandFashionID, ItemContext: string(outfititemcontext.BrandItem)},
			},
		},
	}
	contract := &fakeRotationWardrobeContract{}
	uc := rotationTestUseCase(contract)

	stats := uc.recordRotationObservability(context.Background(), uuid.New(), uuid.New(), finalRes, filter, rotationTestSettings(), time.Now())

	if !reflect.DeepEqual(stats.ReturnedItemIDs, []string{brandFashionID.String()}) {
		t.Fatalf("returned ids = %v, want the brand fashion item", stats.ReturnedItemIDs)
	}
	if contract.touchCalls != 0 {
		t.Fatalf("TouchLastRecommendedAt must not run for brand items, calls = %d", contract.touchCalls)
	}
}

func TestRecordRotationObservabilityToleratesTouchFailure(t *testing.T) {
	item, fashion := rotationTestItem("ao", "", nil, nil)
	filter := &candidateFilterResult{
		Candidates:     []types.CandidateForPrompt{{Item: item, ItemContext: outfititemcontext.UserWardrobe}},
		FashionItemMap: fashionIndex(fashion),
	}
	finalRes := &dto.RecommendedOutfitRes{
		Items: []*dto.RecommendedItemGroup{
			{Role: "top", Primary: &dto.RecommendedItemRes{ID: item.ID, ItemContext: string(outfititemcontext.UserWardrobe)}},
		},
	}
	contract := &fakeRotationWardrobeContract{touchErr: errors.New("write failed")}
	uc := rotationTestUseCase(contract)

	stats := uc.recordRotationObservability(context.Background(), uuid.New(), uuid.New(), finalRes, filter, rotationTestSettings(), time.Now())

	if contract.touchCalls != 1 {
		t.Fatalf("TouchLastRecommendedAt calls = %d, want 1", contract.touchCalls)
	}
	if len(stats.ReturnedItemIDs) != 1 {
		t.Fatalf("returned ids = %v, want one item even when the write fails", stats.ReturnedItemIDs)
	}
}

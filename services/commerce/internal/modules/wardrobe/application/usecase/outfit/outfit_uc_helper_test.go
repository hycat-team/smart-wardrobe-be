package outfit

import (
	"context"
	"testing"

	brand_dto "smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	brand_contract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

// --- pure helpers -----------------------------------------------------------

func TestEnsureUniqueOutfitItemsRejectsDuplicateFashionItem(t *testing.T) {
	id := uuid.New()
	err := ensureUniqueOutfitItems([]dto.SaveOutfitItemReq{
		{FashionItemID: id},
		{FashionItemID: id},
	})
	if err == nil {
		t.Fatal("expected duplicate outfit item error")
	}
	want := wardrobeerrors.ErrDuplicateOutfitItem(id)
	if err.Error() != want.Error() {
		t.Fatalf("error = %q, want %q", err.Error(), want.Error())
	}
}

func TestEnsureUniqueOutfitItemsAcceptsDistinctItems(t *testing.T) {
	if err := ensureUniqueOutfitItems([]dto.SaveOutfitItemReq{
		{FashionItemID: uuid.New()},
		{FashionItemID: uuid.New()},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExceedsWardrobeLimit(t *testing.T) {
	cases := []struct {
		name       string
		current    int
		newClones  int
		maxItems   int
		wantExceed bool
	}{
		{"no clone never exceeds", 10, 0, 10, false},
		{"exactly at limit", 8, 2, 10, false},
		{"one over limit", 9, 2, 10, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := exceedsWardrobeLimit(tc.current, tc.newClones, tc.maxItems)
			if got != tc.wantExceed {
				t.Fatalf("exceedsWardrobeLimit(%d,%d,%d) = %v, want %v", tc.current, tc.newClones, tc.maxItems, got, tc.wantExceed)
			}
		})
	}
}

func TestNewlyAddedFashionItemIDs(t *testing.T) {
	keep := uuid.New()
	add := uuid.New()
	extra := uuid.New()

	got := newlyAddedFashionItemIDs(
		[]uuid.UUID{keep},
		[]uuid.UUID{keep, add, extra, add},
	)
	if len(got) != 2 || got[0] != add || got[1] != extra {
		t.Fatalf("newlyAddedFashionItemIDs = %v, want [%v %v]", got, add, extra)
	}
}

func TestFashionItemIDsOfDedupesPreservingOrder(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	items := []*entities.OutfitItem{
		{FashionItemID: first},
		{FashionItemID: second},
		{FashionItemID: first},
	}
	got := fashionItemIDsOf(items)
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("fashionItemIDsOf = %v, want [%v %v]", got, first, second)
	}
}

func TestPickOwnedWardrobeRowPrefersInWardrobe(t *testing.T) {
	fashionID := uuid.New()
	sold := &entities.WardrobeItem{FashionItemID: &fashionID, Status: wardrobestatus.Sold}
	processing := &entities.WardrobeItem{FashionItemID: &fashionID, Status: wardrobestatus.Processing}
	inWardrobe := &entities.WardrobeItem{FashionItemID: &fashionID, Status: wardrobestatus.InWardrobe}

	if got := pickOwnedWardrobeRow([]*entities.WardrobeItem{sold, processing, inWardrobe}); got != inWardrobe {
		t.Fatalf("expected InWardrobe row, got status %v", got.Status)
	}
	if got := pickOwnedWardrobeRow([]*entities.WardrobeItem{sold, processing}); got != processing {
		t.Fatalf("expected non-sold fallback, got status %v", got.Status)
	}
	if got := pickOwnedWardrobeRow([]*entities.WardrobeItem{sold}); got != sold {
		t.Fatal("expected sold fallback when it is the only row")
	}
}

// --- resolution -------------------------------------------------------------

type fakeWardrobeRepo struct {
	repositories.IWardrobeItemRepository
	owned      []*entities.WardrobeItem
	catalog    []*entities.WardrobeItem
	ownedErr   error
	catalogErr error
}

func (f *fakeWardrobeRepo) GetByUserIDAndFashionItemIDs(context.Context, uuid.UUID, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return f.owned, f.ownedErr
}

func (f *fakeWardrobeRepo) GetActiveSystemCatalogItemsByIDs(context.Context, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return f.catalog, f.catalogErr
}

type fakeBrandContract struct {
	brand_contract.IBrandContract
	eligible map[uuid.UUID]bool
}

func (f *fakeBrandContract) CheckBrandItemEligibility(_ context.Context, _ uuid.UUID, fashionItemID uuid.UUID) (bool, *brand_dto.BrandItemContractDTO, *fashion_dto.FashionItemContractDTO, error) {
	return f.eligible[fashionItemID], nil, nil, nil
}

func newResolveUC(repo *fakeWardrobeRepo, brand *fakeBrandContract) *OutfitUseCase {
	return &OutfitUseCase{wardrobeRepo: repo, brandContract: brand}
}

func TestResolveAutoClonesCatalogItemWithoutAnalysisTask(t *testing.T) {
	fashionID := uuid.New()
	price := 250000.0
	catalog := &entities.WardrobeItem{
		FashionItemID: &fashionID,
		Price:         &price,
		ItemType:      itemtype.SystemCatalogItem,
		Status:        wardrobestatus.InWardrobe,
	}
	catalog.ID = uuid.New()

	uc := newResolveUC(
		&fakeWardrobeRepo{catalog: []*entities.WardrobeItem{catalog}},
		&fakeBrandContract{eligible: map[uuid.UUID]bool{}},
	)

	resolution, err := uc.resolveOutfitItems(context.Background(), uuid.New(), uuid.Nil, []dto.SaveOutfitItemReq{
		{FashionItemID: fashionID, Scale: 1, LayerOrder: 1},
	}, 10, map[uuid.UUID]bool{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolution.cloneItems) != 1 {
		t.Fatalf("clone items = %d, want 1", len(resolution.cloneItems))
	}
	clone := resolution.cloneItems[0]
	if clone.ItemType != itemtype.UserItem || clone.Status != wardrobestatus.InWardrobe {
		t.Fatalf("clone type/status = %v/%v, want UserItem/InWardrobe", clone.ItemType, clone.Status)
	}
	if clone.Price == nil || *clone.Price != price {
		t.Fatalf("clone price = %v, want %v", clone.Price, price)
	}
	if clone.TaskID != nil {
		t.Fatal("clone must not start an AI analysis task")
	}
	if got := resolution.items[0].ItemContext; got != outfititemcontext.UserWardrobe {
		t.Fatalf("outfit item context = %q, want user_wardrobe", got)
	}
}

func TestResolveSystemCatalogReusesOwnedRow(t *testing.T) {
	fashionID := uuid.New()
	owned := &entities.WardrobeItem{FashionItemID: &fashionID, Status: wardrobestatus.InWardrobe}
	owned.ID = uuid.New()
	catalog := &entities.WardrobeItem{FashionItemID: &fashionID, ItemType: itemtype.SystemCatalogItem, Status: wardrobestatus.InWardrobe}
	catalog.ID = uuid.New()

	uc := newResolveUC(
		&fakeWardrobeRepo{owned: []*entities.WardrobeItem{owned}, catalog: []*entities.WardrobeItem{catalog}},
		&fakeBrandContract{eligible: map[uuid.UUID]bool{}},
	)
	ctxStr := itemContextSystemCatalog

	resolution, err := uc.resolveOutfitItems(context.Background(), uuid.New(), uuid.Nil, []dto.SaveOutfitItemReq{
		{FashionItemID: fashionID, ItemContext: &ctxStr, Scale: 1, LayerOrder: 1},
	}, 10, map[uuid.UUID]bool{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolution.cloneItems) != 0 {
		t.Fatalf("clone items = %d, want 0 (reuse owned row)", len(resolution.cloneItems))
	}
	if len(resolution.reuseItemIDs) != 1 || resolution.reuseItemIDs[0] != owned.ID {
		t.Fatalf("reuse ids = %v, want [%v]", resolution.reuseItemIDs, owned.ID)
	}
	if got := resolution.items[0].ItemContext; got != outfititemcontext.UserWardrobe {
		t.Fatalf("outfit item context = %q, want user_wardrobe", got)
	}
}

func TestResolveExplicitUserWardrobeRequiresOwnership(t *testing.T) {
	ctxStr := itemContextUserWardrobe
	uc := newResolveUC(
		&fakeWardrobeRepo{},
		&fakeBrandContract{eligible: map[uuid.UUID]bool{}},
	)

	_, err := uc.resolveOutfitItems(context.Background(), uuid.New(), uuid.Nil, []dto.SaveOutfitItemReq{
		{FashionItemID: uuid.New(), ItemContext: &ctxStr, Scale: 1, LayerOrder: 1},
	}, 10, map[uuid.UUID]bool{})
	if err == nil {
		t.Fatal("expected forbidden/unavailable error")
	}
}

func TestResolveExplicitSystemCatalogRejectsUnavailableItem(t *testing.T) {
	ctxStr := itemContextSystemCatalog
	uc := newResolveUC(
		&fakeWardrobeRepo{},
		&fakeBrandContract{eligible: map[uuid.UUID]bool{}},
	)

	_, err := uc.resolveOutfitItems(context.Background(), uuid.New(), uuid.Nil, []dto.SaveOutfitItemReq{
		{FashionItemID: uuid.New(), ItemContext: &ctxStr, Scale: 1, LayerOrder: 1},
	}, 10, map[uuid.UUID]bool{})
	if err == nil {
		t.Fatal("expected catalog unavailable error")
	}
	want := wardrobeerrors.ErrCatalogItemUnavailable()
	if err.Error() != want.Error() {
		t.Fatalf("error = %q, want %q", err.Error(), want.Error())
	}
}

func TestResolveBrandItemNeverClones(t *testing.T) {
	fashionID := uuid.New()
	ctxStr := itemContextBrandItem
	uc := newResolveUC(
		&fakeWardrobeRepo{},
		&fakeBrandContract{eligible: map[uuid.UUID]bool{fashionID: true}},
	)

	resolution, err := uc.resolveOutfitItems(context.Background(), uuid.New(), uuid.Nil, []dto.SaveOutfitItemReq{
		{FashionItemID: fashionID, ItemContext: &ctxStr, Scale: 1, LayerOrder: 1},
	}, 10, map[uuid.UUID]bool{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolution.cloneItems) != 0 {
		t.Fatalf("clone items = %d, want 0", len(resolution.cloneItems))
	}
	if got := resolution.items[0].ItemContext; got != outfititemcontext.BrandItem {
		t.Fatalf("outfit item context = %q, want brand_item", got)
	}
}

func TestResolveSoldOwnedRowIsRejected(t *testing.T) {
	fashionID := uuid.New()
	owned := &entities.WardrobeItem{FashionItemID: &fashionID, Status: wardrobestatus.Sold}
	owned.ID = uuid.New()
	uc := newResolveUC(
		&fakeWardrobeRepo{owned: []*entities.WardrobeItem{owned}},
		&fakeBrandContract{eligible: map[uuid.UUID]bool{}},
	)

	_, err := uc.resolveOutfitItems(context.Background(), uuid.New(), uuid.Nil, []dto.SaveOutfitItemReq{
		{FashionItemID: fashionID, Scale: 1, LayerOrder: 1},
	}, 10, map[uuid.UUID]bool{})
	if err == nil {
		t.Fatal("expected sold item error")
	}
	want := wardrobeerrors.ErrOutfitItemSold(fashionID)
	if err.Error() != want.Error() {
		t.Fatalf("error = %q, want %q", err.Error(), want.Error())
	}
}

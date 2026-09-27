package item

import (
	"context"
	"testing"

	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type filterFashionMock struct {
	fashion_contract.IFashionContract
	searchFn func(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error)
}

func (m *filterFashionMock) SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error) {
	return m.searchFn(ctx, query, categorySlug)
}

type filterRepoMock struct {
	repositories.IWardrobeItemRepository
	countFn func(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filter bool) (int64, error)
	pageFn  func(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filter bool, p shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
}

func (m *filterRepoMock) CountByUserIDAndFilters(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filter bool) (int64, error) {
	return m.countFn(ctx, userID, ids, statuses, filter)
}

func (m *filterRepoMock) GetByUserIDAndFiltersPaginated(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filter bool, p shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	return m.pageFn(ctx, userID, ids, statuses, filter, p)
}

func (m *filterRepoMock) GetByUserID(context.Context, uuid.UUID, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return nil, nil
}

func TestGetWardrobeItems_EmptyCategoryReturnsZeroWithoutDB(t *testing.T) {
	repo := &filterRepoMock{
		countFn: func(context.Context, uuid.UUID, []uuid.UUID, []wardrobestatus.WardrobeItemStatus, bool) (int64, error) {
			t.Fatal("repo CountByUserIDAndFilters must not be called on short-circuit")
			return 0, nil
		},
		pageFn: func(context.Context, uuid.UUID, []uuid.UUID, []wardrobestatus.WardrobeItemStatus, bool, shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
			t.Fatal("repo GetByUserIDAndFiltersPaginated must not be called on short-circuit")
			return nil, nil
		},
	}
	fashion := &filterFashionMock{searchFn: func(context.Context, *string, *string) ([]uuid.UUID, error) {
		return []uuid.UUID{}, nil // category exists but matches nothing
	}}
	uc := &WardrobeItemUseCase{
		wardrobeRepo:    repo,
		fashionContract: fashion,
		userSubContract: &mockSubContract{},
		logger:          silentLogger{},
	}

	res, err := uc.GetWardrobeItems(context.Background(), uuid.New(), dto.GetWardrobeItemsQueryReq{
		CategorySlug: "empty-category",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res.Items) != 0 {
		t.Fatalf("expected empty result, got %+v", res)
	}
	if res.Metadata.TotalItems != 0 {
		t.Fatalf("expected total 0, got %d", res.Metadata.TotalItems)
	}
}

func TestGetWardrobeItems_NoFilterQueriesWithoutFashionFilter(t *testing.T) {
	var sawFilter bool
	repo := &filterRepoMock{
		countFn: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ []wardrobestatus.WardrobeItemStatus, filter bool) (int64, error) {
			sawFilter = filter
			return 0, nil
		},
		pageFn: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ []wardrobestatus.WardrobeItemStatus, filter bool, _ shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
			sawFilter = filter
			return nil, nil
		},
	}
	fashion := &filterFashionMock{searchFn: func(context.Context, *string, *string) ([]uuid.UUID, error) {
		t.Fatal("SearchFashionItemIDs must not be called without a category filter")
		return nil, nil
	}}
	uc := &WardrobeItemUseCase{
		wardrobeRepo:    repo,
		fashionContract: fashion,
		userSubContract: &mockSubContract{},
		logger:          silentLogger{},
	}

	if _, err := uc.GetWardrobeItems(context.Background(), uuid.New(), dto.GetWardrobeItemsQueryReq{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sawFilter {
		t.Fatal("expected filterByFashionItems=false when no category is given")
	}
}

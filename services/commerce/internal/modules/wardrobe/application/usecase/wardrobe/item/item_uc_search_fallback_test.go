package item

import (
	"context"
	"testing"

	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	wardrobe_search "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/infrastructure/search"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/config"
	shared_search "smart-wardrobe-be/shared/infrastructure/search"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// silentLogger satisfies logger.Interface without emitting anything.
type silentLogger struct{}

func (silentLogger) Debug(string, ...zap.Field) {}
func (silentLogger) Info(string, ...zap.Field)  {}
func (silentLogger) Warn(string, ...zap.Field)  {}
func (silentLogger) Error(string, ...zap.Field) {}
func (silentLogger) Fatal(string, ...zap.Field) {}

// fallbackRepoMock serves one system catalog item from the database path.
// The embedded interface satisfies the rest of IWardrobeItemRepository.
type fallbackRepoMock struct {
	repositories.IWardrobeItemRepository
	items []*entities.WardrobeItem
}

func (m *fallbackRepoMock) CountItems(ctx context.Context, fashionItemIDs []uuid.UUID, itemType itemtype.ItemType) (int64, error) {
	return int64(len(m.items)), nil
}

func (m *fallbackRepoMock) GetItemsPaginated(ctx context.Context, fashionItemIDs []uuid.UUID, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	return m.items, nil
}

// fallbackFashionMock resolves fashion lookups to empty (no fashion linkage).
type fallbackFashionMock struct {
	fashion_contract.IFashionContract
}

func (fallbackFashionMock) SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error) {
	return nil, nil
}

func (fallbackFashionMock) ListFashionItemsBrief(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.FashionItemBriefRes, error) {
	return nil, nil
}

// TestGetSystemCatalogFallbackWhenDisabled proves FR-002: with the engine
// switched off, the system catalog still returns database results instead of
// an error, and no sync failure is produced.
func TestGetSystemCatalogFallbackWhenDisabled(t *testing.T) {
	engine := shared_search.NewElasticsearchClient(&config.Config{}, nil)
	defer engine.Stop()
	searchSvc := wardrobe_search.NewWardrobeSearchService(engine)

	itemID := uuid.New()
	dbItem := &entities.WardrobeItem{}
	dbItem.ID = itemID
	repo := &fallbackRepoMock{items: []*entities.WardrobeItem{dbItem}}

	uc := NewWardrobeItemUseCase(
		&config.Config{},
		silentLogger{},
		repo,
		nil, // outfitRepo (unused on this path)
		searchSvc,
		nil, // mediaService (unused on this path)
		nil, // userSubContract (unused on this path)
		nil, // brandContract (unused on this path)
		nil, // eventPublisher (unused on this path)
		fallbackFashionMock{},
		nil, // categoryContract (unused on this path)
		nil, // uow (unused on this path)
	)

	res, err := uc.GetSystemCatalogWardrobeItems(context.Background(), dto.SearchWardrobeItemsQueryReq{
		PaginationQuery: shared_dto.PaginationQuery{Page: 1, Limit: 20},
		Query:           "linen shirt",
	})
	if err != nil {
		t.Fatalf("fallback must succeed when search is disabled, got %v", err)
	}
	if res == nil || len(res.Items) != 1 {
		t.Fatalf("want 1 database item, got %+v", res)
	}
	if res.Items[0] == nil || res.Items[0].ID != itemID {
		t.Fatalf("wrong item returned: %+v", res.Items[0])
	}
	if res.Metadata.TotalItems != 1 {
		t.Fatalf("TotalItems = %d, want 1", res.Metadata.TotalItems)
	}
}

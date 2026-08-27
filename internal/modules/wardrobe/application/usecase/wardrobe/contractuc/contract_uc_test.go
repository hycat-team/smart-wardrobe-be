package contractuc_test

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/contractuc"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

type mockWardrobeItemRepo struct {
	repositories.IWardrobeItemRepository
	getByUserIDFn func(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) ([]*entities.WardrobeItem, error)
}

func (m *mockWardrobeItemRepo) GetByUserID(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return m.getByUserIDFn(ctx, userID, itemIDs)
}

func TestGetWardrobeItemsByUserID_ReturnsPureItems(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()

	repo := &mockWardrobeItemRepo{
		getByUserIDFn: func(ctx context.Context, uID uuid.UUID, itemIDs []uuid.UUID) ([]*entities.WardrobeItem, error) {
			item := &entities.WardrobeItem{
				UserID:        userID,
				FashionItemID: &fashionID,
			}
			item.ID = uuid.New()
			return []*entities.WardrobeItem{item}, nil
		},
	}

	uc := contractuc.NewWardrobeContractUseCase(repo)
	res, err := uc.GetWardrobeItemsByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res))
	}

	if res[0].FashionItemID == nil || *res[0].FashionItemID != fashionID {
		t.Errorf("expected FashionItemID to be %v, got %v", fashionID, res[0].FashionItemID)
	}

	if res[0].FashionItem != nil {
		t.Errorf("expected FashionItem to be nil (pure contract), got %v", res[0].FashionItem)
	}
}

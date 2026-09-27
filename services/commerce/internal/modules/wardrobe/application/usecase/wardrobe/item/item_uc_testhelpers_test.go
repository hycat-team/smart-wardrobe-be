package item

import (
	"context"
	"testing"
	"time"

	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/shared/application/event"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type confirmRepoMock struct {
	repositories.IWardrobeItemRepository
	getByIDFn func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error)
	claimFn   func(ctx context.Context, userID, itemID uuid.UUID, now time.Time) (*entities.WardrobeItem, bool, error)
	updateFn  func(ctx context.Context, item *entities.WardrobeItem) error
}

func (m *confirmRepoMock) GetByID(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
	return m.getByIDFn(ctx, id)
}

func (m *confirmRepoMock) ClaimManualAnalysisRetry(ctx context.Context, userID, itemID uuid.UUID, now time.Time) (*entities.WardrobeItem, bool, error) {
	return m.claimFn(ctx, userID, itemID, now)
}

func (m *confirmRepoMock) Update(ctx context.Context, item *entities.WardrobeItem) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, item)
	}
	return nil
}

type confirmFashionMock struct {
	fashion_contract.IFashionContract
	getFashionItemFn       func(ctx context.Context, id uuid.UUID) (*fashion_dto.FashionItemContractDTO, error)
	isWearableCategoryFn   func(ctx context.Context, id uuid.UUID) (bool, error)
	resetProcessingStateFn func(ctx context.Context, id uuid.UUID) error
}

func (m *confirmFashionMock) GetFashionItem(ctx context.Context, id uuid.UUID) (*fashion_dto.FashionItemContractDTO, error) {
	return m.getFashionItemFn(ctx, id)
}

func (m *confirmFashionMock) IsWearableCategory(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.isWearableCategoryFn(ctx, id)
}

func (m *confirmFashionMock) ResetProcessingState(ctx context.Context, id uuid.UUID) error {
	if m.resetProcessingStateFn != nil {
		return m.resetProcessingStateFn(ctx, id)
	}
	return nil
}

type directUoW struct{}

func (directUoW) Execute(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type noopPublisher struct{ published int }

func (p *noopPublisher) Publish(context.Context, string, any) error { p.published++; return nil }

var (
	_ shared_repos.IUnitOfWork = directUoW{}
	_ event.IEventPublisher    = (*noopPublisher)(nil)
)

func assertSameErr(t *testing.T, got, want error) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected error %q, got nil", want.Error())
	}
	if got.Error() != want.Error() {
		t.Fatalf("expected error %q, got %q", want.Error(), got.Error())
	}
}

func wardItem(userID uuid.UUID, status wardrobestatus.WardrobeItemStatus, fashionItemID *uuid.UUID) *entities.WardrobeItem {
	item := &entities.WardrobeItem{
		UserID:        userID,
		Status:        status,
		ItemType:      itemtype.UserItem,
		FashionItemID: fashionItemID,
	}
	item.ID = uuid.New()
	return item
}

package worker

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/internal/shared/application/event"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type mockFashionItemRepoForWorker struct {
	repositories.IFashionItemRepository
	completeSuccessFn func(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
}

func (m *mockFashionItemRepoForWorker) CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	return m.completeSuccessFn(ctx, itemID, processingVersion, updates)
}

type mockWardrobeContractForWorker struct {
	wardrobe_contract.IWardrobeContract
	updateItemStatusFn func(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error
	getItemsByIDsFn    func(ctx context.Context, itemIDs []uuid.UUID) ([]*wardrobeDto.WardrobeItemRes, error)
}

func (m *mockWardrobeContractForWorker) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	return m.updateItemStatusFn(ctx, itemID, status)
}

func (m *mockWardrobeContractForWorker) GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*wardrobeDto.WardrobeItemRes, error) {
	return m.getItemsByIDsFn(ctx, itemIDs)
}

type mockEventPublisherForWorker struct {
	event.IEventPublisher
	publishFn func(ctx context.Context, topic string, payload any) error
}

func (m *mockEventPublisherForWorker) Publish(ctx context.Context, topic string, payload any) error {
	return m.publishFn(ctx, topic, payload)
}

type mockUOWForWorker struct {
	shared_repos.IUnitOfWork
	executeFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockUOWForWorker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.executeFn(ctx, fn)
}

// TestCompleteProcessedItemCallsBothSides verifies the wardrobe-item completion path:
// fashion module updates fashion_items via its own repo, and wardrobe_items status
// changes only through the wardrobe contract (no cross-module raw access).
func TestCompleteProcessedItemCallsBothSides(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()

	fashionRepoCalled := false
	wardrobeStatusUpdated := false

	uc := &FashionWorkerUseCase{
		fashionItemRepo: &mockFashionItemRepoForWorker{
			completeSuccessFn: func(ctx context.Context, id uuid.UUID, version int, updates map[string]any) (bool, error) {
				if id != fashionItemID {
					t.Fatalf("expected fashion item id %s, got %s", fashionItemID, id)
				}
				if version != 3 {
					t.Fatalf("expected processing version 3, got %d", version)
				}
				if updates["embedding"] == nil {
					t.Fatalf("expected embedding in fashion updates, got %v", updates)
				}
				fashionRepoCalled = true
				return true, nil
			},
		},
		wardrobeContract: &mockWardrobeContractForWorker{
			updateItemStatusFn: func(ctx context.Context, id uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
				if id != itemID {
					t.Fatalf("expected wardrobe item id %s, got %s", itemID, id)
				}
				if status != wardrobestatus.InWardrobe {
					t.Fatalf("expected InWardrobe status, got %v", status)
				}
				wardrobeStatusUpdated = true
				return nil
			},
			getItemsByIDsFn: func(ctx context.Context, ids []uuid.UUID) ([]*wardrobeDto.WardrobeItemRes, error) {
				return []*wardrobeDto.WardrobeItemRes{{ID: itemID}}, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
	}

	job := dto.FashionAnalyzeJobDTO{
		ItemID:            itemID,
		FashionItemID:     &fashionItemID,
		ItemType:          "wardrobe",
		UserID:            uuid.New(),
		ProcessingVersion: 3,
	}

	err := uc.completeProcessedItem(context.Background(), job, map[string]any{"embedding": []float32{1.0, 2.0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fashionRepoCalled {
		t.Fatal("expected fashionItemRepo.CompleteProcessingSuccess to be called")
	}
	if !wardrobeStatusUpdated {
		t.Fatal("expected wardrobeContract.UpdateItemStatus to be called")
	}
}

// TestCompleteProcessedItemSkipsFashionWhenNoFashionItemID verifies that when a
// wardrobe item has no fashion_item_id, the fashion update is skipped but the
// wardrobe status is still updated.
func TestCompleteProcessedItemSkipsFashionWhenNoFashionItemID(t *testing.T) {
	itemID := uuid.New()

	fashionRepoCalled := false
	wardrobeStatusUpdated := false

	uc := &FashionWorkerUseCase{
		fashionItemRepo: &mockFashionItemRepoForWorker{
			completeSuccessFn: func(ctx context.Context, id uuid.UUID, version int, updates map[string]any) (bool, error) {
				fashionRepoCalled = true
				return true, nil
			},
		},
		wardrobeContract: &mockWardrobeContractForWorker{
			updateItemStatusFn: func(ctx context.Context, id uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
				wardrobeStatusUpdated = true
				return nil
			},
			getItemsByIDsFn: func(ctx context.Context, ids []uuid.UUID) ([]*wardrobeDto.WardrobeItemRes, error) {
				return []*wardrobeDto.WardrobeItemRes{{ID: itemID}}, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
	}

	job := dto.FashionAnalyzeJobDTO{
		ItemID:            itemID,
		ItemType:          "wardrobe",
		UserID:            uuid.New(),
		ProcessingVersion: 1,
	}

	err := uc.completeProcessedItem(context.Background(), job, map[string]any{"embedding": []float32{1.0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fashionRepoCalled {
		t.Fatal("expected fashionItemRepo NOT to be called when fashion_item_id is nil")
	}
	if !wardrobeStatusUpdated {
		t.Fatal("expected wardrobeContract.UpdateItemStatus to be called")
	}
}

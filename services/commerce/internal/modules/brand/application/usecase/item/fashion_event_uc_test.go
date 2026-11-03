package item

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/shared/application/constants/analyzetaskstatus"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockBrandItemRepoForEvent struct {
	repositories.IBrandItemRepository
	getByIDFn func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error)
	updateFn  func(ctx context.Context, item *entities.BrandItem) error
}

func (m *mockBrandItemRepoForEvent) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockBrandItemRepoForEvent) Update(ctx context.Context, item *entities.BrandItem) error {
	return m.updateFn(ctx, item)
}

type mockBrandSSEManagerForEvent struct {
	publishedTaskID string
	publishedBytes  []byte
}

func (m *mockBrandSSEManagerForEvent) Publish(_ context.Context, taskID string, payload []byte) error {
	m.publishedTaskID = taskID
	m.publishedBytes = payload
	return nil
}

func (m *mockBrandSSEManagerForEvent) Subscribe(taskID string) chan []byte {
	return make(chan []byte, 1)
}

func (m *mockBrandSSEManagerForEvent) Unsubscribe(taskID string, ch chan []byte) {}

type mockBrandLoggerForEvent struct{}

func (m *mockBrandLoggerForEvent) Debug(msg string, fields ...zap.Field) {}
func (m *mockBrandLoggerForEvent) Info(msg string, fields ...zap.Field)  {}
func (m *mockBrandLoggerForEvent) Warn(msg string, fields ...zap.Field)  {}
func (m *mockBrandLoggerForEvent) Error(msg string, fields ...zap.Field) {}
func (m *mockBrandLoggerForEvent) Fatal(msg string, fields ...zap.Field) {}
func (m *mockBrandLoggerForEvent) With(fields ...zap.Field) interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
} {
	return m
}

func TestBrandHandleFashionAnalyzed_Completed(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	taskID := taskUUID.String()
	item := &entities.BrandItem{
		Name:     "Brand Shirt",
		ItemType: branditemtype.Product,
		Status:   branditemstatus.Draft,
		TaskID:   &taskUUID,
	}
	item.ID = itemID

	updateCalled := false
	repo := &mockBrandItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.BrandItem) error {
			updateCalled = true
			if updated.Status != branditemstatus.Active {
				t.Fatalf("expected status Active, got %v", updated.Status)
			}
			if updated.TaskID != nil {
				t.Fatalf("expected TaskID cleared on terminal analysis event")
			}
			return nil
		},
	}
	sseMgr := &mockBrandSSEManagerForEvent{}
	logger := &mockBrandLoggerForEvent{}

	uc := NewBrandFashionEventUseCase(repo, sseMgr, logger)

	payload := fashion_dto.FashionAnalyzedEventPayload{
		TaskID: taskID,
		ItemID: itemID,
		Status: analyzetaskstatus.Completed,
		Total:  1,
		Index:  1,
		FashionBrief: &fashion_dto.FashionItemBriefRes{
			ID: uuid.New(),
		},
	}

	err := uc.HandleFashionAnalyzed(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected repo.Update to be called")
	}
	if sseMgr.publishedTaskID != taskID {
		t.Fatalf("expected SSE published to task %s, got %s", taskID, sseMgr.publishedTaskID)
	}
}

func TestBrandHandleFashionAnalyzed_Idempotent(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	taskID := taskUUID.String()
	item := &entities.BrandItem{
		Name:     "Brand Shirt",
		ItemType: branditemtype.Product,
		Status:   branditemstatus.Active,
	}
	item.ID = itemID

	updateCalled := false
	repo := &mockBrandItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.BrandItem) error {
			updateCalled = true
			return nil
		},
	}
	sseMgr := &mockBrandSSEManagerForEvent{}
	logger := &mockBrandLoggerForEvent{}

	uc := NewBrandFashionEventUseCase(repo, sseMgr, logger)

	payload := fashion_dto.FashionAnalyzedEventPayload{
		TaskID: taskID,
		ItemID: itemID,
		Status: analyzetaskstatus.Completed,
	}

	err := uc.HandleFashionAnalyzed(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateCalled {
		t.Fatal("expected repo.Update NOT to be called when item is already Active")
	}
	if sseMgr.publishedTaskID != taskID {
		t.Fatal("expected SSE to still be published even when DB update is skipped")
	}
}

func TestBrandHandleFashionFailed(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	taskID := taskUUID.String()
	item := &entities.BrandItem{
		Name:     "Brand Shirt",
		ItemType: branditemtype.Product,
		Status:   branditemstatus.Draft,
		TaskID:   &taskUUID,
	}
	item.ID = itemID

	repo := &mockBrandItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.BrandItem) error {
			return nil
		},
	}
	sseMgr := &mockBrandSSEManagerForEvent{}
	logger := &mockBrandLoggerForEvent{}

	uc := NewBrandFashionEventUseCase(repo, sseMgr, logger)

	payload := fashion_dto.FashionAnalyzedEventPayload{
		TaskID: taskID,
		ItemID: itemID,
		Status: analyzetaskstatus.Failed,
		Error:  "Failed to analyze vision",
	}

	err := uc.HandleFashionFailed(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sseMgr.publishedTaskID != taskID {
		t.Fatalf("expected SSE published to task %s, got %s", taskID, sseMgr.publishedTaskID)
	}
}

func TestBrandHandleFashionItemCreated_SetsID(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.BrandItem{Name: "Brand Shirt", ItemType: branditemtype.Product, Status: branditemstatus.Draft}
	item.ID = itemID
	updateCalled := false
	repo := &mockBrandItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.BrandItem) error {
			updateCalled = true
			if updated.FashionItemID == nil || *updated.FashionItemID != fashionItemID {
				t.Fatalf("expected FashionItemID %s", fashionItemID)
			}
			return nil
		},
	}
	uc := NewBrandFashionEventUseCase(repo, &mockBrandSSEManagerForEvent{}, &mockBrandLoggerForEvent{})
	payload := fashion_dto.FashionItemCreatedEventPayload{ItemID: itemID, FashionItemID: fashionItemID, ItemType: "brand"}
	if err := uc.HandleFashionItemCreated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected Update to be called")
	}
}

func TestBrandHandleFashionItemCreated_Idempotent(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.BrandItem{Name: "Brand Shirt", ItemType: branditemtype.Product, Status: branditemstatus.Draft, FashionItemID: &fashionItemID}
	item.ID = itemID
	repo := &mockBrandItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.BrandItem) error {
			t.Fatal("expected no Update when already set")
			return nil
		},
	}
	uc := NewBrandFashionEventUseCase(repo, &mockBrandSSEManagerForEvent{}, &mockBrandLoggerForEvent{})
	payload := fashion_dto.FashionItemCreatedEventPayload{ItemID: itemID, FashionItemID: fashionItemID, ItemType: "brand"}
	if err := uc.HandleFashionItemCreated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrandHandleFashionProcessingExhausted_ClearsTaskID(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	item := &entities.BrandItem{
		Name:     "Brand Shirt",
		ItemType: branditemtype.Product,
		Status:   branditemstatus.Draft,
		TaskID:   &taskUUID,
	}
	item.ID = itemID

	var updated *entities.BrandItem
	repo := &mockBrandItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, next *entities.BrandItem) error {
			updated = next
			return nil
		},
	}

	uc := NewBrandFashionEventUseCase(repo, &mockBrandSSEManagerForEvent{}, &mockBrandLoggerForEvent{})
	if err := uc.HandleFashionProcessingExhausted(context.Background(), fashion_dto.FashionProcessingExhaustedEventPayload{
		ItemID: itemID,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected item update")
	}
	if updated.Status != branditemstatus.Failed {
		t.Fatalf("expected failed status, got %s", updated.Status)
	}
	if updated.TaskID != nil {
		t.Fatal("expected task id cleared for exhausted analysis")
	}
}

package wardrobe

import (
	"context"
	"testing"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockWardrobeItemRepoForEvent struct {
	repositories.IWardrobeItemRepository
	getByIDFn func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error)
	updateFn  func(ctx context.Context, item *entities.WardrobeItem) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockWardrobeItemRepoForEvent) GetByID(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockWardrobeItemRepoForEvent) Update(ctx context.Context, item *entities.WardrobeItem) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, item)
	}
	return nil
}

func (m *mockWardrobeItemRepoForEvent) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockSSEManagerForEvent struct {
	publishedTaskID string
	publishedBytes  []byte
}

func (m *mockSSEManagerForEvent) Publish(taskID string, payload []byte) {
	m.publishedTaskID = taskID
	m.publishedBytes = payload
}

func (m *mockSSEManagerForEvent) Subscribe(taskID string) chan []byte {
	return make(chan []byte, 1)
}

func (m *mockSSEManagerForEvent) Unsubscribe(taskID string, ch chan []byte) {}

type mockEventPublisherForEvent struct {
	event.IEventPublisher
	publishedTopic string
}

func (m *mockEventPublisherForEvent) Publish(ctx context.Context, topic string, payload any) error {
	m.publishedTopic = topic
	return nil
}

type mockUOWForEvent struct {
	shared_repos.IUnitOfWork
}

func (m *mockUOWForEvent) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type mockLoggerForEvent struct{}

func (m *mockLoggerForEvent) Debug(msg string, fields ...zap.Field) {}
func (m *mockLoggerForEvent) Info(msg string, fields ...zap.Field)  {}
func (m *mockLoggerForEvent) Warn(msg string, fields ...zap.Field)  {}
func (m *mockLoggerForEvent) Error(msg string, fields ...zap.Field) {}
func (m *mockLoggerForEvent) Fatal(msg string, fields ...zap.Field) {}
func (m *mockLoggerForEvent) With(fields ...zap.Field) interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
} {
	return m
}

func TestHandleFashionAnalyzed_Completed(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	taskID := taskUUID.String()
	item := &entities.WardrobeItem{
		Status: wardrobestatus.Processing,
		TaskID: &taskUUID,
	}
	item.ID = itemID

	updateCalled := false
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			updateCalled = true
			if updated.Status != wardrobestatus.InWardrobe {
				t.Fatalf("expected status InWardrobe, got %v", updated.Status)
			}
			return nil
		},
	}
	sseMgr := &mockSSEManagerForEvent{}
	pub := &mockEventPublisherForEvent{}
	uow := &mockUOWForEvent{}
	logger := &mockLoggerForEvent{}

	uc := NewWardrobeFashionEventUseCase(repo, sseMgr, pub, uow, logger)

	payload := fashionDto.FashionAnalyzedEventPayload{
		TaskID: taskID,
		ItemID: itemID,
		Status: analyzetaskstatus.Completed,
		Total:  1,
		Index:  1,
		FashionBrief: &fashionDto.FashionItemBriefRes{
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
	if pub.publishedTopic != eventconstants.TopicWardrobeCreated {
		t.Fatalf("expected TopicWardrobeCreated published, got %s", pub.publishedTopic)
	}
}

func TestHandleFashionAnalyzed_IdempotentSkipsUpdate(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	taskID := taskUUID.String()
	item := &entities.WardrobeItem{
		Status: wardrobestatus.InWardrobe,
		TaskID: &taskUUID,
	}
	item.ID = itemID

	updateCalled := false
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			updateCalled = true
			return nil
		},
	}
	sseMgr := &mockSSEManagerForEvent{}
	pub := &mockEventPublisherForEvent{}
	uow := &mockUOWForEvent{}
	logger := &mockLoggerForEvent{}

	uc := NewWardrobeFashionEventUseCase(repo, sseMgr, pub, uow, logger)

	payload := fashionDto.FashionAnalyzedEventPayload{
		TaskID: taskID,
		ItemID: itemID,
		Status: analyzetaskstatus.Completed,
	}

	err := uc.HandleFashionAnalyzed(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateCalled {
		t.Fatal("expected repo.Update NOT to be called when status is already InWardrobe (idempotency)")
	}
	if sseMgr.publishedTaskID != taskID {
		t.Fatal("expected SSE to still be published even when DB update is skipped")
	}
}

func TestHandleFashionFailed(t *testing.T) {
	itemID := uuid.New()
	taskUUID := uuid.New()
	taskID := taskUUID.String()
	item := &entities.WardrobeItem{
		Status: wardrobestatus.Processing,
		TaskID: &taskUUID,
	}
	item.ID = itemID

	updateCalled := false
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			updateCalled = true
			if updated.Status != wardrobestatus.Failed {
				t.Fatalf("expected status Failed, got %v", updated.Status)
			}
			return nil
		},
	}
	sseMgr := &mockSSEManagerForEvent{}
	pub := &mockEventPublisherForEvent{}
	uow := &mockUOWForEvent{}
	logger := &mockLoggerForEvent{}

	uc := NewWardrobeFashionEventUseCase(repo, sseMgr, pub, uow, logger)

	payload := fashionDto.FashionAnalyzedEventPayload{
		TaskID: taskID,
		ItemID: itemID,
		Status: analyzetaskstatus.Failed,
		Error:  "AI vision failure",
	}

	err := uc.HandleFashionFailed(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected repo.Update to be called with Failed status")
	}
	if sseMgr.publishedTaskID != taskID {
		t.Fatalf("expected SSE published to task %s, got %s", taskID, sseMgr.publishedTaskID)
	}
}

func TestHandleFashionItemCreated_SetsID(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.WardrobeItem{}
	item.ID = itemID
	updateCalled := false
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			updateCalled = true
			if updated.FashionItemID == nil || *updated.FashionItemID != fashionItemID {
				t.Fatalf("expected FashionItemID %s", fashionItemID)
			}
			return nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	payload := fashionDto.FashionItemCreatedEventPayload{ItemID: itemID, FashionItemID: fashionItemID, ItemType: "wardrobe"}
	if err := uc.HandleFashionItemCreated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected Update to be called")
	}
}

func TestHandleFashionItemCreated_Idempotent(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.WardrobeItem{FashionItemID: &fashionItemID}
	item.ID = itemID
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			t.Fatal("expected Update NOT to be called when already set (idempotent)")
			return nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	payload := fashionDto.FashionItemCreatedEventPayload{ItemID: itemID, FashionItemID: fashionItemID, ItemType: "wardrobe"}
	if err := uc.HandleFashionItemCreated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleFashionProcessingExhausted_MarksFailed(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.WardrobeItem{Status: wardrobestatus.Processing}
	item.ID = itemID
	updateCalled := false
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			updateCalled = true
			if updated.Status != wardrobestatus.Failed {
				t.Fatalf("expected Failed")
			}
			return nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	payload := fashionDto.FashionProcessingExhaustedEventPayload{WardrobeItemID: itemID, FashionItemID: fashionItemID, Reason: "exhausted"}
	if err := uc.HandleFashionProcessingExhausted(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected Update to be called")
	}
}

func TestHandleFashionProcessingExhausted_Idempotent(t *testing.T) {
	itemID := uuid.New()
	item := &entities.WardrobeItem{Status: wardrobestatus.Failed}
	item.ID = itemID
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		updateFn: func(ctx context.Context, updated *entities.WardrobeItem) error {
			t.Fatal("expected no Update when already Failed")
			return nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	payload := fashionDto.FashionProcessingExhaustedEventPayload{WardrobeItemID: itemID, FashionItemID: uuid.New(), Reason: "exhausted"}
	if err := uc.HandleFashionProcessingExhausted(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleWardrobeDeleted_Deletes(t *testing.T) {
	itemID := uuid.New()
	item := &entities.WardrobeItem{}
	item.ID = itemID
	deleteCalled := false
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleteCalled = true
			if id != itemID {
				t.Fatalf("wrong id")
			}
			return nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	payload := struct {
		ItemID uuid.UUID `json:"itemId"`
		UserID uuid.UUID `json:"userId"`
		Action string    `json:"action"`
	}{ItemID: itemID, Action: eventconstants.ActionDeleted}
	// Use real DTO type
	dtoPayload := structToWardrobePayload(payload.ItemID, payload.UserID, payload.Action)
	if err := uc.HandleWardrobeDeleted(context.Background(), dtoPayload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected Delete to be called")
	}
}

func TestHandleWardrobeDeleted_IdempotentIsDeleted(t *testing.T) {
	itemID := uuid.New()
	item := &entities.WardrobeItem{}
	item.ID = itemID
	item.SoftDeleteEntity.IsDeleted = true
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return item, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			t.Fatal("should not delete when already IsDeleted")
			return nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	dtoPayload := structToWardrobePayload(itemID, uuid.New(), eventconstants.ActionDeleted)
	if err := uc.HandleWardrobeDeleted(context.Background(), dtoPayload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleWardrobeDeleted_NilItem(t *testing.T) {
	repo := &mockWardrobeItemRepoForEvent{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
			return nil, nil
		},
	}
	uc := NewWardrobeFashionEventUseCase(repo, &mockSSEManagerForEvent{}, &mockEventPublisherForEvent{}, &mockUOWForEvent{}, &mockLoggerForEvent{})
	dtoPayload := structToWardrobePayload(uuid.New(), uuid.New(), eventconstants.ActionDeleted)
	if err := uc.HandleWardrobeDeleted(context.Background(), dtoPayload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func structToWardrobePayload(itemID, userID uuid.UUID, action string) dto.WardrobeEventPayload {
	// helper to avoid import cycle - uses real DTO
	return dto.WardrobeEventPayload{ItemID: itemID, UserID: userID, Action: action}
}

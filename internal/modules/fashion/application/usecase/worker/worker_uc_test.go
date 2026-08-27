package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockFashionItemRepoForWorker struct {
	repositories.IFashionItemRepository
	completeSuccessFn      func(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
	getByIDFn              func(ctx context.Context, itemID uuid.UUID) (*entities.FashionItem, error)
	getByIDsFn             func(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error)
	incrementRetryCountFn  func(ctx context.Context, itemID uuid.UUID) error
	markProcessingFailedFn func(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
}

func (m *mockFashionItemRepoForWorker) CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	return m.completeSuccessFn(ctx, itemID, processingVersion, updates)
}

func (m *mockFashionItemRepoForWorker) GetByID(ctx context.Context, itemID uuid.UUID) (*entities.FashionItem, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, itemID)
	}
	fi := &entities.FashionItem{
		ImageUrl: "https://example.com/item.jpg",
	}
	fi.ID = itemID
	return fi, nil
}

func (m *mockFashionItemRepoForWorker) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, ids)
	}
	if m.getByIDFn != nil {
		result := make([]*entities.FashionItem, 0, len(ids))
		for _, id := range ids {
			if fi, _ := m.getByIDFn(ctx, id); fi != nil {
				result = append(result, fi)
			}
		}
		return result, nil
	}
	result := make([]*entities.FashionItem, 0, len(ids))
	for _, id := range ids {
		fi := &entities.FashionItem{
			ImageUrl: "https://example.com/item.jpg",
		}
		fi.ID = id
		result = append(result, fi)
	}
	return result, nil
}

func (m *mockFashionItemRepoForWorker) IncrementProcessingRetryCount(ctx context.Context, itemID uuid.UUID) error {
	return m.incrementRetryCountFn(ctx, itemID)
}

func (m *mockFashionItemRepoForWorker) MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
	return m.markProcessingFailedFn(ctx, itemID, processingVersion, reason, reviewReason)
}

type mockWardrobeContractForWorker struct {
	wardrobe_contract.IWardrobeContract
	updateItemStatusFn         func(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error
	getItemsByIDsFn            func(ctx context.Context, itemIDs []uuid.UUID) ([]*wardrobeDto.WardrobeItemRes, error)
	getStaleProcessingItemsFn  func(ctx context.Context, staleBefore time.Time, limit int) ([]*wardrobeDto.WardrobeItemRes, error)
	getFailedItemsForCleanupFn func(ctx context.Context, limit int) ([]*wardrobeDto.WardrobeItemRes, error)
}

func (m *mockWardrobeContractForWorker) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	return m.updateItemStatusFn(ctx, itemID, status)
}

func (m *mockWardrobeContractForWorker) GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*wardrobeDto.WardrobeItemRes, error) {
	return m.getItemsByIDsFn(ctx, itemIDs)
}

func (m *mockWardrobeContractForWorker) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*wardrobeDto.WardrobeItemRes, error) {
	return m.getStaleProcessingItemsFn(ctx, staleBefore, limit)
}

func (m *mockWardrobeContractForWorker) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*wardrobeDto.WardrobeItemRes, error) {
	if m.getFailedItemsForCleanupFn != nil {
		return m.getFailedItemsForCleanupFn(ctx, limit)
	}
	return nil, nil
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

// TestCompleteProcessedItemPublishesEvent verifies that when an item completes analysis,
// FashionWorkerUseCase updates fashion_items via its repo and publishes TopicFashionAnalyzed event.
func TestCompleteProcessedItemPublishesEvent(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()

	fashionRepoCalled := false
	eventPublished := false

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
			getByIDFn: func(ctx context.Context, itemID uuid.UUID) (*entities.FashionItem, error) {
				return &entities.FashionItem{
					AuditableEntity: entities.AuditableEntity{
						BaseEntity: entities.BaseEntity{ID: fashionItemID},
					},
					ImageUrl: "https://example.com/item.jpg",
				}, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				if topic == eventconstants.TopicFashionAnalyzed {
					eventPublished = true
				}
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
	if !eventPublished {
		t.Fatal("expected TopicFashionAnalyzed event to be published")
	}
}

// TestRecoverStaleProcessingItemsRepublishes verifies that stale processing items
// below the retry cap are republished and have their retry count incremented.
func TestRecoverStaleProcessingItemsRepublishes(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	userID := uuid.New()

	publishCalls := 0
	uc := &FashionWorkerUseCase{
		cfg: &config.Config{Wardrobe: config.WardrobeProcessing{MaxRetryCount: 3}},
		wardrobeContract: &mockWardrobeContractForWorker{
			getStaleProcessingItemsFn: func(ctx context.Context, staleBefore time.Time, limit int) ([]*wardrobeDto.WardrobeItemRes, error) {
				return []*wardrobeDto.WardrobeItemRes{{
					ID:            itemID,
					UserID:        userID,
					FashionItem:   &wardrobeDto.FashionItemRes{ID: fashionItemID},
					FashionItemID: &fashionItemID,
				}}, nil
			},
		},
		fashionItemRepo: &mockFashionItemRepoForWorker{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.FashionItem, error) {
				return newFashionItemForTest(id, 2, 1), nil
			},
			incrementRetryCountFn: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				if topic != eventconstants.TopicFashionAnalyze {
					t.Fatalf("expected topic %s, got %s", eventconstants.TopicFashionAnalyze, topic)
				}
				publishCalls++
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
		logger: &mockLogger{},
	}

	run := workerlog.New("recovery", "test")
	if err := uc.RecoverStaleProcessingItems(context.Background(), run); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publishCalls != 1 {
		t.Fatalf("expected 1 republish, got %d", publishCalls)
	}
	counters := run.Counters()
	if counters.SuccessCount != 1 {
		t.Fatalf("expected 1 success, got %d", counters.SuccessCount)
	}
}

// TestRecoverStaleProcessingItemsMarksFailed verifies that stale processing items
// at or above the retry cap are marked failed and publish FashionProcessingExhausted instead of republished.
func TestRecoverStaleProcessingItemsMarksFailed(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	userID := uuid.New()

	markedFailed := false
	publishCalls := 0
	var publishedTopic string
	uc := &FashionWorkerUseCase{
		cfg: &config.Config{Wardrobe: config.WardrobeProcessing{MaxRetryCount: 3}},
		wardrobeContract: &mockWardrobeContractForWorker{
			getStaleProcessingItemsFn: func(ctx context.Context, staleBefore time.Time, limit int) ([]*wardrobeDto.WardrobeItemRes, error) {
				return []*wardrobeDto.WardrobeItemRes{{
					ID:            itemID,
					UserID:        userID,
					FashionItem:   &wardrobeDto.FashionItemRes{ID: fashionItemID},
					FashionItemID: &fashionItemID,
				}}, nil
			},
		},
		fashionItemRepo: &mockFashionItemRepoForWorker{
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.FashionItem, error) {
				return newFashionItemForTest(id, 2, 3), nil
			},
			markProcessingFailedFn: func(ctx context.Context, id uuid.UUID, version int, reason string, reviewReason *string) (bool, error) {
				markedFailed = true
				return true, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				publishedTopic = topic
				publishCalls++
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
		logger: &mockLogger{},
	}

	run := workerlog.New("recovery", "test")
	if err := uc.RecoverStaleProcessingItems(context.Background(), run); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !markedFailed {
		t.Fatal("expected MarkProcessingFailed to be called")
	}
	if publishCalls != 1 {
		t.Fatalf("expected 1 publish for exhausted retries (FashionProcessingExhausted), got %d", publishCalls)
	}
	if publishedTopic != eventconstants.TopicFashionProcessingExhausted {
		t.Fatalf("expected topic %s, got %s", eventconstants.TopicFashionProcessingExhausted, publishedTopic)
	}
}

func TestCleanupFailedItemsPublishes(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()
	userID := uuid.New()
	publishCalls := 0
	var publishedTopic string
	uc := &FashionWorkerUseCase{
		wardrobeContract: &mockWardrobeContractForWorker{
			getFailedItemsForCleanupFn: func(ctx context.Context, limit int) ([]*wardrobeDto.WardrobeItemRes, error) {
				if limit == 100 {
					return []*wardrobeDto.WardrobeItemRes{{
						ID:            itemID,
						UserID:        userID,
						FashionItemID: &fashionItemID,
					}}, nil
				}
				return nil, nil
			},
		},
		fashionItemRepo: &mockFashionItemRepoForWorker{
			getByIDsFn: func(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
				return nil, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				publishedTopic = topic
				publishCalls++
				if topic != eventconstants.TopicWardrobeDeleted {
					t.Fatalf("expected WardrobeDeleted, got %s", topic)
				}
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
		mediaService: &mockMediaServiceForWorker{},
		logger:       &mockLogger{},
	}
	run := workerlog.New("cleanup", "test")
	if err := uc.CleanupFailedItems(context.Background(), run); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publishCalls != 1 {
		t.Fatalf("expected 1 publish WardrobeDeleted, got %d", publishCalls)
	}
	if publishedTopic != eventconstants.TopicWardrobeDeleted {
		t.Fatalf("expected TopicWardrobeDeleted, got %s", publishedTopic)
	}
}

type mockMediaServiceForWorker struct{}

func (m *mockMediaServiceForWorker) DeleteImage(ctx context.Context, publicID string) error {
	return nil
}
func (m *mockMediaServiceForWorker) GenerateUploadSignature(ctx context.Context, params shared_dto.UploadSignatureParams) (*shared_dto.UploadSignatureResult, error) {
	return nil, nil
}

// TestRaceConditionClaimSingleWinner verifies SC-007 race-condition behaviour:
// two workers claiming the same item — only one wins because the version check
// on the fashion side rejects the second update.
func TestRaceConditionClaimSingleWinner(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()

	var mu sync.Mutex
	version := 5
	completed := false
	winCount := 0

	uc := &FashionWorkerUseCase{
		fashionItemRepo: &mockFashionItemRepoForWorker{
			completeSuccessFn: func(ctx context.Context, id uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
				// Emulate the DB-level atomic claim: WHERE processing_version = ? succeeds
				// only once; the second concurrent worker sees a stale version and loses.
				mu.Lock()
				defer mu.Unlock()
				if completed || processingVersion != version {
					return false, nil
				}
				completed = true
				version++
				winCount++
				return true, nil
			},
		},
		wardrobeContract: &mockWardrobeContractForWorker{
			updateItemStatusFn: func(ctx context.Context, id uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
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
		ProcessingVersion: version,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_ = uc.completeProcessedItem(context.Background(), job, map[string]any{"embedding": []float32{1.0}})
		}()
	}
	wg.Wait()

	if winCount != 1 {
		t.Fatalf("expected exactly 1 winner under race condition, got %d", winCount)
	}
}

// TestClaimRetryLatencyWithinBudget verifies SC-007: the claim-retry completion path
// completes well within the +100ms budget allowed after the refactor.
func TestClaimRetryLatencyWithinBudget(t *testing.T) {
	itemID := uuid.New()
	fashionItemID := uuid.New()

	uc := &FashionWorkerUseCase{
		fashionItemRepo: &mockFashionItemRepoForWorker{
			completeSuccessFn: func(ctx context.Context, id uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
				return true, nil
			},
		},
		wardrobeContract: &mockWardrobeContractForWorker{
			updateItemStatusFn: func(ctx context.Context, id uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
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
		ProcessingVersion: 1,
	}

	const budget = 100 * time.Millisecond
	const iterations = 50
	start := time.Now()
	for i := 0; i < iterations; i++ {
		if err := uc.completeProcessedItem(context.Background(), job, map[string]any{"embedding": []float32{1.0}}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)
	avg := elapsed / iterations
	if avg > budget {
		t.Fatalf("average claim-retry latency %v exceeds +100ms budget (SC-007)", avg)
	}
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...zap.Field) {}
func (m *mockLogger) Info(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Error(msg string, fields ...zap.Field) {}
func (m *mockLogger) Fatal(msg string, fields ...zap.Field) {}
func (m *mockLogger) With(fields ...zap.Field) logger.Interface {
	return m
}

func newFashionItemForTest(id uuid.UUID, version int, retryCount int) *entities.FashionItem {
	fi := &entities.FashionItem{ProcessingVersion: version, ProcessingRetryCount: retryCount}
	fi.ID = id
	return fi
}

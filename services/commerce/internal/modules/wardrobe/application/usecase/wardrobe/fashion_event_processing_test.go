package wardrobe

import (
	"context"
	"encoding/json"
	"testing"

	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/analyzetaskstatus"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type processingLogger struct{}

func (processingLogger) Debug(string, ...zap.Field) {}
func (processingLogger) Info(string, ...zap.Field)  {}
func (processingLogger) Warn(string, ...zap.Field)  {}
func (processingLogger) Error(string, ...zap.Field) {}
func (processingLogger) Fatal(string, ...zap.Field) {}

type processingRepoMock struct {
	repositories.IWardrobeItemRepository
	item *entities.WardrobeItem
}

func (m *processingRepoMock) GetByID(context.Context, uuid.UUID) (*entities.WardrobeItem, error) {
	return m.item, nil
}

type capturePublisher struct {
	calls  int
	taskID string
	data   []byte
}

func (p *capturePublisher) Publish(_ context.Context, taskId string, data []byte) error {
	p.calls++
	p.taskID = taskId
	p.data = data
	return nil
}

func TestWardrobeHandleFashionProcessing_EmitsIdempotentSSE(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()
	item := &entities.WardrobeItem{
		UserID:        userID,
		Status:        wardrobestatus.Processing,
		ItemType:      itemtype.UserItem,
		FashionItemID: &fashionID,
	}
	item.ID = uuid.New()

	pub := &capturePublisher{}
	uc := &WardrobeFashionEventUseCase{
		wardrobeItemRepo:  &processingRepoMock{item: item},
		realtimePublisher: pub,
		logger:            processingLogger{},
	}

	payload := fashion_dto.FashionAnalyzedEventPayload{
		TaskID:   "task-1",
		ItemID:   item.ID,
		ItemType: "wardrobe",
		Status:   analyzetaskstatus.Processing,
		Total:    3,
		Index:    1,
	}

	if err := uc.HandleFashionProcessing(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Idempotent: a duplicate delivery re-emits the SSE event only.
	if err := uc.HandleFashionProcessing(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error on duplicate: %v", err)
	}

	if pub.calls != 2 {
		t.Fatalf("expected 2 SSE publishes, got %d", pub.calls)
	}
	if pub.taskID != "task-1" {
		t.Fatalf("expected task-1, got %s", pub.taskID)
	}

	var evt fashion_dto.SSEEventPayload
	if err := json.Unmarshal(pub.data, &evt); err != nil {
		t.Fatalf("unmarshal SSE payload: %v", err)
	}
	if evt.Status != analyzetaskstatus.Processing {
		t.Fatalf("expected processing status, got %s", evt.Status)
	}
	if evt.Total != 3 || evt.Index != 1 {
		t.Fatalf("expected progress total/index 3/1, got %d/%d", evt.Total, evt.Index)
	}
}

func TestWardrobeHandleFashionEvent_DispatchesProcessing(t *testing.T) {
	userID := uuid.New()
	item := &entities.WardrobeItem{UserID: userID, Status: wardrobestatus.Processing, ItemType: itemtype.UserItem}
	item.ID = uuid.New()

	pub := &capturePublisher{}
	uc := &WardrobeFashionEventUseCase{
		wardrobeItemRepo:  &processingRepoMock{item: item},
		realtimePublisher: pub,
		logger:            processingLogger{},
	}

	if err := uc.HandleFashionEvent(context.Background(), fashion_dto.FashionAnalyzedEventPayload{
		TaskID: "t", ItemID: item.ID, Status: analyzetaskstatus.Processing,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub.calls != 1 {
		t.Fatalf("expected processing dispatch to emit once, got %d", pub.calls)
	}
}

// FR-005: terminal SSE payloads carry item id, progress index, summary data and
// (for failures) an error message.
func TestWardrobeTerminalSSEPayloadCompleteness(t *testing.T) {
	userID := uuid.New()
	cases := []struct {
		name        string
		status      wardrobestatus.WardrobeItemStatus
		payload     fashion_dto.FashionAnalyzedEventPayload
		wantStatus  analyzetaskstatus.AnalyzeTaskStatus
		wantError   string
		wantSummary bool
	}{
		{
			name:        "completed",
			status:      wardrobestatus.InWardrobe,
			payload:     fashion_dto.FashionAnalyzedEventPayload{Status: analyzetaskstatus.Completed},
			wantStatus:  analyzetaskstatus.Completed,
			wantSummary: true,
		},
		{
			name:        "needs_review",
			status:      wardrobestatus.NeedsReview,
			payload:     fashion_dto.FashionAnalyzedEventPayload{Status: analyzetaskstatus.NeedsReview},
			wantStatus:  analyzetaskstatus.NeedsReview,
			wantSummary: true,
		},
		{
			name:       "failed",
			status:     wardrobestatus.Failed,
			payload:    fashion_dto.FashionAnalyzedEventPayload{Status: analyzetaskstatus.Failed, Error: "phân tích thất bại"},
			wantStatus: analyzetaskstatus.Failed,
			wantError:  "phân tích thất bại",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := &entities.WardrobeItem{UserID: userID, Status: tc.status, ItemType: itemtype.UserItem}
			item.ID = uuid.New()
			pub := &capturePublisher{}
			uc := &WardrobeFashionEventUseCase{
				wardrobeItemRepo:  &processingRepoMock{item: item},
				realtimePublisher: pub,
				logger:            processingLogger{},
			}

			payload := tc.payload
			payload.TaskID = "task-1"
			payload.ItemID = item.ID
			payload.Total = 4
			payload.Index = 2

			if err := uc.HandleFashionEvent(context.Background(), payload); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pub.calls != 1 {
				t.Fatalf("expected 1 SSE publish, got %d", pub.calls)
			}

			var evt fashion_dto.SSEEventPayload
			if err := json.Unmarshal(pub.data, &evt); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if evt.ItemID != item.ID {
				t.Fatalf("itemId mismatch: %v", evt.ItemID)
			}
			if evt.Status != tc.wantStatus {
				t.Fatalf("status = %s, want %s", evt.Status, tc.wantStatus)
			}
			if evt.Total != 4 || evt.Index != 2 {
				t.Fatalf("progress total/index = %d/%d, want 4/2", evt.Total, evt.Index)
			}
			if tc.wantSummary && evt.Data == nil {
				t.Fatal("expected summary data in payload")
			}
			if tc.wantError != "" && evt.Error != tc.wantError {
				t.Fatalf("error = %q, want %q", evt.Error, tc.wantError)
			}
		})
	}
}

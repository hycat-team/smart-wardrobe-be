package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"

	aiDTO "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	aiContract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"

	brandDTO "smart-wardrobe-be/internal/modules/brand/application/dto"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"

	"github.com/google/uuid"
)

type mockDrillDownAIReader struct {
	aiContract.IAICostDashboardReader
	events []aiDTO.AIEventDrillDownDTO
	total  int64
	err    error
}

func (m *mockDrillDownAIReader) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]aiDTO.AIEventDrillDownDTO, int64, error) {
	return m.events, m.total, m.err
}

type mockDrillDownBrandReader struct {
	brandContract.IBrandDashboardReader
	feedbacks []brandDTO.SampleFeedbackDrillDownDTO
	total     int64
	err       error
}

func (m *mockDrillDownBrandReader) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]brandDTO.SampleFeedbackDrillDownDTO, int64, error) {
	return m.feedbacks, m.total, m.err
}

func TestDrillDown_GetAIEvents(t *testing.T) {
	now := time.Now()
	aiReader := &mockDrillDownAIReader{
		events: []aiDTO.AIEventDrillDownDTO{
			{
				RequestID:    "req-001",
				Provider:     "openai",
				Model:        "gpt-4",
				StatusCode:   500,
				ErrorMessage: "internal error",
				CreatedAt:    now,
			},
			{
				RequestID:    "req-002",
				Provider:     "anthropic",
				Model:        "claude-3",
				StatusCode:   429,
				ErrorMessage: "rate limit",
				CreatedAt:    now.Add(-time.Hour),
			},
		},
		total: 2,
	}

	uc := NewDrillDownUseCase(aiReader, &mockDrillDownBrandReader{})
	res, err := uc.GetAIEventsDrillDown(context.Background(), &dto.DrillDownAIEventQueryReq{
		Status: "FAILED",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	if res.Items[0].RequestID != "req-001" {
		t.Errorf("expected req-001 as first item, got %s", res.Items[0].RequestID)
	}
	if res.Items[0].Provider != "openai" {
		t.Errorf("expected provider 'openai', got %s", res.Items[0].Provider)
	}
	if res.Items[1].StatusCode != 429 {
		t.Errorf("expected status 429 for second item, got %d", res.Items[1].StatusCode)
	}
}

func TestDrillDown_GetAIEvents_NoData(t *testing.T) {
	aiReader := &mockDrillDownAIReader{
		events: nil,
		total:  0,
	}
	uc := NewDrillDownUseCase(aiReader, &mockDrillDownBrandReader{})
	_, err := uc.GetAIEventsDrillDown(context.Background(), &dto.DrillDownAIEventQueryReq{})
	if err == nil {
		t.Fatal("expected error when no data")
	}
}

func TestDrillDown_GetSampleLabFeedbacks(t *testing.T) {
	itemID := uuid.New()
	brandReader := &mockDrillDownBrandReader{
		feedbacks: []brandDTO.SampleFeedbackDrillDownDTO{
			{
				FeedbackID: "fb-001",
				ItemID:     itemID,
				UserID:     uuid.New(),
				Rating:     4.5,
				Comment:    "Nice sample",
				CreatedAt:  time.Now(),
			},
		},
		total: 1,
	}

	uc := NewDrillDownUseCase(&mockDrillDownAIReader{}, brandReader)
	res, err := uc.GetSampleLabFeedbacksDrillDown(context.Background(), &dto.DrillDownSampleFeedbackQueryReq{
		ItemID: itemID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.Items))
	}
	if res.Items[0].FeedbackID != "fb-001" {
		t.Errorf("expected FeedbackID 'fb-001', got %s", res.Items[0].FeedbackID)
	}
	if res.Items[0].Rating != 4.5 {
		t.Errorf("expected Rating 4.5, got %v", res.Items[0].Rating)
	}
}

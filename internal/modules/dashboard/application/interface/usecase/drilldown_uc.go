package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
)

type IDrillDownUseCase interface {
	GetAIEventsDrillDown(ctx context.Context, req *dto.DrillDownAIEventQueryReq) (*dto.DrillDownAIEventListRes, error)
	GetSampleLabFeedbacksDrillDown(ctx context.Context, req *dto.DrillDownSampleFeedbackQueryReq) (*dto.DrillDownSampleFeedbackListRes, error)
}

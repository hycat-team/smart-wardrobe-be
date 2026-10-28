package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"

	"github.com/google/uuid"
)

type IDrillDownUseCase interface {
	GetAIEventsDrillDown(ctx context.Context, req *dto.DrillDownAIEventQueryReq) (*dto.DrillDownAIEventListRes, error)
	GetSampleLabFeedbacksDrillDown(ctx context.Context, brandID uuid.UUID, req *dto.DrillDownSampleFeedbackQueryReq) (*dto.DrillDownSampleFeedbackListRes, error)
}

package usecase

import (
	"context"

	aiContract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type DrillDownUseCase struct {
	aiCostReader aiContract.IAICostDashboardReader
	brandReader  brandContract.IBrandDashboardReader
}

func NewDrillDownUseCase(
	aiCostReader aiContract.IAICostDashboardReader,
	brandReader brandContract.IBrandDashboardReader,
) uc_interfaces.IDrillDownUseCase {
	return &DrillDownUseCase{
		aiCostReader: aiCostReader,
		brandReader:  brandReader,
	}
}

func (uc *DrillDownUseCase) GetAIEventsDrillDown(ctx context.Context, req *dto.DrillDownAIEventQueryReq) (*dto.DrillDownAIEventListRes, error) {
	normQuery := req.PaginationQuery.Normalize()

	rawItems, totalCount, err := uc.aiCostReader.GetAIEventsDrillDown(ctx, req.Status, normQuery.Page, normQuery.Limit)
	if err != nil {
		return nil, err
	}

	if len(rawItems) == 0 {
		return nil, dashboarderrors.ErrDashboardDataNotFound()
	}

	items := make([]dto.DrillDownAIEventDTO, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, dto.DrillDownAIEventDTO{
			RequestID:    item.RequestID,
			Provider:     item.Provider,
			Model:        item.Model,
			StatusCode:   item.StatusCode,
			ErrorMessage: item.ErrorMessage,
			CreatedAt:    item.CreatedAt,
		})
	}

	meta := shared_dto.BuildPaginationMetadata(normQuery, totalCount)

	return &dto.DrillDownAIEventListRes{
		Items:    items,
		Metadata: meta,
	}, nil
}

func (uc *DrillDownUseCase) GetSampleLabFeedbacksDrillDown(ctx context.Context, req *dto.DrillDownSampleFeedbackQueryReq) (*dto.DrillDownSampleFeedbackListRes, error) {
	normQuery := req.PaginationQuery.Normalize()
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
	}

	rawItems, totalCount, err := uc.brandReader.GetSampleLabFeedbacksDrillDown(ctx, itemID, normQuery.Page, normQuery.Limit)
	if err != nil {
		return nil, err
	}

	if len(rawItems) == 0 {
		return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
	}

	items := make([]dto.DrillDownSampleFeedbackDTO, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, dto.DrillDownSampleFeedbackDTO{
			FeedbackID: item.FeedbackID,
			ItemID:     item.ItemID,
			UserID:     item.UserID,
			Rating:     item.Rating,
			Comment:    item.Comment,
			CreatedAt:  item.CreatedAt,
		})
	}

	meta := shared_dto.BuildPaginationMetadata(normQuery, totalCount)

	return &dto.DrillDownSampleFeedbackListRes{
		Items:    items,
		Metadata: meta,
	}, nil
}

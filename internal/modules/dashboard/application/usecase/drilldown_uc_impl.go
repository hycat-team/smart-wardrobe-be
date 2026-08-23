package usecase

import (
	"context"

	aiContract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brand_contract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type DrillDownUseCase struct {
	aiCostReader  aiContract.IAICostDashboardReader
	brandReader   brand_contract.IBrandDashboardReader
	userContract  identity_contract.IUserContract
	brandContract brand_contract.IBrandContract
}

func NewDrillDownUseCase(
	aiCostReader aiContract.IAICostDashboardReader,
	brandReader brand_contract.IBrandDashboardReader,
	userContract identity_contract.IUserContract,
	brandContract brand_contract.IBrandContract,
) uc_interfaces.IDrillDownUseCase {
	return &DrillDownUseCase{
		aiCostReader:  aiCostReader,
		brandReader:   brandReader,
		userContract:  userContract,
		brandContract: brandContract,
	}
}

func (uc *DrillDownUseCase) GetAIEventsDrillDown(ctx context.Context, req *dto.DrillDownAIEventQueryReq) (*dto.DrillDownAIEventListRes, error) {
	normQuery := req.PaginationQuery.Normalize()

	rawItems, totalCount, err := uc.aiCostReader.GetAIEventsDrillDown(ctx, req.Status, normQuery.Page, normQuery.Limit)
	if err != nil {
		return nil, err
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

	userIDs := make([]uuid.UUID, 0, len(rawItems))
	brandItemIDs := make([]uuid.UUID, 0, len(rawItems))
	for _, item := range rawItems {
		userIDs = append(userIDs, item.UserID)
		brandItemIDs = append(brandItemIDs, item.ItemID)
	}

	userBriefs := make([]*identity_dto.UserBriefRes, 0)
	if uc.userContract != nil {
		userBriefs, _ = uc.userContract.GetBriefByIDs(ctx, userIDs)
	}
	userMap := make(map[uuid.UUID]*identity_dto.UserBriefRes, len(userBriefs))
	for _, ub := range userBriefs {
		userMap[ub.ID] = ub
	}

	brandItemNameMap := make(map[uuid.UUID]*string)
	brandItemImageMap := make(map[uuid.UUID]*string)
	if len(brandItemIDs) > 0 && uc.brandContract != nil {
		brandItems, _ := uc.brandContract.GetBrandItemsByIDs(ctx, brandItemIDs)
		for _, bi := range brandItems {
			name := bi.Name
			brandItemNameMap[bi.ID] = &name
			if bi.FashionItem != nil {
				img := bi.FashionItem.ImageUrl
				brandItemImageMap[bi.ID] = &img
			}
		}
	}

	items := make([]dto.DrillDownSampleFeedbackDTO, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, dto.DrillDownSampleFeedbackDTO{
			FeedbackID:        item.FeedbackID,
			ItemID:            item.ItemID,
			UserID:            item.UserID,
			Rating:            item.Rating,
			Comment:           item.Comment,
			CreatedAt:         item.CreatedAt,
			User:              userMap[item.UserID],
			BrandItemName:     brandItemNameMap[item.ItemID],
			BrandItemImageURL: brandItemImageMap[item.ItemID],
		})
	}

	meta := shared_dto.BuildPaginationMetadata(normQuery, totalCount)

	return &dto.DrillDownSampleFeedbackListRes{
		Items:    items,
		Metadata: meta,
	}, nil
}

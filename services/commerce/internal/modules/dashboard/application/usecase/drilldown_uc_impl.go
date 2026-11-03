package usecase

import (
	"context"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	aiContract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	brand_contract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/mapper"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type DrillDownUseCase struct {
	aiCostReader      aiContract.IAICostDashboardReader
	brandReader       brand_contract.IBrandDashboardReader
	userBriefContract identity_contract.IUserBriefContract
	brandContract     brand_contract.IBrandContract
}

func NewDrillDownUseCase(
	aiCostReader aiContract.IAICostDashboardReader,
	brandReader brand_contract.IBrandDashboardReader,
	userBriefContract identity_contract.IUserBriefContract,
	brandContract brand_contract.IBrandContract,
) uc_interfaces.IDrillDownUseCase {
	return &DrillDownUseCase{
		aiCostReader:      aiCostReader,
		brandReader:       brandReader,
		userBriefContract: userBriefContract,
		brandContract:     brandContract,
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
			ErrorCode:    item.ErrorCode,
			CreatedAt:    item.CreatedAt,
		})
	}

	meta := shared_dto.BuildPaginationMetadata(normQuery, totalCount)

	return &dto.DrillDownAIEventListRes{
		Items:    items,
		Metadata: meta,
	}, nil
}

func (uc *DrillDownUseCase) GetSampleLabFeedbacksDrillDown(ctx context.Context, brandID uuid.UUID, req *dto.DrillDownSampleFeedbackQueryReq) (*dto.DrillDownSampleFeedbackListRes, error) {
	normQuery := req.PaginationQuery.Normalize()
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
	}

	if uc.brandContract != nil {
		item, err := uc.brandContract.GetBrandItemByID(ctx, itemID)
		if err != nil {
			return nil, err
		}
		if item == nil || item.BrandID != brandID {
			return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
		}
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

	userBriefs := make([]*identity_dto.UserBriefContractDTO, 0)
	if uc.userBriefContract != nil {
		userBriefs, _ = uc.userBriefContract.GetByIDs(ctx, userIDs)
	}
	userMap := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(userBriefs))
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
			User:              mapper.MapIdentityUserBrief(userMap[item.UserID]),
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

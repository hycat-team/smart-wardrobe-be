package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type IBrandItemUseCase interface {
	GetBrandItemUploadSignature(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, []*fashion_dto.FashionItemContractDTO, error)
	ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *fashion_dto.FashionItemContractDTO, error)
	CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error)
	GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) ([]*dto.BrandItemRes, error)
	AssertTaskBelongsToBrand(ctx context.Context, brandID uuid.UUID, taskID uuid.UUID) error
	GetBrandItemForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemForUser(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error)
	UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error)
	ConvertSampleToProduct(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.ConvertSampleToProductReq) (*dto.BrandItemRes, error)
	UpdateBrandItemStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, status branditemstatus.BrandItemStatus) (*dto.BrandItemRes, error)
	RetryBrandItemAnalysis(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, categoryID *uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error)
	ListBrandProductsForCustomer(ctx context.Context, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) (*dto.BrandItemListBriefRes, error)
	ListBrandSamplesForCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) (*dto.BrandItemListBriefRes, error)
	SubmitSampleFeedback(ctx context.Context, userID uuid.UUID, brandItemID uuid.UUID, input dto.SubmitSampleFeedbackReq) (*dto.DigitalSampleResponseRes, error)
	UpdateBrandItemStatusDirect(ctx context.Context, brandItemID uuid.UUID, status branditemstatus.BrandItemStatus) error
	GetBrandItemByIDDirect(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemBriefRes, error)
	GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemRes, error)
}

type IBrandFashionEventUseCase interface {
	HandleFashionEvent(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error
	HandleFashionAnalyzed(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error
	HandleFashionFailed(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error
	HandleFashionProcessing(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error
	HandleFashionProcessingExhausted(ctx context.Context, payload fashion_dto.FashionProcessingExhaustedEventPayload) error
	HandleFashionItemCreated(ctx context.Context, payload fashion_dto.FashionItemCreatedEventPayload) error
}

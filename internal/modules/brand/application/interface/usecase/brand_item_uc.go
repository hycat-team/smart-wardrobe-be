package usecase

import (
	"context"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
)

type IBrandItemUseCase interface {
	GetBrandItemUploadSignature(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemStylingDTO, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, error)
	CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error)
	GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error)
	GetBrandItemForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemForUser(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error)
	UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error)
	UpdateBrandItemStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, status string) (*dto.BrandItemRes, error)
	GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error)
	ListBrandItemsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error)
	SubmitSampleFeedback(ctx context.Context, userID uuid.UUID, brandItemID uuid.UUID, input dto.SubmitSampleFeedbackReq) (*dto.DigitalSampleResponseRes, error)
}

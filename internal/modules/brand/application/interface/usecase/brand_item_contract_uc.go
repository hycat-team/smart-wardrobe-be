package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"

	"github.com/google/uuid"
)

type IBrandItemContractUseCase interface {
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, error)
	ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	SetFashionItemID(ctx context.Context, brandItemID uuid.UUID, fashionItemID uuid.UUID) error
	UpdateBrandItemStatusDirect(ctx context.Context, brandItemID uuid.UUID, status branditemstatus.BrandItemStatus) error
	GetBrandItemByIDDirect(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemBriefRes, error)
	GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemRes, error)
}

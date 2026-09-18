package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"

	"github.com/google/uuid"
)

type IBrandItemContractUseCase interface {
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemContractDTO, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemContractDTO, error)
	ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	UpdateBrandItemStatusDirect(ctx context.Context, brandItemID uuid.UUID, status branditemstatus.BrandItemStatus) error
	GetBrandItemByIDDirect(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemContractDTO, error)
	GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemContractDTO, error)
	GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemContractDTO, error)
}

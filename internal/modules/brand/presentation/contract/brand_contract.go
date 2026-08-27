package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type IBrandContract interface {
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, []*fashion_dto.FashionItemRes, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *fashion_dto.FashionItemRes, error)
	ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	SetFashionItemID(ctx context.Context, brandItemID uuid.UUID, fashionItemID uuid.UUID) error
	GetBrandItemByID(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemBriefRes, error)
	GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemRes, error)
}

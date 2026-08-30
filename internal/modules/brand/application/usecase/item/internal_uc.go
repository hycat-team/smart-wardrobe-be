package item

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"

	"github.com/google/uuid"
)

func (uc *BrandItemUseCase) UpdateBrandItemStatusDirect(ctx context.Context, brandItemID uuid.UUID, status branditemstatus.BrandItemStatus) error {
	item, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	item.Status = status
	return uc.brandItemRepo.Update(ctx, item)
}

func (uc *BrandItemUseCase) GetBrandItemByIDDirect(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemBriefRes, error) {
	if len(fashionItemIDs) == 0 {
		return []*dto.BrandItemBriefRes{}, nil
	}
	items, err := uc.brandItemRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	activeStatus := branditemstatus.Active
	filtered := make([]*entities.BrandItem, 0, len(items))
	for _, item := range items {
		if item.Status == activeStatus {
			filtered = append(filtered, item)
		}
	}
	return uc.mapSliceToBriefRes(ctx, filtered), nil
}

func (uc *BrandItemUseCase) GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemRes, error) {
	if len(brandItemIDs) == 0 {
		return []*dto.BrandItemRes{}, nil
	}
	items, err := uc.brandItemRepo.GetByIDs(ctx, brandItemIDs)
	if err != nil {
		return nil, err
	}
	return uc.mapSliceToRes(ctx, items), nil
}

package contractuc

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

type WardrobeContractUseCase struct {
	wardrobeRepo repositories.IWardrobeItemRepository
}

func NewWardrobeContractUseCase(
	wardrobeRepo repositories.IWardrobeItemRepository,
) uc_interfaces.IWardrobeContractUseCase {
	return &WardrobeContractUseCase{
		wardrobeRepo: wardrobeRepo,
	}
}

func (uc *WardrobeContractUseCase) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	item.Status = status
	return uc.wardrobeRepo.Update(ctx, item)
}

func (uc *WardrobeContractUseCase) SetFashionItemID(ctx context.Context, itemID uuid.UUID, fashionItemID uuid.UUID) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	item.FashionItemID = &fashionItemID
	return uc.wardrobeRepo.Update(ctx, item)
}

func (uc *WardrobeContractUseCase) GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.WardrobeItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapper.MapToWardrobeItemRes(item, nil))
	}
	return result, nil
}

func (uc *WardrobeContractUseCase) GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetActiveByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.WardrobeItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapper.MapToWardrobeItemRes(item, nil))
	}
	return result, nil
}

func (uc *WardrobeContractUseCase) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetFailedItemsForCleanup(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.WardrobeItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapper.MapToWardrobeItemRes(item, nil))
	}
	return result, nil
}

func (uc *WardrobeContractUseCase) DeleteWardrobeItem(ctx context.Context, itemID uuid.UUID) error {
	return uc.wardrobeRepo.Delete(ctx, itemID)
}

func (uc *WardrobeContractUseCase) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetStaleProcessingItems(ctx, staleBefore, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.WardrobeItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapper.MapToWardrobeItemRes(item, nil))
	}
	return result, nil
}

func (uc *WardrobeContractUseCase) CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error) {
	return uc.wardrobeRepo.CountItems(ctx, fashionItemIDs, 1) // UserItem = 1
}

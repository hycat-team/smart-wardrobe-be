package contractuc

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
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

package contractuc

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

type WardrobeContractUseCase struct {
	wardrobeRepo repositories.IWardrobeItemRepository
	outfitRepo   repositories.IOutfitRepository
}

func NewWardrobeContractUseCase(
	wardrobeRepo repositories.IWardrobeItemRepository,
	outfitRepo repositories.IOutfitRepository,
) uc_interfaces.IWardrobeContractUseCase {
	return &WardrobeContractUseCase{
		wardrobeRepo: wardrobeRepo,
		outfitRepo:   outfitRepo,
	}
}

func (uc *WardrobeContractUseCase) GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemContractDTO, error) {
	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	return mapper.MapToWardrobeItemContractDTOList(items), nil
}

func (uc *WardrobeContractUseCase) GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	items, err := uc.wardrobeRepo.GetActiveByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	return mapper.MapToWardrobeItemContractDTOList(items), nil
}

func (uc *WardrobeContractUseCase) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	items, err := uc.wardrobeRepo.GetFailedItemsForCleanup(ctx, limit)
	if err != nil {
		return nil, err
	}

	return mapper.MapToWardrobeItemContractDTOList(items), nil
}

func (uc *WardrobeContractUseCase) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	items, err := uc.wardrobeRepo.GetStaleProcessingItems(ctx, staleBefore, limit)
	if err != nil {
		return nil, err
	}

	return mapper.MapToWardrobeItemContractDTOList(items), nil
}

// CountFashionItemOwnersByFashionIDs returns, per fashion item, how many owners
// still reference it: non-deleted wardrobe items of every item type (optionally
// excluding the given wardrobe item IDs) plus outfit items. Callers use it to
// decide whether a shared image asset may be hard-deleted and to guard category
// deletion.
func (uc *WardrobeContractUseCase) CountFashionItemOwnersByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID, excludeWardrobeItemIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	owners, err := uc.wardrobeRepo.CountItemsGroupedByFashionIDs(ctx, fashionItemIDs, excludeWardrobeItemIDs)
	if err != nil {
		return nil, err
	}
	outfitOwners, err := uc.outfitRepo.CountOutfitItemsByFashionIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	for fashionItemID, count := range outfitOwners {
		owners[fashionItemID] += count
	}
	return owners, nil
}

// GetWardrobeItemsByFashionItemIDs returns wardrobe items referencing the given
// fashion items (across users), for stale-processing recovery (FR-035).
func (uc *WardrobeContractUseCase) GetWardrobeItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.WardrobeItemContractDTO, error) {
	items, err := uc.wardrobeRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	return mapper.MapToWardrobeItemContractDTOList(items), nil
}

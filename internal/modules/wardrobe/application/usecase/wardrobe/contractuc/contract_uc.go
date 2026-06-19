package contractuc

import (
	"context"

	subcontract "smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type WardrobeContractUseCase struct {
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract subcontract.IUserSubscriptionContract
}

func NewWardrobeContractUseCase(
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract subcontract.IUserSubscriptionContract,
) contract.IWardrobeContract {
	return &WardrobeContractUseCase{
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
	}
}

func (uc *WardrobeContractUseCase) CopyItemToUser(ctx context.Context, sourceItemID uuid.UUID, targetUserID uuid.UUID) (*dto.WardrobeItemRes, error) {
	sourceItem, err := uc.wardrobeRepo.GetByID(ctx, sourceItemID)
	if err != nil {
		return nil, err
	}
	if sourceItem == nil {
		return nil, wardrobeerrors.ErrItemToCloneNotFound()
	}

	cloned := &entities.WardrobeItem{
		UserID:        targetUserID,
		CategoryID:    sourceItem.CategoryID,
		ImageUrl:      sourceItem.ImageUrl,
		ImagePublicID: sourceItem.ImagePublicID,
		Color:         sourceItem.Color,
		Style:         sourceItem.Style,
		Material:      sourceItem.Material,
		Pattern:       sourceItem.Pattern,
		Fit:           sourceItem.Fit,
		Seasonality:   sourceItem.Seasonality,
		Description:   sourceItem.Description,
		Price:         sourceItem.Price,
		Status:        wardrobestatus.InWardrobe,
		ItemType:      itemtype.UserItem,
		Embedding:     sourceItem.Embedding,
	}

	if err := uc.wardrobeRepo.Create(ctx, cloned); err != nil {
		return nil, err
	}

	cloned.Category = sourceItem.Category
	return mapper.MapToWardrobeItemRes(cloned), nil
}

func (uc *WardrobeContractUseCase) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return wardrobeerrors.ErrItemNotFound()
	}

	item.Status = status
	return uc.wardrobeRepo.Update(ctx, item)
}

func (uc *WardrobeContractUseCase) VerifyItemsForPost(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		return nil
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return err
	}

	userItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return err
	}

	lockedMap := shared.BuildLockedMap(userItems, subOverview.MaxWardrobeItems)

	items, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return err
	}

	itemMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range items {
		itemMap[item.ID] = item
	}

	for _, itemID := range itemIDs {
		item, ok := itemMap[itemID]
		if !ok {
			return wardrobeerrors.ErrItemNotFoundWithID(itemID)
		}
		if item.UserID != userID {
			return wardrobeerrors.ErrItemForbiddenWithID(itemID)
		}
		if item.Status == wardrobestatus.Sold {
			return wardrobeerrors.ErrItemSoldWithID(itemID)
		}
		if lockedMap[itemID] {
			return wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
		}
	}

	return nil
}

func (uc *WardrobeContractUseCase) GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.WardrobeItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapper.MapToWardrobeItemRes(item))
	}
	return result, nil
}

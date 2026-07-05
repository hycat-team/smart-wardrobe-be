package contractuc

import (
	"context"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	subcontract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

type WardrobeContractUseCase struct {
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract subcontract.IUserSubscriptionContract
	fashionContract fashion_contract.IFashionContract
}

func NewWardrobeContractUseCase(
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract subcontract.IUserSubscriptionContract,
	fashionContract fashion_contract.IFashionContract,
) uc_interfaces.IWardrobeContractUseCase {
	return &WardrobeContractUseCase{
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
		fashionContract: fashionContract,
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

	fashionItemID := sourceItem.FashionItemID
	if fashionItemID == nil {
		fashionItemID = &sourceItem.ID
	}
	cloned := &entities.WardrobeItem{
		UserID:        targetUserID,
		FashionItemID: fashionItemID,
		Price:         sourceItem.Price,
		Status:        wardrobestatus.InWardrobe,
		ItemType:      itemtype.UserItem,
	}

	if err := uc.wardrobeRepo.Create(ctx, cloned); err != nil {
		return nil, err
	}

	var fi *fashion_dto.FashionItemRes
	if cloned.FashionItemID != nil {
		fi, _ = uc.fashionContract.GetFashionItem(ctx, *cloned.FashionItemID)
	}
	return mapper.MapToWardrobeItemRes(cloned, fi), nil
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

func (uc *WardrobeContractUseCase) GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionMap := map[uuid.UUID]*fashion_dto.FashionItemRes{}
	if len(fashionIDs) > 0 {
		fashionItems, err := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
		if err == nil {
			fashionMap = sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemRes) uuid.UUID { return f.ID })
		}
	}

	result := make([]*dto.WardrobeItemRes, 0, len(items))
	for _, item := range items {
		var fi *fashion_dto.FashionItemRes
		if item.FashionItemID != nil {
			fi = fashionMap[*item.FashionItemID]
		}
		result = append(result, mapper.MapToWardrobeItemRes(item, fi))
	}
	return result, nil
}

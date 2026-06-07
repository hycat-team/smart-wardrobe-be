package wardrobe

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

func (uc *WardrobeUseCase) InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	if len(catalogItemIDs) == 0 {
		return nil, wardrobeerrors.ErrCatalogItemIDsEmpty
	}

	// 1. Check wardrobe item limit of the subscription plan
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if int(currentCount)+len(catalogItemIDs) > subOverview.MaxWardrobeItems {
		return nil, wardrobeerrors.ErrWardrobeLimitExceededForCatalog(int(currentCount), subOverview.MaxWardrobeItems, len(catalogItemIDs))
	}

	// 2. Fetch system template catalog items from Database
	templates, err := uc.wardrobeRepo.GetByIDs(ctx, catalogItemIDs)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, wardrobeerrors.ErrCatalogItemNotFound
	}

	newItems := make([]*entities.WardrobeItem, len(templates))
	for i, original := range templates {
		newItems[i] = &entities.WardrobeItem{
			UserID:        userID,
			CategoryID:    original.CategoryID,
			ImageUrl:      original.ImageUrl,
			ImagePublicID: original.ImagePublicID,
			Color:         original.Color,
			Style:         original.Style,
			Material:      original.Material,
			Pattern:       original.Pattern,
			Fit:           original.Fit,
			Seasonality:   original.Seasonality,
			Description:   original.Description,
			Embedding:     original.Embedding,
			Status:        wardrobestatus.InWardrobe,
			ItemType:      itemtype.UserItem, // Once copied to the user's wardrobe, it becomes a personal UserItem
		}
	}

	err = uc.wardrobeRepo.BulkCreate(ctx, newItems)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i := 0; i < len(newItems); i++ {
		newItems[i].Category = templates[i].Category
		resList[i] = mapper.MapToWardrobeItemRes(newItems[i])
		resList[i].IsLocked = false
	}

	return resList, nil
}

func (uc *WardrobeUseCase) GetSystemCatalogItems(ctx context.Context, query dto.GetSystemCatalogItemsQueryReq) ([]*dto.WardrobeItemRes, error) {
	items, err := uc.wardrobeRepo.GetItems(ctx, query.Query, query.CategorySlug, itemtype.SystemCatalogItem)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(items))
	for i, item := range items {
		resList[i] = mapper.MapToWardrobeItemRes(item)
		resList[i].IsLocked = false
	}
	return resList, nil
}

func (uc *WardrobeUseCase) UpdateSystemCatalogItem(ctx context.Context, id uuid.UUID, input dto.UpdateSystemCatalogItemReq) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.ItemType != itemtype.SystemCatalogItem {
		return nil, wardrobeerrors.ErrCatalogItemNotFound
	}

	if input.CategoryID != nil {
		category, err := uc.categoryRepo.GetByID(ctx, *input.CategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, wardrobeerrors.ErrCategoryNotFound
		}
		item.CategoryID = input.CategoryID
		item.Category = category
	}

	if input.Color != nil {
		item.Color = input.Color
	}
	if input.Style != nil {
		item.Style = input.Style
	}
	if input.Material != nil {
		item.Material = input.Material
	}
	if input.Pattern != nil {
		item.Pattern = input.Pattern
	}
	if input.Fit != nil {
		item.Fit = input.Fit
	}
	if input.Seasonality != nil {
		item.Seasonality = input.Seasonality
	}
	if input.Price != nil {
		item.Price = input.Price
	}

	if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	res := mapper.MapToWardrobeItemRes(item)
	res.IsLocked = false
	return res, nil
}

func (uc *WardrobeUseCase) DeleteSystemCatalogItem(ctx context.Context, id uuid.UUID) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil || item.ItemType != itemtype.SystemCatalogItem {
		return wardrobeerrors.ErrCatalogItemNotFound
	}

	return uc.wardrobeRepo.Delete(ctx, id)
}


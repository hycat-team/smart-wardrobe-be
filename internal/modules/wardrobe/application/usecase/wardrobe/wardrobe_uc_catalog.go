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

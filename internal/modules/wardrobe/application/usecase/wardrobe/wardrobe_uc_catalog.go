package wardrobe

import (
	"context"
	"math"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

// ... InitClosetFromCatalog omitted, let's keep targetContent precise ...
func (uc *WardrobeCatalogUseCase) GetSystemCatalogItems(ctx context.Context, query dto.GetSystemCatalogItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	totalItems, err := uc.wardrobeRepo.CountItems(ctx, query.Query, query.CategorySlug, itemtype.SystemCatalogItem)
	if err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	items, err := uc.wardrobeRepo.GetItemsPaginated(ctx, query.Query, query.CategorySlug, itemtype.SystemCatalogItem, paginationQuery)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(items))
	for i, item := range items {
		resList[i] = mapper.MapToWardrobeItemRes(item)
		resList[i].IsLocked = false
	}

	totalPages := 0
	if limit > 0 && totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	}

	return &shared_dto.PaginationResult[*dto.WardrobeItemRes]{
		Items: resList,
		Metadata: shared_dto.PaginationMetadata{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *WardrobeCatalogUseCase) InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
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


func (uc *WardrobeCatalogUseCase) UpdateSystemCatalogItem(ctx context.Context, id uuid.UUID, input dto.UpdateSystemCatalogItemReq) (*dto.WardrobeItemRes, error) {
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

	payload := dto.WardrobeEventPayload{
		ItemID: item.ID,
		UserID: item.UserID,
		Action: eventconstants.ActionUpdated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeUpdated, payload)

	res := mapper.MapToWardrobeItemRes(item)
	res.IsLocked = false
	return res, nil
}

func (uc *WardrobeCatalogUseCase) DeleteSystemCatalogItem(ctx context.Context, id uuid.UUID) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil || item.ItemType != itemtype.SystemCatalogItem {
		return wardrobeerrors.ErrCatalogItemNotFound
	}

	if err := uc.wardrobeRepo.Delete(ctx, id); err != nil {
		return err
	}

	payload := dto.WardrobeEventPayload{
		ItemID: id,
		UserID: item.UserID,
		Action: eventconstants.ActionDeleted,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeDeleted, payload)

	return nil
}

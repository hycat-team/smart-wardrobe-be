package catalog

import (
	"context"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

type WardrobeCatalogUseCase struct {
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract contract.IUserSubscriptionContract
	eventPublisher  event.IEventPublisher
	fashionContract fashion_contract.IFashionContract
}

func NewWardrobeCatalogUseCase(
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
	fashionContract fashion_contract.IFashionContract,
) uc_interfaces.IWardrobeCatalogUseCase {
	return &WardrobeCatalogUseCase{
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
		eventPublisher:  eventPublisher,
		fashionContract: fashionContract,
	}
}

func (uc *WardrobeCatalogUseCase) GetSystemCatalogItems(ctx context.Context, query dto.GetSystemCatalogItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	totalItems, err := uc.wardrobeRepo.CountItems(ctx, query.Query, query.CategorySlug, itemtype.SystemCatalogItem)
	if err != nil {
		return nil, err
	}

	items, err := uc.wardrobeRepo.GetItemsPaginated(ctx, query.Query, query.CategorySlug, itemtype.SystemCatalogItem, paginationQuery)
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

	resList := make([]*dto.WardrobeItemRes, len(items))
	for i, item := range items {
		var fi *fashion_dto.FashionItemRes
		if item.FashionItemID != nil {
			fi = fashionMap[*item.FashionItemID]
		}
		resList[i] = mapper.MapToWardrobeItemRes(item, fi)
		resList[i].IsLocked = false
	}

	return &shared_dto.PaginationResult[*dto.WardrobeItemRes]{
		Items:    resList,
		Metadata: shared_dto.BuildPaginationMetadata(paginationQuery, totalItems),
	}, nil
}

func (uc *WardrobeCatalogUseCase) InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	if len(catalogItemIDs) == 0 {
		return nil, wardrobeerrors.ErrCatalogItemIDsEmpty()
	}

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

	templates, err := uc.wardrobeRepo.GetByIDs(ctx, catalogItemIDs)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, wardrobeerrors.ErrCatalogItemNotFound()
	}

	newItems := make([]*entities.WardrobeItem, len(templates))
	for i, original := range templates {
		fashionItemID := original.FashionItemID
		if fashionItemID == nil {
			fashionItemID = &original.ID
		}
		newItems[i] = &entities.WardrobeItem{
			UserID:        userID,
			FashionItemID: fashionItemID,
			Price:         original.Price,
			Status:        wardrobestatus.InWardrobe,
			ItemType:      itemtype.UserItem,
		}
	}

	err = uc.wardrobeRepo.BulkCreate(ctx, newItems)
	if err != nil {
		return nil, err
	}

	fashionIDs := make([]uuid.UUID, 0, len(newItems))
	for _, item := range newItems {
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

	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i := range newItems {
		var fi *fashion_dto.FashionItemRes
		if newItems[i].FashionItemID != nil {
			fi = fashionMap[*newItems[i].FashionItemID]
		}
		resList[i] = mapper.MapToWardrobeItemRes(newItems[i], fi)
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
		return nil, wardrobeerrors.ErrCatalogItemNotFound()
	}
	if input.Price != nil {
		item.Price = input.Price
	}

	if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	payload := dto.WardrobeEventPayload{
		ItemID:        item.ID,
		UserID:        item.UserID,
		FashionItemID: fashionItemID,
		ItemType:      int(item.ItemType),
		Action:        eventconstants.ActionUpdated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeUpdated, payload)

	var fi *fashion_dto.FashionItemRes
	if item.FashionItemID != nil {
		fi, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	res := mapper.MapToWardrobeItemRes(item, fi)
	res.IsLocked = false
	return res, nil
}

func (uc *WardrobeCatalogUseCase) DeleteSystemCatalogItem(ctx context.Context, id uuid.UUID) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil || item.ItemType != itemtype.SystemCatalogItem {
		return wardrobeerrors.ErrCatalogItemNotFound()
	}

	if err := uc.wardrobeRepo.Delete(ctx, id); err != nil {
		return err
	}

	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	payload := dto.WardrobeEventPayload{
		ItemID:        id,
		UserID:        item.UserID,
		FashionItemID: fashionItemID,
		ItemType:      int(item.ItemType),
		Action:        eventconstants.ActionDeleted,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeDeleted, payload)

	return nil
}

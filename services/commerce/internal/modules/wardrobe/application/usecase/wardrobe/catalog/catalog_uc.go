package catalog

import (
	"context"

	"smart-wardrobe-be/services/account/contract/subscription"
	brand_contract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/application/event"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

type WardrobeCatalogUseCase struct {
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract contract.IUserSubscriptionContract
	brandContract   brand_contract.IBrandContract
	eventPublisher  event.IEventPublisher
	fashionContract fashion_contract.IFashionContract
	uow             shared_repos.IUnitOfWork
}

func NewWardrobeCatalogUseCase(
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract contract.IUserSubscriptionContract,
	brandContract brand_contract.IBrandContract,
	eventPublisher event.IEventPublisher,
	fashionContract fashion_contract.IFashionContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IWardrobeCatalogUseCase {
	return &WardrobeCatalogUseCase{
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
		brandContract:   brandContract,
		eventPublisher:  eventPublisher,
		fashionContract: fashionContract,
		uow:             uow,
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

	var queryStr *string
	if query.Query != nil && *query.Query != "" {
		queryStr = query.Query
	}
	var categorySlug *string
	if query.CategorySlug != nil && *query.CategorySlug != "" {
		categorySlug = query.CategorySlug
	}

	filterByFashionItems := queryStr != nil || categorySlug != nil

	fashionIDs, err := uc.fashionContract.SearchFashionItemIDs(ctx, queryStr, categorySlug)
	if err != nil {
		return nil, err
	}

	// A requested filter that matched no fashion item returns zero catalog items
	// instead of the whole catalog (FR-019/FR-023).
	if filterByFashionItems && len(fashionIDs) == 0 {
		return &shared_dto.PaginationResult[*dto.WardrobeItemRes]{
			Items:    []*dto.WardrobeItemRes{},
			Metadata: shared_dto.BuildPaginationMetadata(paginationQuery, 0),
		}, nil
	}

	totalItems, err := uc.wardrobeRepo.CountItems(ctx, fashionIDs, filterByFashionItems, itemtype.SystemCatalogItem)
	if err != nil {
		return nil, err
	}

	items, err := uc.wardrobeRepo.GetItemsPaginated(ctx, fashionIDs, filterByFashionItems, itemtype.SystemCatalogItem, paginationQuery)
	if err != nil {
		return nil, err
	}

	fashionMap := map[uuid.UUID]*fashion_dto.FashionItemContractDTO{}
	if len(fashionIDs) > 0 {
		fashionItems, err := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
		if err == nil {
			fashionMap = sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemContractDTO) uuid.UUID { return f.ID })
		}
	}

	resList := make([]*dto.WardrobeItemRes, len(items))
	for i, item := range items {
		var fi *fashion_dto.FashionItemContractDTO
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

func (uc *WardrobeCatalogUseCase) InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemBriefRes, error) {
	if len(catalogItemIDs) == 0 {
		return nil, wardrobeerrors.ErrCatalogItemIDsEmpty()
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// The quota check and the insert it guards run in one transaction behind a
	// per-user wardrobe advisory lock, so two concurrent initializations cannot
	// both pass the check and exceed the limit. A failed insert rolls the whole
	// transaction back; the response is mapped after the commit.
	var newItems []*entities.WardrobeItem
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.wardrobeRepo.LockUserWardrobe(txCtx, userID); err != nil {
			return err
		}

		currentCount, err := uc.wardrobeRepo.CountByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		if int(currentCount)+len(catalogItemIDs) > subOverview.MaxWardrobeItems {
			return wardrobeerrors.ErrWardrobeLimitExceededForCatalog(int(currentCount), subOverview.MaxWardrobeItems, len(catalogItemIDs))
		}

		templates, err := uc.wardrobeRepo.GetByIDs(txCtx, catalogItemIDs)
		if err != nil {
			return err
		}
		if len(templates) == 0 {
			return wardrobeerrors.ErrCatalogItemNotFound()
		}

		// Only genuine system catalog templates may be cloned: item_type
		// SystemCatalogItem, status InWardrobe and not soft-deleted. This also
		// rejects another user's private wardrobe item when its id is supplied
		// (FR-035); GetByIDs alone would happily return it.
		templateByID := make(map[uuid.UUID]*entities.WardrobeItem, len(templates))
		for _, template := range templates {
			if template == nil || template.IsDeleted ||
				template.ItemType != itemtype.SystemCatalogItem ||
				template.Status != wardrobestatus.InWardrobe {
				continue
			}
			templateByID[template.ID] = template
		}

		clones := make([]*entities.WardrobeItem, 0, len(catalogItemIDs))
		for _, catalogItemID := range catalogItemIDs {
			original, ok := templateByID[catalogItemID]
			if !ok {
				return wardrobeerrors.ErrCatalogItemNotFound()
			}
			// A genuine system catalog template always references a fashion item.
			// Cloning one without it would leave the new wardrobe row pointing at a
			// fashion item that does not exist, so reject the whole request instead
			// of fabricating a reference from the catalog row id.
			if original.FashionItemID == nil {
				return wardrobeerrors.ErrCatalogItemUnavailable()
			}
			clones = append(clones, &entities.WardrobeItem{
				UserID:        userID,
				FashionItemID: original.FashionItemID,
				Price:         original.Price,
				Status:        wardrobestatus.InWardrobe,
				ItemType:      itemtype.UserItem,
			})
		}

		if err := uc.wardrobeRepo.BulkCreate(txCtx, clones); err != nil {
			return err
		}
		newItems = clones
		return nil
	})
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
	fashionBriefMap := map[uuid.UUID]*fashion_dto.FashionItemBriefRes{}
	if len(fashionIDs) > 0 {
		briefs, err := uc.fashionContract.ListFashionItemsBrief(ctx, fashionIDs)
		if err == nil {
			for _, b := range briefs {
				fashionBriefMap[b.ID] = b
			}
		}
	}

	brandItemMap := map[uuid.UUID]*shared_dto.BrandItemBriefRes{}
	if len(fashionIDs) > 0 && uc.brandContract != nil {
		brandItems, err := uc.brandContract.GetBrandItemsByFashionItemIDs(ctx, fashionIDs)
		if err == nil {
			for _, bi := range brandItems {
				brandItemMap[bi.FashionItemID] = &shared_dto.BrandItemBriefRes{
					ID:          bi.ID,
					BrandID:     bi.BrandID,
					BrandName:   bi.BrandName,
					ProductCode: bi.ProductCode,
					Name:        bi.Name,
					Price:       bi.Price,
					ItemType:    string(bi.ItemType),
					Status:      string(bi.Status),
				}
			}
		}
	}

	resList := make([]*dto.WardrobeItemBriefRes, len(newItems))
	for i := range newItems {
		var fiBrief *fashion_dto.FashionItemBriefRes
		if newItems[i].FashionItemID != nil {
			fiBrief = fashionBriefMap[*newItems[i].FashionItemID]
		}
		var brandItem *shared_dto.BrandItemBriefRes
		if newItems[i].FashionItemID != nil {
			brandItem = brandItemMap[*newItems[i].FashionItemID]
		}
		resList[i] = mapper.MapToWardrobeItemBrief(newItems[i], fiBrief, brandItem)
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

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.wardrobeRepo.Update(txCtx, item); err != nil {
			return err
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
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeUpdated, payload)
	})
	if err != nil {
		return nil, err
	}

	var fi *fashion_dto.FashionItemContractDTO
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

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.wardrobeRepo.Delete(txCtx, id); err != nil {
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
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeDeleted, payload)
	})

	return err
}

package item

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/config"
	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"

	"github.com/google/uuid"
	"go.uber.org/zap"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

type WardrobeItemUseCase struct {
	cfg              *config.Config
	logger           logger.Interface
	wardrobeRepo     repositories.IWardrobeItemRepository
	outfitRepo       repositories.IOutfitRepository
	searchEngine     search.IWardrobeSearchService
	mediaService     media.IMediaService
	aiService        aiexecutor_contract.IAIOrchestrator
	userSubContract  contract.IUserSubscriptionContract
	eventPublisher   event.IEventPublisher
	fashionContract  fashion_contract.IFashionContract
	categoryContract fashion_contract.ICategoryContract
	uow              shared_repos.IUnitOfWork
}

func NewWardrobeItemUseCase(
	cfg *config.Config,
	l logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	outfitRepo repositories.IOutfitRepository,
	searchEngine search.IWardrobeSearchService,
	mediaService media.IMediaService,
	aiService aiexecutor_contract.IAIOrchestrator,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
	fashionContract fashion_contract.IFashionContract,
	categoryContract fashion_contract.ICategoryContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IWardrobeItemUseCase {
	return &WardrobeItemUseCase{
		cfg:              cfg,
		logger:           l,
		wardrobeRepo:     wardrobeRepo,
		outfitRepo:       outfitRepo,
		searchEngine:     searchEngine,
		mediaService:     mediaService,
		aiService:        aiService,
		userSubContract:  userSubContract,
		eventPublisher:   eventPublisher,
		fashionContract:  fashionContract,
		categoryContract: categoryContract,
		uow:              uow,
	}
}

func (uc *WardrobeItemUseCase) GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error) {
	folder := uc.cfg.Cloudinary.ItemFolder
	if folder == "" {
		folder = "smart_wardrobe/items"
	}
	params := shared_dto.UploadSignatureParams{
		Folder: folder,
	}
	return uc.mediaService.GenerateUploadSignature(ctx, params)
}

func (uc *WardrobeItemUseCase) GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemBriefRes], error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	var categorySlug *string
	if query.CategorySlug != "" {
		categorySlug = &query.CategorySlug
	}

	var filterFashionIDs []uuid.UUID
	if categorySlug != nil {
		filterFashionIDs, err = uc.fashionContract.SearchFashionItemIDs(ctx, nil, categorySlug)
		if err != nil {
			return nil, err
		}
	}

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

	statuses, err := resolveWardrobeStatusFilter(query.Status)
	if err != nil {
		return nil, err
	}

	totalItems, err := uc.wardrobeRepo.CountByUserIDAndFilters(ctx, userID, filterFashionIDs, statuses)
	if err != nil {
		return nil, err
	}

	items, err := uc.wardrobeRepo.GetByUserIDAndFiltersPaginated(ctx, userID, filterFashionIDs, statuses, paginationQuery)
	if err != nil {
		return nil, err
	}

	activeItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	lockedMap := shared.BuildLockedMap(activeItems, subOverview.MaxWardrobeItems)

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
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

	resList := make([]*dto.WardrobeItemBriefRes, len(items))
	for idx, item := range items {
		var fiBrief *fashion_dto.FashionItemBriefRes
		if item.FashionItemID != nil {
			fiBrief = fashionBriefMap[*item.FashionItemID]
		}
		res := mapper.MapToWardrobeItemBrief(item, fiBrief, nil)
		res.IsLocked = lockedMap[item.ID]
		resList[idx] = res
	}

	return &shared_dto.PaginationResult[*dto.WardrobeItemBriefRes]{
		Items:    resList,
		Metadata: shared_dto.BuildPaginationMetadata(paginationQuery, totalItems),
	}, nil
}

func (uc *WardrobeItemUseCase) GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.UserID != userID {
		return nil, wardrobeerrors.ErrItemNotFound()
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Read active items list to determine lock status by order
	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	isLocked := shared.IsItemLocked(items, id, subOverview.MaxWardrobeItems)

	var fi *fashion_dto.FashionItemRes
	if item.FashionItemID != nil {
		fi, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	res := mapper.MapToWardrobeItemRes(item, fi)
	res.IsLocked = isLocked
	return res, nil
}

func (uc *WardrobeItemUseCase) GetSystemCatalogWardrobeItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.SearchWardrobeItemRes], error) {
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

	var results []*dto.SearchWardrobeItemRes
	var totalItems int64
	var err error

	searchCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	results, totalItems, err = uc.searchEngine.SearchItems(searchCtx, query)
	if err != nil {
		uc.logger.Warn("[GetSystemCatalogWardrobeItems] Search failed or timed out, falling back to database", zap.Error(err))
		var searchQ *string
		if query.Query != "" {
			searchQ = &query.Query
		}
		var categorySlug *string
		if query.CategorySlug != "" {
			categorySlug = &query.CategorySlug
		}

		fashionIDs, searchErr := uc.fashionContract.SearchFashionItemIDs(ctx, searchQ, categorySlug)
		if searchErr != nil {
			return nil, searchErr
		}

		totalItems, err = uc.wardrobeRepo.CountItems(ctx, fashionIDs, itemtype.SystemCatalogItem)
		if err != nil {
			return nil, err
		}

		dbItems, err := uc.wardrobeRepo.GetItemsPaginated(ctx, fashionIDs, itemtype.SystemCatalogItem, paginationQuery)
		if err != nil {
			return nil, err
		}

		fashionBriefMap := map[uuid.UUID]*fashion_dto.FashionItemBriefRes{}
		if len(fashionIDs) > 0 {
			briefs, err := uc.fashionContract.ListFashionItemsBrief(ctx, fashionIDs)
			if err == nil {
				for _, b := range briefs {
					fashionBriefMap[b.ID] = b
				}
			}
		}
		results = make([]*dto.SearchWardrobeItemRes, len(dbItems))
		for idx, item := range dbItems {
			var fiBrief *fashion_dto.FashionItemBriefRes
			if item.FashionItemID != nil {
				fiBrief = fashionBriefMap[*item.FashionItemID]
			}
			results[idx] = mapper.MapToSearchWardrobeItemRes(item, fiBrief)
		}
	}

	return &shared_dto.PaginationResult[*dto.SearchWardrobeItemRes]{
		Items:    results,
		Metadata: shared_dto.BuildPaginationMetadata(paginationQuery, totalItems),
	}, nil
}

func resolveWardrobeStatusFilter(status string) ([]wardrobestatus.WardrobeItemStatus, error) {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return nil, nil
	}

	switch normalized {
	case "all":
		return nil, nil
	case "usable", "inWardrobe":
		return []wardrobestatus.WardrobeItemStatus{wardrobestatus.InWardrobe}, nil
	case "pending":
		return []wardrobestatus.WardrobeItemStatus{
			wardrobestatus.Processing,
			wardrobestatus.Failed,
			wardrobestatus.NeedsReview,
		}, nil
	case "selling":
		return []wardrobestatus.WardrobeItemStatus{wardrobestatus.Selling}, nil
	case "sold":
		return []wardrobestatus.WardrobeItemStatus{wardrobestatus.Sold}, nil
	case "processing":
		return []wardrobestatus.WardrobeItemStatus{wardrobestatus.Processing}, nil
	case "failed":
		return []wardrobestatus.WardrobeItemStatus{wardrobestatus.Failed}, nil
	case "needsReview":
		return []wardrobestatus.WardrobeItemStatus{wardrobestatus.NeedsReview}, nil
	default:
		return nil, wardrobeerrors.ErrInvalidWardrobeStatusFilter()
	}
}

func (uc *WardrobeItemUseCase) GetWardrobeStats(ctx context.Context, userID uuid.UUID) (*dto.WardrobeStatsRes, error) {
	itemsCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	outfitsCount, err := uc.outfitRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.WardrobeStatsRes{
		ActiveItemsCount: int(itemsCount),
		OutfitsCount:     int(outfitsCount),
	}, nil
}

func (uc *WardrobeItemUseCase) GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error) {
	return uc.wardrobeRepo.GetUserWardrobeSummaryStats(ctx, userID)
}

func (uc *WardrobeItemUseCase) GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.UnderutilizedItemInfoDTO, error) {
	items, err := uc.wardrobeRepo.GetUnderutilizedItems(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return []*dto.UnderutilizedItemInfoDTO{}, nil
	}

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.FashionItemID != nil && *item.FashionItemID != uuid.Nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}

	var fashionItems []*fashion_dto.FashionItemRes
	if len(fashionIDs) > 0 && uc.fashionContract != nil {
		fashionItems, _ = uc.fashionContract.ListFashionItems(ctx, fashionIDs)
	}

	fashionMap := sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemRes) uuid.UUID {
		return f.ID
	})

	now := time.Now()
	res := make([]*dto.UnderutilizedItemInfoDTO, 0, len(items))
	for _, item := range items {
		price := 0.0
		if item.Price != nil {
			price = *item.Price
		}
		wearCount := 0
		cpw := price

		daysAgo := 30
		if item.LastUsedAt != nil {
			daysAgo = int(now.Sub(*item.LastUsedAt).Hours() / 24)
		}

		name := "Trang phục"
		categoryName := "Chưa phân loại"
		imageURL := ""

		if item.FashionItemID != nil {
			if fItem, ok := fashionMap[*item.FashionItemID]; ok && fItem != nil {
				imageURL = fItem.ImageUrl
				if fItem.Category != nil && fItem.Category.Name != "" {
					categoryName = fItem.Category.Name
				}
				if fItem.Style != nil && *fItem.Style != "" {
					name = *fItem.Style
				} else if fItem.Category != nil && fItem.Category.Name != "" {
					name = fItem.Category.Name
				}
			}
		}

		res = append(res, &dto.UnderutilizedItemInfoDTO{
			ItemID:           item.ID,
			Name:             name,
			CategoryName:     categoryName,
			PurchasePriceVND: price,
			WearCount:        wearCount,
			CostPerWearVND:   cpw,
			LastWornDaysAgo:  daysAgo,
			ImageURL:         imageURL,
		})
	}
	return res, nil
}

func (uc *WardrobeItemUseCase) GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(outfitIDs) == 0 {
		return nil, nil
	}
	return uc.outfitRepo.GetFashionItemIDsByOutfitIDs(ctx, outfitIDs)
}

func (uc *WardrobeItemUseCase) GetUserWardrobeFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return uc.wardrobeRepo.GetUserFashionItemIDs(ctx, userID)
}

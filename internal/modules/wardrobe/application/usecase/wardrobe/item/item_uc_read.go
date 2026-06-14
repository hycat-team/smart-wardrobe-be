package item

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WardrobeItemUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryRepo    repositories.ICategoryRepository
	searchEngine    search.IWardrobeSearchService
	mediaService    media.IMediaService
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	eventPublisher  event.IEventPublisher
}

func NewWardrobeItemUseCase(
	cfg *config.Config,
	l logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryRepo repositories.ICategoryRepository,
	searchEngine search.IWardrobeSearchService,
	mediaService media.IMediaService,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
) uc_interfaces.IWardrobeItemUseCase {
	return &WardrobeItemUseCase{
		cfg:             cfg,
		logger:          l,
		wardrobeRepo:    wardrobeRepo,
		categoryRepo:    categoryRepo,
		searchEngine:    searchEngine,
		mediaService:    mediaService,
		aiService:       aiService,
		userSubContract: userSubContract,
		eventPublisher:  eventPublisher,
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

func (uc *WardrobeItemUseCase) GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	var categorySlug *string
	if query.CategorySlug != "" {
		categorySlug = &query.CategorySlug
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	paginationQuery := shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	items, err := uc.wardrobeRepo.GetByUserIDPaginated(ctx, userID, categorySlug, paginationQuery)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(items))
	for idx, item := range items {
		res := mapper.MapToWardrobeItemRes(item)
		globalIdx := offset + idx
		if globalIdx >= subOverview.MaxWardrobeItems {
			res.IsLocked = true
		} else {
			res.IsLocked = false
		}
		resList[idx] = res
	}

	return &shared_dto.PaginationResult[*dto.WardrobeItemRes]{
		Items:    resList,
		Metadata: shared.BuildCurrentPageMetadata(query.PaginationQuery, len(resList)),
	}, nil
}

func (uc *WardrobeItemUseCase) GetPendingWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetPendingWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	items, err := uc.wardrobeRepo.GetPendingByUserIDPaginated(ctx, userID, query.Status, shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(items))
	for idx, item := range items {
		res := mapper.MapToWardrobeItemRes(item)
		res.IsLocked = false
		resList[idx] = res
	}

	return &shared_dto.PaginationResult[*dto.WardrobeItemRes]{
		Items:    resList,
		Metadata: shared.BuildCurrentPageMetadata(query.PaginationQuery, len(resList)),
	}, nil
}

func (uc *WardrobeItemUseCase) GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.UserID != userID {
		return nil, wardrobeerrors.ErrItemNotFound
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

	res := mapper.MapToWardrobeItemRes(item)
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

	var results []*dto.SearchWardrobeItemRes
	var err error

	searchCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	results, _, err = uc.searchEngine.SearchItems(searchCtx, query)
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

		paginationQuery := shared_dto.PaginationQuery{
			Page:  page,
			Limit: limit,
		}

		dbItems, err := uc.wardrobeRepo.GetItemsPaginated(ctx, searchQ, categorySlug, itemtype.SystemCatalogItem, paginationQuery)
		if err != nil {
			return nil, err
		}

		results = make([]*dto.SearchWardrobeItemRes, len(dbItems))
		for idx, item := range dbItems {
			results[idx] = mapper.MapToSearchWardrobeItemRes(item)
		}
	}

	return &shared_dto.PaginationResult[*dto.SearchWardrobeItemRes]{
		Items:    results,
		Metadata: shared.BuildCurrentPageMetadata(query.PaginationQuery, len(results)),
	}, nil
}

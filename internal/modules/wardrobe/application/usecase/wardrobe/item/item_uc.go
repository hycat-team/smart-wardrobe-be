package item

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/colorutils"

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

	totalItems, err := uc.wardrobeRepo.CountByUserIDAndCategory(ctx, userID, categorySlug)
	if err != nil {
		return nil, err
	}

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
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, totalItems),
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

	isLocked := false
	for idx, it := range items {
		if it.ID == id {
			if idx >= subOverview.MaxWardrobeItems {
				isLocked = true
			}
			break
		}
	}

	if isLocked {
		return nil, wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
	}

	res := mapper.MapToWardrobeItemRes(item)
	res.IsLocked = false
	return res, nil
}

func (uc *WardrobeItemUseCase) CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error) {
	if quantity < 1 || quantity > 5 {
		return nil, wardrobeerrors.ErrInvalidCloneQuantity
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if int(currentCount)+quantity > subOverview.MaxWardrobeItems {
		return nil, wardrobeerrors.ErrWardrobeLimitExceededForClone(int(currentCount), subOverview.MaxWardrobeItems, quantity)
	}

	original, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, wardrobeerrors.ErrOriginalItemToCloneNotFound
	}

	if original.UserID != userID {
		return nil, wardrobeerrors.ErrCloneOtherUserItemForbidden
	}

	if original.Status == wardrobestatus.Sold {
		return nil, wardrobeerrors.ErrCloneSoldItem
	}

	clonedItems := make([]*entities.WardrobeItem, quantity)
	for i := range quantity {
		clonedItems[i] = &entities.WardrobeItem{
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
			ItemType:      itemtype.UserItem,
		}
	}

	err = uc.wardrobeRepo.BulkCreate(ctx, clonedItems)
	if err != nil {
		return nil, err
	}

	for _, cloned := range clonedItems {
		payload := dto.WardrobeEventPayload{
			ItemID: cloned.ID,
			UserID: cloned.UserID,
			Action: "created",
		}
		_ = uc.eventPublisher.Publish(ctx, "wardrobe.event.created", payload)
	}

	resList := make([]*dto.WardrobeItemRes, quantity)
	for i := range quantity {
		clonedItems[i].Category = original.Category
		resList[i] = mapper.MapToWardrobeItemRes(clonedItems[i])
		resList[i].IsLocked = false
	}

	return resList, nil
}

func (uc *WardrobeItemUseCase) SearchWardrobeItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.SearchWardrobeItemRes], error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	var results []*dto.SearchWardrobeItemRes
	var totalItems int64
	var err error

	searchCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	results, totalItems, err = uc.searchEngine.SearchItems(searchCtx, query)
	if err != nil {
		uc.logger.Warn("[SearchWardrobeItems] Search failed or timed out, falling back to database", zap.Error(err))
		var searchQ *string
		if query.Query != "" {
			searchQ = &query.Query
		}
		var categorySlug *string
		if query.CategorySlug != "" {
			categorySlug = &query.CategorySlug
		}

		totalItems, err = uc.wardrobeRepo.CountItems(ctx, searchQ, categorySlug, itemtype.SystemCatalogItem)
		if err != nil {
			return nil, err
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
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, totalItems),
	}, nil
}

func (uc *WardrobeItemUseCase) ManualClassify(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.ManualClassifyReq) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, wardrobeerrors.ErrItemNotFound
	}

	if item.UserID != userID {
		return nil, wardrobeerrors.ErrUpdateItemForbidden
	}

	if item.Status == wardrobestatus.Sold {
		return nil, wardrobeerrors.ErrManualClassifySoldItem
	}

	category, err := uc.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound
	}

	tokens := fmt.Sprintf("[CAT:%s][COL:%s][STY:%s][MAT:%s][PAT:%s][FIT:%s][SEA:%s]",
		category.Slug, input.Color, input.Style, input.Material, input.Pattern, input.Fit, input.Seasonality)

	freeForm := fmt.Sprintf("Món đồ thời trang %s màu %s phong cách %s được làm từ %s với họa tiết %s, dáng %s thích hợp mặc vào %s.",
		category.Name, input.Color, input.Style, input.Material, input.Pattern, input.Fit, input.Seasonality)

	description := tokens + " " + freeForm

	richTextContext := fmt.Sprintf(
		"Danh mục trang phục: %s, Thuộc tính màu sắc: %s, Định hình phong cách thiết kế: %s, Chất liệu: %s, Họa tiết: %s, Kiểu dáng: %s, Mùa phù hợp: %s. Mô tả chi tiết: %s",
		category.Name,
		input.Color,
		input.Style,
		input.Material,
		input.Pattern,
		input.Fit,
		input.Seasonality,
		freeForm,
	)

	embeddings, err := uc.aiService.GenerateEmbeddings(ctx, []string{richTextContext})
	if err != nil || len(embeddings) == 0 {
		return nil, wardrobeerrors.ErrProcessFashionTextFailed
	}
	embedding := embeddings[0]

	item.CategoryID = &category.ID
	item.Color = &input.Color
	item.Style = &input.Style
	item.Material = &input.Material
	item.Pattern = &input.Pattern
	item.Fit = &input.Fit
	item.Seasonality = &input.Seasonality
	item.Description = &description
	item.Price = input.Price
	item.Embedding = entities.Vector(embedding)
	item.Status = wardrobestatus.InWardrobe

	if h, s, l, hex, ok := colorutils.ResolveHSLFromColorName(input.Color); ok {
		item.ColorHex = &hex
		item.ColorHue = &h
		item.ColorSaturation = &s
		item.ColorLightness = &l
	}

	err = uc.wardrobeRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	payload := dto.WardrobeEventPayload{
		ItemID: item.ID,
		UserID: item.UserID,
		Action: eventconstants.ActionCreated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)

	return mapper.MapToWardrobeItemRes(item), nil
}

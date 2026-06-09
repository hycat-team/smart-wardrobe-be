package wardrobe

import (
	"context"
	"math"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"

	"go.uber.org/zap"
)

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

	// Set a short timeout (300ms) for the Elasticsearch query.
	// If Elasticsearch is down or slow, it will quickly fail (max 300ms for the first request, 0ms subsequently due to circuit breaker) and fallback to PostgreSQL.
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

	totalPages := 0
	if limit > 0 && totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	}

	return &shared_dto.PaginationResult[*dto.SearchWardrobeItemRes]{
		Items: results,
		Metadata: shared_dto.PaginationMetadata{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

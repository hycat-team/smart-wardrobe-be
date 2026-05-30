package wardrobe

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"

	"go.uber.org/zap"
)

func (uc *WardrobeUseCase) SearchWardrobeItems(ctx context.Context, query string) ([]*dto.SearchWardrobeItemRes, error) {
	respBytes, err := uc.searchEngine.SearchItems(ctx, query)
	if err != nil {
		uc.logger.Warn("[SearchWardrobeItems] Search failed, fallback to Repository", zap.Error(err))
		items, err := uc.wardrobeRepo.GetItems(ctx, &query, itemtype.SystemCatalogItem)
		if err != nil {
			return nil, err
		}

		respBytes = make([]*dto.SearchWardrobeItemRes, len(items))
		for idx, item := range items {
			respBytes[idx] = mapper.MapToSearchWardrobeItemRes(item)
		}
	}

	return respBytes, nil
}

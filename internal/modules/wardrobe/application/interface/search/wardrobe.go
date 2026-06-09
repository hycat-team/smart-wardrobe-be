package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

type IWardrobeSearchService interface {
	SearchItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) ([]*dto.SearchWardrobeItemRes, int64, error)
	IsHealthy() bool
}

type IWardrobeSearchIndexService interface {
	IndexItem(ctx context.Context, item *entities.WardrobeItem) error
	DeleteItem(ctx context.Context, itemID string) error
	IsHealthy() bool
}

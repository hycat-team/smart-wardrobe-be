package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type IWardrobeSearchService interface {
	SearchItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) ([]*dto.SearchWardrobeItemRes, int64, error)
	IsHealthy() bool
}

type IWardrobeSearchIndexService interface {
	IndexItem(ctx context.Context, item *dto.SearchDocumentDTO) error
	DeleteItem(ctx context.Context, itemID string) error
	IsHealthy() bool
}

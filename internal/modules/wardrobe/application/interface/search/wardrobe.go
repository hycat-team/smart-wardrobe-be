package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type IWardrobeSearchService interface {
	SearchItems(ctx context.Context, query string) ([]*dto.SearchWardrobeItemRes, error)
}

package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

func NewWardrobeSearchIndexService(searchEngine *shared_search.ElasticsearchClient, l logger.Interface) search.IWardrobeSearchIndexService {
	return &WardrobeSearchService{searchEngine: searchEngine, logger: l}
}

func (s *WardrobeSearchService) IndexItem(ctx context.Context, item *entities.WardrobeItem) error {
	doc := buildItemDocument(item)
	return s.searchEngine.IndexDocument(ctx, "wardrobe_items", item.ID.String(), doc)
}

func (s *WardrobeSearchService) DeleteItem(ctx context.Context, itemID string) error {
	return s.searchEngine.DeleteDocument(ctx, "wardrobe_items", itemID)
}

func buildItemDocument(item *entities.WardrobeItem) map[string]any {
	return map[string]any{
		"id":        item.ID.String(),
		"user_id":   item.UserID.String(),
		"item_type": int(item.ItemType),
		"category": map[string]any{
			"id":   item.CategoryID.String(),
			"name": item.Category.Name,
			"slug": item.Category.Slug,
		},
		"image_url":       item.ImageUrl,
		"image_public_id": item.ImagePublicID,
		"color":           stringutils.GetString(item.Color),
		"style":           stringutils.GetString(item.Style),
		"material":        stringutils.GetString(item.Material),
		"pattern":         stringutils.GetString(item.Pattern),
		"fit":             stringutils.GetString(item.Fit),
		"seasonality":     stringutils.GetString(item.Seasonality),
		"description":     stringutils.GetString(item.Description),
		"status":          item.Status,
		"created_at":      item.CreatedAt,
	}
}

package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

func NewWardrobeSearchIndexService(searchEngine *shared_search.ElasticsearchClient) search.IWardrobeSearchIndexService {
	return &WardrobeSearchService{searchEngine: searchEngine}
}

func (s *WardrobeSearchService) IndexItem(ctx context.Context, item *entities.WardrobeItem) error {
	doc := buildItemDocument(item)
	return s.searchEngine.IndexDocument(ctx, "wardrobe_items", item.ID.String(), doc)
}

func (s *WardrobeSearchService) DeleteItem(ctx context.Context, itemID string) error {
	return s.searchEngine.DeleteDocument(ctx, "wardrobe_items", itemID)
}

func buildItemDocument(item *entities.WardrobeItem) map[string]any {
	doc := map[string]any{
		"id":              item.ID.String(),
		"user_id":         item.UserID.String(),
		"item_type":       int(item.ItemType),
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

	var categoryIDStr string
	if item.CategoryID != nil {
		categoryIDStr = item.CategoryID.String()
	}

	if item.Category != nil {
		doc["category"] = map[string]any{
			"id":   categoryIDStr,
			"name": item.Category.Name,
			"slug": item.Category.Slug,
		}
	} else {
		doc["category"] = map[string]any{
			"id":   categoryIDStr,
			"name": "",
			"slug": "",
		}
	}

	if item.Price != nil {
		doc["price"] = *item.Price
	}
	if item.ColorHex != nil {
		doc["color_hex"] = *item.ColorHex
	}
	if item.ColorHue != nil {
		doc["color_hue"] = *item.ColorHue
	}
	if item.ColorSaturation != nil {
		doc["color_saturation"] = *item.ColorSaturation
	}
	if item.ColorLightness != nil {
		doc["color_lightness"] = *item.ColorLightness
	}

	return doc
}

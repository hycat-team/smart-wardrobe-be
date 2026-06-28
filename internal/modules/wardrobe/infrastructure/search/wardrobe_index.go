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
	fashion := item.FashionItem
	if fashion == nil {
		fashion = &entities.FashionItem{}
	}
	doc := map[string]any{
		"id":              item.ID.String(),
		"user_id":         item.UserID.String(),
		"fashion_item_id": item.FashionItemID.String(),
		"item_type":       int(item.ItemType),
		"image_url":       fashion.ImageUrl,
		"image_public_id": fashion.ImagePublicID,
		"color":           stringutils.GetString(fashion.Color),
		"style":           stringutils.GetString(fashion.Style),
		"material":        stringutils.GetString(fashion.Material),
		"pattern":         stringutils.GetString(fashion.Pattern),
		"fit":             stringutils.GetString(fashion.Fit),
		"seasonality":     stringutils.GetString(fashion.Seasonality),
		"description":     stringutils.GetString(fashion.Description),
		"status":          item.Status,
		"created_at":      item.CreatedAt,
	}

	var categoryIDStr string
	if fashion.CategoryID != nil {
		categoryIDStr = fashion.CategoryID.String()
	}

	if fashion.Category != nil {
		doc["category"] = map[string]any{
			"id":   categoryIDStr,
			"name": fashion.Category.Name,
			"slug": fashion.Category.Slug,
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
	if fashion.ColorHex != nil {
		doc["color_hex"] = *fashion.ColorHex
	}
	if fashion.ColorHue != nil {
		doc["color_hue"] = *fashion.ColorHue
	}
	if fashion.ColorSaturation != nil {
		doc["color_saturation"] = *fashion.ColorSaturation
	}
	if fashion.ColorLightness != nil {
		doc["color_lightness"] = *fashion.ColorLightness
	}

	return doc
}

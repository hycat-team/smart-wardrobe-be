package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/itemtype"
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
)

func NewWardrobeSearchIndexService(searchEngine *shared_search.ElasticsearchClient) search.IWardrobeSearchIndexService {
	return &WardrobeSearchService{searchEngine: searchEngine}
}

func (s *WardrobeSearchService) IndexItem(ctx context.Context, item *dto.SearchDocumentDTO) error {
	if item.ItemType != itemtype.SystemCatalogItem {
		return nil
	}
	doc := buildItemDocument(item)
	return s.searchEngine.IndexDocument(ctx, "wardrobe_items", item.FashionItemID.String(), doc)
}

func (s *WardrobeSearchService) DeleteItem(ctx context.Context, itemID string) error {
	return s.searchEngine.DeleteDocument(ctx, "wardrobe_items", itemID)
}

func buildItemDocument(item *dto.SearchDocumentDTO) map[string]any {
	doc := map[string]any{
		"id":              item.ID.String(),
		"user_id":         item.UserID.String(),
		"fashion_item_id": item.FashionItemID.String(),
		"item_type":       int(item.ItemType),
		"image_url":       item.ImageUrl,
		"image_public_id": item.ImagePublicID,
		"color":           item.Color,
		"style":           item.Style,
		"material":        item.Material,
		"pattern":         item.Pattern,
		"fit":             item.Fit,
		"seasonality":     item.Seasonality,
		"description":     item.Description,
		"status":          item.Status,
		"created_at":      item.CreatedAt,
	}

	doc["category"] = map[string]any{
		"id":   item.Category.ID,
		"name": item.Category.Name,
		"slug": item.Category.Slug,
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

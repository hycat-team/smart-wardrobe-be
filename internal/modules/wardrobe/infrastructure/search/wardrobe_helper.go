package search

import (
	"encoding/json"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"time"

	"github.com/google/uuid"
)

// parseSearchWardrobeItemRes parses raw JSON from Elasticsearch to a slice of SearchWardrobeItemRes and total hits
func (s *WardrobeSearchService) parseSearchWardrobeItemRes(respBytes []byte) ([]*dto.SearchWardrobeItemRes, int64, error) {
	type esDocSource struct {
		ID            string `json:"id"`
		FashionItemID string `json:"fashion_item_id"`
		ItemType      int    `json:"item_type"`
		Category      struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"category"`
		ImageUrl         string    `json:"image_url"`
		ImagePublicID    string    `json:"image_public_id"`
		Color            string    `json:"color"`
		ColorHex         *string   `json:"color_hex,omitempty"`
		ColorHue         *float64  `json:"color_hue,omitempty"`
		ColorSaturation  *float64  `json:"color_saturation,omitempty"`
		ColorLightness   *float64  `json:"color_lightness,omitempty"`
		Style            string    `json:"style"`
		Material         string    `json:"material"`
		Pattern          string    `json:"pattern"`
		Fit              string    `json:"fit"`
		Seasonality      string    `json:"seasonality"`
		Description      string    `json:"description"`
		Price            *float64  `json:"price,omitempty"`
		FashionCreatedAt time.Time `json:"fashion_created_at"`
		FashionUpdatedAt time.Time `json:"fashion_updated_at"`
	}

	var esResult struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source esDocSource `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(respBytes, &esResult); err != nil {
		return nil, 0, wardrobeerrors.ErrSearchItemsFailed()
	}

	results := make([]*dto.SearchWardrobeItemRes, len(esResult.Hits.Hits))
	for idx, hit := range esResult.Hits.Hits {
		id, _ := uuid.Parse(hit.Source.ID)
		catID, _ := uuid.Parse(hit.Source.Category.ID)
		fiID, _ := uuid.Parse(hit.Source.FashionItemID)

		strPtr := func(s string) *string {
			if s == "" {
				return nil
			}
			return &s
		}

		var category *dto.CategoryBriefRes
		if hit.Source.Category.ID != "" && hit.Source.Category.ID != uuid.Nil.String() {
			category = &dto.CategoryBriefRes{
				ID:   catID,
				Name: hit.Source.Category.Name,
				Slug: hit.Source.Category.Slug,
			}
		}

		fashionItem := &dto.FashionItemRes{
			ID:              fiID,
			Category:        category,
			ImageUrl:        hit.Source.ImageUrl,
			Color:           strPtr(hit.Source.Color),
			ColorHex:        hit.Source.ColorHex,
			ColorHue:        hit.Source.ColorHue,
			ColorSaturation: hit.Source.ColorSaturation,
			ColorLightness:  hit.Source.ColorLightness,
			Style:           strPtr(hit.Source.Style),
			Material:        strPtr(hit.Source.Material),
			Pattern:         strPtr(hit.Source.Pattern),
			Fit:             strPtr(hit.Source.Fit),
			Seasonality:     strPtr(hit.Source.Seasonality),
			Description:     strPtr(hit.Source.Description),
			CreatedAt:       hit.Source.FashionCreatedAt,
			UpdatedAt:       hit.Source.FashionUpdatedAt,
		}

		results[idx] = &dto.SearchWardrobeItemRes{
			ID:          id,
			Price:       hit.Source.Price,
			IsSystem:    hit.Source.ItemType == int(itemtype.SystemCatalogItem),
			FashionItem: fashionItem,
		}
	}

	return results, esResult.Hits.Total.Value, nil
}

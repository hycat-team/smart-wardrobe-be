package search

import (
	"encoding/json"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// parseSearchWardrobeItemRes parses raw JSON from Elasticsearch to a slice of SearchWardrobeItemRes and total hits
func (s *WardrobeSearchService) parseSearchWardrobeItemRes(respBytes []byte) ([]*dto.SearchWardrobeItemRes, int64, error) {
	type esDocSource struct {
		ID       string `json:"id"`
		ItemType int    `json:"item_type"`
		Category struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"category"`
		ImageUrl        string   `json:"image_url"`
		ImagePublicID   string   `json:"image_public_id"`
		Color           string   `json:"color"`
		ColorHex        *string  `json:"color_hex,omitempty"`
		ColorHue        *float64 `json:"color_hue,omitempty"`
		ColorSaturation *float64 `json:"color_saturation,omitempty"`
		ColorLightness  *float64 `json:"color_lightness,omitempty"`
		Style           string   `json:"style"`
		Material        string   `json:"material"`
		Pattern         string   `json:"pattern"`
		Fit             string   `json:"fit"`
		Seasonality     string   `json:"seasonality"`
		Description     string   `json:"description"`
		Price           *float64 `json:"price,omitempty"`
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
		return nil, 0, wardrobeerrors.ErrSearchItemsFailed
	}

	results := make([]*dto.SearchWardrobeItemRes, len(esResult.Hits.Hits))
	for idx, hit := range esResult.Hits.Hits {
		id, _ := uuid.Parse(hit.Source.ID)
		catID, _ := uuid.Parse(hit.Source.Category.ID)

		var colorHex string
		if hit.Source.ColorHex != nil {
			colorHex = *hit.Source.ColorHex
		}

		results[idx] = &dto.SearchWardrobeItemRes{
			ID: id,
			Category: &dto.CategoryRes{
				ID:   catID,
				Name: hit.Source.Category.Name,
				Slug: hit.Source.Category.Slug,
			},
			ImageUrl:        hit.Source.ImageUrl,
			ImagePublicID:   hit.Source.ImagePublicID,
			Color:           hit.Source.Color,
			ColorHex:        colorHex,
			ColorHue:        hit.Source.ColorHue,
			ColorSaturation: hit.Source.ColorSaturation,
			ColorLightness:  hit.Source.ColorLightness,
			Style:           hit.Source.Style,
			Material:        hit.Source.Material,
			Pattern:         hit.Source.Pattern,
			Fit:             hit.Source.Fit,
			Seasonality:     hit.Source.Seasonality,
			Price:           hit.Source.Price,
			IsSystem:        hit.Source.ItemType == 1,
		}
	}

	return results, esResult.Hits.Total.Value, nil
}


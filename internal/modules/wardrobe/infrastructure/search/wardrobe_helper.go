package search

import (
	"encoding/json"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"

	"github.com/google/uuid"
)

// parseEsResToSearchRes parse JSON thô từ Elasticsearch sang danh sách SearchWardrobeItemRes
func (s *WardrobeSearchService) parseSearchWardrobeItemRes(respBytes []byte) ([]*dto.SearchWardrobeItemRes, error) {
	type esDocSource struct {
		ID       string `json:"id"`
		ItemType int    `json:"item_type"`
		Category struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"category"`
		ImageUrl      string `json:"image_url"`
		ImagePublicID string `json:"image_public_id"`
		Color         string `json:"color"`
		Style         string `json:"style"`
		Material      string `json:"material"`
		Pattern       string `json:"pattern"`
		Fit           string `json:"fit"`
		Seasonality   string `json:"seasonality"`
		Description   string `json:"description"`
	}

	var esResult struct {
		Hits struct {
			Hits []struct {
				Source esDocSource `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(respBytes, &esResult); err != nil {
		return nil, apperror.NewInternalError("Failed to unmarshal raw search response")
	}

	results := make([]*dto.SearchWardrobeItemRes, len(esResult.Hits.Hits))
	for idx, hit := range esResult.Hits.Hits {
		id, _ := uuid.Parse(hit.Source.ID)
		catID, _ := uuid.Parse(hit.Source.Category.ID)

		results[idx] = &dto.SearchWardrobeItemRes{
			ID: id,
			Category: &dto.CategoryRes{
				ID:   catID,
				Name: hit.Source.Category.Name,
				Slug: hit.Source.Category.Slug,
			},
			ImageUrl:      hit.Source.ImageUrl,
			ImagePublicID: hit.Source.ImagePublicID,
			Color:         hit.Source.Color,
			Style:         hit.Source.Style,
			Material:      hit.Source.Material,
			Pattern:       hit.Source.Pattern,
			Fit:           hit.Source.Fit,
			Seasonality:   hit.Source.Seasonality,
			// Description:   hit.Source.Description,
			IsSystem: hit.Source.ItemType == 1,
		}
	}

	return results, nil
}


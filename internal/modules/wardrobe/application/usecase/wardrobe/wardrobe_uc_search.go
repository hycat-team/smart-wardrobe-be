package wardrobe

import (
	"context"
	"encoding/json"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (uc *WardrobeUseCase) SearchWardrobeItems(ctx context.Context, query string) ([]dto.SearchWardrobeItemRes, error) {
	// 1. Xây dựng Elasticsearch query body lọc theo đồ hệ thống (item_type = 1)
	var esQuery map[string]interface{}

	if query == "" {
		// Nếu query rỗng, trả về match_all kèm lọc đồ hệ thống (item_type = 1)
		esQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"item_type": 1,
							},
						},
					},
				},
			},
			"size": 50,
		}
	} else {
		// Ngược lại, thực hiện multi_match kèm fuzzy matching và lọc đồ hệ thống (item_type = 1)
		esQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"multi_match": map[string]interface{}{
								"query": query,
								"fields": []string{
									"category.name^3",
									"category.name._2gram^3",
									"category.name._3gram^3",
									"color^2",
									"style^2",
									"material",
									"pattern",
									"fit",
									"seasonality",
									"description",
								},
								"type":      "bool_prefix",
								"fuzziness": "AUTO", // Hỗ trợ gõ sai chính tả
							},
						},
					},
					"filter": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"item_type": 1,
							},
						},
					},
				},
			},
			"size": 50,
		}
	}

	// 2. Thực thi lệnh search qua client
	respBytes, err := uc.esClient.Search(ctx, "wardrobe_items", esQuery)
	if err != nil {
		// Index chưa tồn tại (chưa có tài liệu nào được index lần đầu) — trả về mảng rỗng
		if strings.Contains(err.Error(), "status code 404") {
			return []dto.SearchWardrobeItemRes{}, nil
		}
		uc.logger.Error("Elasticsearch search query execution failed", zap.Error(err))
		return nil, err
	}

	// 3. Định nghĩa struct cấu trúc tài liệu của Elasticsearch để parse dữ liệu
	type esDocSource struct {
		ID       string `json:"id"`
		ItemType int    `json:"item_type"`
		Category struct {
			ID string `json:"id"`
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
		uc.logger.Error("Failed to unmarshal Elasticsearch raw search response", zap.Error(err))
		return nil, err
	}

	// 4. Trích xuất và map sang danh sách SearchWardrobeItemRes
	results := make([]dto.SearchWardrobeItemRes, len(esResult.Hits.Hits))
	for idx, hit := range esResult.Hits.Hits {
		id, _ := uuid.Parse(hit.Source.ID)
		catID, _ := uuid.Parse(hit.Source.Category.ID)

		results[idx] = dto.SearchWardrobeItemRes{
			ID:            id,
			CategoryID:    catID,
			ImageUrl:      hit.Source.ImageUrl,
			ImagePublicID: hit.Source.ImagePublicID,
			Color:         hit.Source.Color,
			Style:         hit.Source.Style,
			Material:      hit.Source.Material,
			Pattern:       hit.Source.Pattern,
			Fit:           hit.Source.Fit,
			Seasonality:   hit.Source.Seasonality,
			Description:   hit.Source.Description,
			IsSystem:      hit.Source.ItemType == 1,
		}
	}

	return results, nil
}

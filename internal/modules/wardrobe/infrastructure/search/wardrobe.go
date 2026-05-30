package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
	"smart-wardrobe-be/pkg/logger"
	"strings"
)

type WardrobeSearchService struct {
	searchEngine *shared_search.ElasticsearchClient
	logger       logger.Interface
}

func NewWardrobeSearchService(searchEngine *shared_search.ElasticsearchClient, l logger.Interface) search.IWardrobeSearchService {
	return &WardrobeSearchService{searchEngine: searchEngine, logger: l}
}

func (s *WardrobeSearchService) SearchItems(ctx context.Context, query string) ([]*dto.SearchWardrobeItemRes, error) {
	// 1. Xây dựng Elasticsearch query body lọc theo đồ hệ thống (item_type = 1)
	var esQuery map[string]any

	if query == "" {
		// Nếu query rỗng, trả về match_all kèm lọc đồ hệ thống (item_type = 1)
		esQuery = map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"filter": []map[string]any{
						{
							"term": map[string]any{
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
		esQuery = map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"must": []map[string]any{
						{
							"multi_match": map[string]any{
								"query": query,
								"fields": []string{
									"category.name^3",
									// "category.name._2gram^3",
									// "category.name._3gram^3",
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
					"filter": []map[string]any{
						{
							"term": map[string]any{
								"item_type": 1,
							},
						},
					},
				},
			},
			"size": 50,
		}
	}

	// 2. Thực thi lệnh search qua Elasticsearch
	respBytes, err := s.searchEngine.Search(ctx, "wardrobe_items", esQuery)
	if err != nil {
		// Index chưa tồn tại — trả về mảng rỗng
		if strings.Contains(err.Error(), "status code 404") {
			return []*dto.SearchWardrobeItemRes{}, nil
		}
	}

	// 3. Parse kết quả từ Elasticsearch
	return s.parseElasticsearchResults(respBytes)
}

package search

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
)

func NewWardrobeSearchService(searchEngine *shared_search.ElasticsearchClient) search.IWardrobeSearchService {
	return &WardrobeSearchService{searchEngine: searchEngine}
}

func (s *WardrobeSearchService) SearchItems(ctx context.Context, query string) ([]*dto.SearchWardrobeItemRes, error) {
	var esQuery map[string]any

	if query == "" {
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
		esQuery = map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"must": []map[string]any{
						{
							"multi_match": map[string]any{
								"query": query,
								"fields": []string{
									"category.name^3",
									"color^2",
									"style^2",
									"material",
									"pattern",
									"fit",
									"seasonality",
									"description",
								},
								"type":      "bool_prefix",
								"fuzziness": "AUTO",
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

	respBytes, err := s.searchEngine.Search(ctx, "wardrobe_items", esQuery)
	if err != nil {
		if appErr := apperror.From(err); appErr != nil {
			notFound := apperror.ErrSearchIndexNotFound()
			if appErr.Status == notFound.Status && appErr.Detail == notFound.Detail {
				return []*dto.SearchWardrobeItemRes{}, nil
			}
		}
		return nil, err
	}

	return s.parseSearchWardrobeItemRes(respBytes)
}

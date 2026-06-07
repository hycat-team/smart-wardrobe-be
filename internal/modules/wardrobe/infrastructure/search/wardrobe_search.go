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

func (s *WardrobeSearchService) SearchItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) ([]*dto.SearchWardrobeItemRes, error) {
	var esQuery map[string]any
	filters := []map[string]any{
		{
			"term": map[string]any{
				"item_type": 1,
			},
		},
	}
	if query.CategorySlug != "" {
		filters = append(filters, map[string]any{
			"term": map[string]any{
				"category.slug.keyword": query.CategorySlug,
			},
		})
	}

	if query.Query == "" {
		esQuery = map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"filter": filters,
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
								"query": query.Query,
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
					"filter": filters,
				},
			},
			"size": 50,
		}
	}

	respBytes, err := s.searchEngine.Search(ctx, "wardrobe_items", esQuery)
	if err != nil {
		if appErr := apperror.From(err); appErr != nil {
			notFound := apperror.ErrSearchIndexNotFound()
			if appErr.Status == notFound.Status && appErr.Message == notFound.Message {
				return []*dto.SearchWardrobeItemRes{}, nil
			}
		}
		return nil, err
	}

	return s.parseSearchWardrobeItemRes(respBytes)
}

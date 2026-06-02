package search

import (
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
)

type WardrobeSearchService struct {
	searchEngine *shared_search.ElasticsearchClient
}

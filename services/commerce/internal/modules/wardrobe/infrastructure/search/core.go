package search

import (
	shared_search "smart-wardrobe-be/shared/infrastructure/search"
)

type WardrobeSearchService struct {
	searchEngine *shared_search.ElasticsearchClient
}

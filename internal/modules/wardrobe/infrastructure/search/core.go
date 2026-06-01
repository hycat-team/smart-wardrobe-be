package search

import (
	shared_search "smart-wardrobe-be/internal/shared/infrastructure/search"
	"smart-wardrobe-be/pkg/logger"
)

type WardrobeSearchService struct {
	searchEngine *shared_search.ElasticsearchClient
	logger       logger.Interface
}

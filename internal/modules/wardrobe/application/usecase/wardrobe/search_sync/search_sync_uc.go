package search_sync

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type SearchSyncUseCase struct {
	searchIndex  search.IWardrobeSearchIndexService
	wardrobeRepo repositories.IWardrobeItemRepository
	logger       logger.Interface
}

func NewSearchSyncUseCase(
	searchIndex search.IWardrobeSearchIndexService,
	wardrobeRepo repositories.IWardrobeItemRepository,
	l logger.Interface,
) usecase_interfaces.ISearchSyncUseCase {
	return &SearchSyncUseCase{
		searchIndex:  searchIndex,
		wardrobeRepo: wardrobeRepo,
		logger:       l,
	}
}

func (uc *SearchSyncUseCase) ProcessSyncEvent(ctx context.Context, eventPayload dto.WardrobeEventPayload) error {
	switch eventPayload.Action {
	case eventconstants.ActionCreated, eventconstants.ActionUpdated:
		item, err := uc.wardrobeRepo.GetByID(ctx, eventPayload.ItemID)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("item not found in database for indexing")
		}

		// Only sync system wardrobe items (SystemCatalogItem = 1) to the Search Index
		if item.ItemType != 1 {
			return nil
		}

		if err := uc.searchIndex.IndexItem(ctx, item); err != nil {
			uc.logger.Warn("[SearchSyncUseCase] Failed to index item in search index (Elasticsearch may be offline)",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			if !uc.searchIndex.IsHealthy() {
				return fmt.Errorf("elasticsearch is offline: %w", err)
			}
			return nil
		}

	case eventconstants.ActionDeleted:
		if err := uc.searchIndex.DeleteItem(ctx, eventPayload.ItemID.String()); err != nil {
			uc.logger.Warn("[SearchSyncUseCase] Failed to delete item from search index (Elasticsearch may be offline)",
				zap.String("itemId", eventPayload.ItemID.String()),
				zap.Error(err),
			)
			if !uc.searchIndex.IsHealthy() {
				return fmt.Errorf("elasticsearch is offline: %w", err)
			}
			return nil
		}
	}

	return nil
}

func (uc *SearchSyncUseCase) TryInitialSync(ctx context.Context) (bool, error) {
	if !uc.searchIndex.IsHealthy() {
		return false, nil
	}

	items, err := uc.wardrobeRepo.GetItems(ctx, nil, nil, itemtype.SystemCatalogItem)
	if err != nil {
		uc.logger.Error("[SearchSyncUseCase] Failed to fetch system catalog items for initial sync", zap.Error(err))
		return false, err
	}

	if len(items) == 0 {
		uc.logger.Info("[SearchSyncUseCase] Initial sync succeeded", zap.Int("indexed", 0))
		return true, nil
	}

	successCount := 0
	for _, item := range items {
		if err := uc.searchIndex.IndexItem(ctx, item); err != nil {
			uc.logger.Warn("[SearchSyncUseCase] Failed to index system catalog item during initial sync. Elasticsearch might have gone offline.",
				zap.Error(err),
			)
			return false, err
		}
		successCount++
	}

	uc.logger.Info("[SearchSyncUseCase] Initial sync succeeded",
		zap.Int("indexed", successCount),
	)
	return true, nil
}

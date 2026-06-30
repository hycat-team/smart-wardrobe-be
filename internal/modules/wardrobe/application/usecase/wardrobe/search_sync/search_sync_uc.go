package search_sync

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/itemtype"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
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

func (uc *SearchSyncUseCase) ProcessSyncEvent(ctx context.Context, eventPayload dto.WardrobeEventPayload, run *workerlog.Run) error {
	run.AddTotal(1)
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
			run.AddSkipped(1)
			return nil
		}

		doc := mapper.MapToSearchDocumentDTO(item)
		if err := uc.searchIndex.IndexItem(ctx, doc); err != nil {
			run.ChildWarn(uc.logger, "Failed to index item in search index",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			if !uc.searchIndex.IsHealthy() {
				run.AddRetry(1)
				return fmt.Errorf("elasticsearch is offline: %w", err)
			}
			return nil
		}
		run.AddSuccess(1)

	case eventconstants.ActionDeleted:
		if eventPayload.ItemType != int(itemtype.SystemCatalogItem) {
			run.AddSkipped(1)
			return nil
		}
		documentID := eventPayload.FashionItemID
		if documentID == uuid.Nil {
			documentID = eventPayload.ItemID
		}
		if err := uc.searchIndex.DeleteItem(ctx, documentID.String()); err != nil {
			run.ChildWarn(uc.logger, "Failed to delete item from search index",
				zap.String("itemId", eventPayload.ItemID.String()),
				zap.String("fashionItemId", documentID.String()),
				zap.Error(err),
			)
			if !uc.searchIndex.IsHealthy() {
				run.AddRetry(1)
				return fmt.Errorf("elasticsearch is offline: %w", err)
			}
			return nil
		}
		run.AddSuccess(1)
	}

	return nil
}

func (uc *SearchSyncUseCase) TryInitialSync(ctx context.Context, run *workerlog.Run) (bool, error) {
	if !uc.searchIndex.IsHealthy() {
		run.AddSkipped(1)
		return false, nil
	}

	items, err := uc.wardrobeRepo.GetItems(ctx, nil, nil, itemtype.SystemCatalogItem)
	if err != nil {
		run.ChildError(uc.logger, "Failed to fetch system catalog items for initial sync", zap.Error(err))
		return false, err
	}
	run.AddTotal(len(items))

	if len(items) == 0 {
		run.AddSummaryFields(zap.Int("indexedCount", 0))
		return true, nil
	}

	successCount := 0
	for _, item := range items {
		doc := mapper.MapToSearchDocumentDTO(item)
		if err := uc.searchIndex.IndexItem(ctx, doc); err != nil {
			run.ChildWarn(uc.logger, "Failed to index system catalog item during initial sync",
				zap.Error(err),
			)
			return false, err
		}
		successCount++
	}
	run.AddSuccess(successCount)
	run.AddSummaryFields(zap.Int("indexedCount", successCount))
	return true, nil
}

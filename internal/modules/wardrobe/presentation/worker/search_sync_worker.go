package worker

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type SearchSyncWorker struct {
	eventConsumer event.ISearchSyncEventConsumer
	searchIndex   search.IWardrobeSearchIndexService
	wardrobeRepo  repositories.IWardrobeItemRepository
	logger        logger.Interface
}

func NewSearchSyncWorker(
	eventConsumer event.ISearchSyncEventConsumer,
	searchIndex search.IWardrobeSearchIndexService,
	wardrobeRepo repositories.IWardrobeItemRepository,
	l logger.Interface,
) *SearchSyncWorker {
	w := &SearchSyncWorker{
		eventConsumer: eventConsumer,
		searchIndex:   searchIndex,
		wardrobeRepo:  wardrobeRepo,
		logger:        l,
	}

	// Initial sync: push all System Catalog Items from PostgreSQL to the search engine
	go w.initialSync()

	// Start listening to the sync event queue via the Application layer Consumer
	go w.startConsume()

	return w
}

func (w *SearchSyncWorker) startConsume() {
	ctx := context.Background()
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, eventPayload dto.WardrobeEventPayload) error {
		return w.processSyncEvent(ctx, eventPayload)
	})

	if err != nil {
		w.logger.Error("Failed to initiate search sync event consumption process", zap.Error(err))
	} else {
		w.logger.Info("Search sync coordinator successfully registered event handling callback")
	}
}

func (w *SearchSyncWorker) processSyncEvent(ctx context.Context, eventPayload dto.WardrobeEventPayload) error {
	switch eventPayload.Action {
	case eventconstants.ActionCreated, eventconstants.ActionUpdated:
		item, err := w.wardrobeRepo.GetByID(ctx, eventPayload.ItemID)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("item not found in database for indexing")
		}

		// Only sync system wardrobe items (SystemCatalogItem = 1) to the Search Index
		if item.ItemType != 1 {
			w.logger.Info("[SearchSyncWorker] Skipping user item indexing, only system catalog items are synced to search index",
				zap.String("itemId", item.ID.String()),
			)
			return nil
		}

		if err := w.searchIndex.IndexItem(ctx, item); err != nil {
			w.logger.Warn("[SearchSyncWorker] Failed to index item in search index (Elasticsearch may be offline)",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			// Return nil instead of throwing an error to avoid hanging or queue spam when ES is offline
			return nil
		}

	case eventconstants.ActionDeleted:
		if err := w.searchIndex.DeleteItem(ctx, eventPayload.ItemID.String()); err != nil {
			w.logger.Warn("[SearchSyncWorker] Failed to delete item from search index (Elasticsearch may be offline)",
				zap.String("itemId", eventPayload.ItemID.String()),
				zap.Error(err),
			)
			// Return nil instead of throwing an error to avoid hanging or queue spam when ES is offline
			return nil
		}
	}

	return nil
}

// initialSync pushes all System Catalog Items from PostgreSQL to the Search Index upon app startup
func (w *SearchSyncWorker) initialSync() {
	ctx := context.Background()

	items, err := w.wardrobeRepo.GetItems(ctx, nil, nil, itemtype.SystemCatalogItem)
	if err != nil {
		w.logger.Error("[SearchSyncWorker] Failed to fetch system catalog items for initial sync", zap.Error(err))
		return
	}

	if len(items) == 0 {
		w.logger.Info("[SearchSyncWorker] No system catalog items found in PostgreSQL, skipping initial sync")
		return
	}

	successCount := 0
	for _, item := range items {
		if err := w.searchIndex.IndexItem(ctx, item); err != nil {
			w.logger.Warn("[SearchSyncWorker] Failed to index system catalog item during initial sync. Elasticsearch might be offline. Aborting initial sync to prevent log spam.",
				zap.Error(err),
			)
			break
		}
		successCount++
	}

	w.logger.Info("[SearchSyncWorker] Initial sync process completed",
		zap.Int("total", len(items)),
		zap.Int("indexed", successCount),
	)
}

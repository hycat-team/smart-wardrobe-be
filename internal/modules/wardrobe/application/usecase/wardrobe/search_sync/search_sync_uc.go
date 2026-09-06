package search_sync

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SearchSyncUseCase struct {
	searchIndex     search.IWardrobeSearchIndexService
	wardrobeRepo    repositories.IWardrobeItemRepository
	fashionContract fashion_contract.IFashionContract
	logger          logger.Interface
}

func NewSearchSyncUseCase(
	searchIndex search.IWardrobeSearchIndexService,
	wardrobeRepo repositories.IWardrobeItemRepository,
	fashionContract fashion_contract.IFashionContract,
	l logger.Interface,
) usecase_interfaces.ISearchSyncUseCase {
	return &SearchSyncUseCase{
		searchIndex:     searchIndex,
		wardrobeRepo:    wardrobeRepo,
		fashionContract: fashionContract,
		logger:          l,
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
		if item.ItemType != itemtype.SystemCatalogItem {
			run.AddSkipped(1)
			return nil
		}

		var fashionItem *fashion_dto.FashionItemContractDTO
		if item.FashionItemID != nil {
			fashionItem, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
		}

		doc := mapper.MapToSearchDocumentDTO(item, fashionItem)
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

	items, err := uc.wardrobeRepo.GetItems(ctx, nil, itemtype.SystemCatalogItem)
	if err != nil {
		run.ChildError(uc.logger, "Failed to fetch system catalog items for initial sync", zap.Error(err))
		return false, err
	}
	run.AddTotal(len(items))

	if len(items) == 0 {
		run.AddSummaryFields(zap.Int("indexedCount", 0))
		return true, nil
	}

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionMap := map[uuid.UUID]*fashion_dto.FashionItemContractDTO{}
	if len(fashionIDs) > 0 {
		fashionItems, err := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
		if err == nil {
			fashionMap = sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemContractDTO) uuid.UUID { return f.ID })
		}
	}

	successCount := 0
	for _, item := range items {
		var fashionItem *fashion_dto.FashionItemContractDTO
		if item.FashionItemID != nil {
			fashionItem = fashionMap[*item.FashionItemID]
		}
		doc := mapper.MapToSearchDocumentDTO(item, fashionItem)
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

func (uc *SearchSyncUseCase) HandleFashionProcessingExhausted(ctx context.Context, payload fashion_dto.FashionProcessingExhaustedEventPayload, run *workerlog.Run) error {
	run.AddTotal(1)
	item, err := uc.wardrobeRepo.GetByID(ctx, payload.WardrobeItemID)
	if err != nil {
		return err
	}
	if item == nil {
		run.AddSkipped(1)
		return nil
	}
	if item.ItemType != itemtype.SystemCatalogItem {
		run.AddSkipped(1)
		return nil
	}
	var fashionItem *fashion_dto.FashionItemContractDTO
	if item.FashionItemID != nil {
		fashionItem, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	doc := mapper.MapToSearchDocumentDTO(item, fashionItem)
	if err := uc.searchIndex.IndexItem(ctx, doc); err != nil {
		run.ChildWarn(uc.logger, "Failed to re-index item for processing exhausted",
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
	return nil
}

func (uc *SearchSyncUseCase) IsOfflineError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "elasticsearch is offline")
}

func (uc *SearchSyncUseCase) RunInitialSyncWithRetry(ctx context.Context, log logger.Interface) {
	var initialSyncDone atomic.Int32

	tryInitialSync := func() {
		run := workerlog.New("search_sync", workerlog.TriggerInitialSync)
		done, err := uc.TryInitialSync(ctx, run)
		if err != nil {
			run.LogFailure(log, err)
		}
		if done {
			run.LogSuccessIfData(log)
			initialSyncDone.Store(1)
		}
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	tryInitialSync()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if initialSyncDone.Load() == 1 {
				return
			}
			tryInitialSync()
		}
	}
}

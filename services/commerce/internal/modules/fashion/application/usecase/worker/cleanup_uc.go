package worker

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/analysisreason"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (uc *FashionWorkerUseCase) CleanupFailedItems(ctx context.Context, run *workerlog.Run) error {
	limit := 100
	totalDeleted := 0
	publishedDeleteEventCount := 0

	for {
		items, err := uc.wardrobeContract.GetFailedItemsForCleanup(ctx, limit)
		if err != nil {
			return fmt.Errorf("failed to fetch items for cleanup: %w", err)
		}

		if len(items) == 0 {
			break
		}

		failedItemIDs := make([]uuid.UUID, 0, len(items))
		fashionIDs := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			failedItemIDs = append(failedItemIDs, item.ID)
			if item.FashionItemID != nil {
				fashionIDs = append(fashionIDs, *item.FashionItemID)
			}
		}
		fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
		fashionItemMap := map[uuid.UUID]*entities.FashionItem{}
		if len(fashionIDs) > 0 {
			if fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, fashionIDs); err == nil {
				fashionItemMap = sliceutils.IndexBy(fashionItems, func(fi *entities.FashionItem) uuid.UUID { return fi.ID })
			}
		}

		deletablePublicIDs := uc.deletableImageAssets(ctx, fashionIDs, failedItemIDs, fashionItemMap, run)
		deletedPublicIDs := map[string]bool{}

		for _, item := range items {
			run.AddTotal(1)
			imagePublicID := ""
			if item.FashionItemID != nil {
				if fi, ok := fashionItemMap[*item.FashionItemID]; ok && fi != nil {
					imagePublicID = fi.ImagePublicID
				}
			}
			if deletablePublicIDs[imagePublicID] && !deletedPublicIDs[imagePublicID] {
				if err := uc.mediaService.DeleteImage(ctx, imagePublicID); err != nil {
					run.ChildWarn(uc.logger, "Failed to delete image from Cloudinary during failed item cleanup",
						zap.String("itemId", item.ID.String()),
						zap.String("publicId", imagePublicID),
						zap.Error(err),
					)
				} else {
					deletedPublicIDs[imagePublicID] = true
				}
			}

			if err := uc.publishWardrobeDeleted(ctx, item.ID, item.UserID); err != nil {
				run.ChildError(uc.logger, "Failed to delete item from database during failed item cleanup",
					zap.String("itemId", item.ID.String()),
					zap.Error(err),
				)
				return fmt.Errorf("failed to delete item %s: %w", item.ID.String(), err)
			}
			publishedDeleteEventCount++
			totalDeleted++
			run.AddSuccess(1)
		}

		if len(items) < limit {
			break
		}
	}

	run.AddSummaryFields(
		zap.Int("deletedCount", totalDeleted),
		zap.Int("publishedDeleteEventCount", publishedDeleteEventCount),
	)
	return nil
}

// deletableImageAssets returns the set of Cloudinary public IDs whose fashion
// item has no remaining owner, so a shared image asset is destroyed only when
// the failed items being cleaned are its last owners. Owners are the wardrobe
// items of every item type, the outfit items and the active brand items, with
// the wardrobe items being cleaned in this page excluded. An unresolvable owner
// count makes every asset non-deletable: failing safe keeps a live image.
func (uc *FashionWorkerUseCase) deletableImageAssets(
	ctx context.Context,
	fashionIDs []uuid.UUID,
	failedItemIDs []uuid.UUID,
	fashionItemMap map[uuid.UUID]*entities.FashionItem,
	run *workerlog.Run,
) map[string]bool {
	deletable := map[string]bool{}
	if len(fashionIDs) == 0 {
		return deletable
	}

	otherOwners := map[uuid.UUID]int64{}
	ownersResolved := true
	if owners, err := uc.wardrobeContract.CountFashionItemOwnersByFashionIDs(ctx, fashionIDs, failedItemIDs); err == nil {
		for fashionItemID, count := range owners {
			otherOwners[fashionItemID] += count
		}
	} else {
		ownersResolved = false
		run.ChildWarn(uc.logger, "Failed to count wardrobe/outfit owners before image cleanup", zap.Error(err))
	}
	if uc.brandContract != nil {
		if brandItems, err := uc.brandContract.GetBrandItemsByFashionItemIDs(ctx, fashionIDs); err == nil {
			for _, brandItem := range brandItems {
				otherOwners[brandItem.FashionItemID]++
			}
		} else {
			ownersResolved = false
			run.ChildWarn(uc.logger, "Failed to count brand owners before image cleanup", zap.Error(err))
		}
	}

	for _, fashionItemID := range fashionIDs {
		fashionItem := fashionItemMap[fashionItemID]
		if fashionItem == nil || fashionItem.ImagePublicID == "" {
			continue
		}
		if !ownersResolved {
			run.ChildWarn(uc.logger, "Skipped image deletion: owner reference count unavailable",
				zap.String("fashionItemId", fashionItemID.String()),
				zap.String("publicId", fashionItem.ImagePublicID),
			)
			continue
		}
		if count := otherOwners[fashionItemID]; count > 0 {
			run.ChildWarn(uc.logger, "Skipped image deletion: fashion item still referenced by other owners",
				zap.String("fashionItemId", fashionItemID.String()),
				zap.String("publicId", fashionItem.ImagePublicID),
				zap.Int64("otherOwnerCount", count),
			)
			continue
		}
		deletable[fashionItem.ImagePublicID] = true
	}
	return deletable
}

// publishWardrobeDeleted emits the delete event that removes the wardrobe row of
// a failed item, inside the unit of work so it joins the outbox transaction.
func (uc *FashionWorkerUseCase) publishWardrobeDeleted(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		payload := wardrobeDto.WardrobeEventPayload{
			ItemID: itemID,
			UserID: userID,
			Action: eventconstants.ActionDeleted,
		}
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeDeleted, payload)
	})
}

// staleOwner identifies the item that owns a fashion item's analysis.
type staleOwner struct {
	itemID   uuid.UUID
	itemType string
	userID   uuid.UUID
}

// resolveOwners maps a fashion item to the wardrobe and/or brand items that
// reference it, so recovery can notify the right owner (FR-035).
func (uc *FashionWorkerUseCase) resolveOwners(ctx context.Context, fashionItemID uuid.UUID) []staleOwner {
	var owners []staleOwner
	if items, err := uc.wardrobeContract.GetWardrobeItemsByFashionItemIDs(ctx, []uuid.UUID{fashionItemID}); err == nil {
		for _, it := range items {
			owners = append(owners, staleOwner{itemID: it.ID, itemType: "wardrobe", userID: it.UserID})
		}
	}
	if uc.brandContract != nil {
		if items, err := uc.brandContract.GetBrandItemsByFashionItemIDs(ctx, []uuid.UUID{fashionItemID}); err == nil {
			for _, it := range items {
				owners = append(owners, staleOwner{itemID: it.ID, itemType: "brand"})
			}
		}
	}
	return owners
}

// RecoverStaleProcessingItems scans stale processing fashion items (fashion owns
// the processing state) and either republishes the analysis or marks them failed
// once retries are exhausted, for both wardrobe and brand owners (FR-035).
func (uc *FashionWorkerUseCase) RecoverStaleProcessingItems(ctx context.Context, run *workerlog.Run) error {
	staleBefore := time.Now().UTC().Add(-time.Duration(uc.cfg.Wardrobe.StaleMinutes) * time.Minute)
	items, err := uc.fashionItemRepo.GetStaleProcessingItems(ctx, staleBefore, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch stale processing items: %w", err)
	}
	run.AddTotal(len(items))
	republishedCount := 0
	markedFailedCount := 0

	for _, fi := range items {
		if fi == nil {
			continue
		}
		owners := uc.resolveOwners(ctx, fi.ID)
		if len(owners) == 0 {
			run.AddSkipped(1)
			continue
		}

		if fi.ProcessingRetryCount >= uc.cfg.Wardrobe.MaxRetryCount {
			for _, o := range owners {
				payload := dto.FashionProcessingExhaustedEventPayload{
					ItemID:        o.itemID,
					ItemType:      o.itemType,
					FashionItemID: fi.ID,
					Reason:        analysisreason.ErrorAutoRetryExceeded,
					UserID:        o.userID,
				}
				err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
					if _, err := uc.fashionItemRepo.MarkProcessingFailed(txCtx, fi.ID, fi.ProcessingVersion, analysisreason.ErrorAutoRetryExceeded, nil); err != nil {
						return err
					}
					return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionProcessingExhausted, payload)
				})
				if err == nil {
					markedFailedCount++
				}
			}
			run.ChildInfo(uc.logger, "Stale processing item marked failed after retries exhausted",
				zap.String("fashionItemId", fi.ID.String()),
				zap.Int("processingVersion", fi.ProcessingVersion),
			)
			continue
		}

		if err := uc.fashionItemRepo.IncrementProcessingRetryCount(ctx, fi.ID); err != nil {
			run.ChildError(uc.logger, "Failed to increment retry count for stale processing item",
				zap.String("fashionItemId", fi.ID.String()),
				zap.Error(err),
			)
			continue
		}
		for _, o := range owners {
			if err := uc.publishProcessingRetry(ctx, o.itemID, &fi.ID, o.userID, o.itemType); err != nil {
				run.ChildError(uc.logger, "Failed to republish stale processing item",
					zap.String("itemId", o.itemID.String()),
					zap.Error(err),
				)
				continue
			}
			republishedCount++
			run.AddSuccess(1)
		}
	}

	run.AddSummaryFields(
		zap.Int("staleItemsFoundCount", len(items)),
		zap.Int("republishedCount", republishedCount),
		zap.Int("markedFailedCount", markedFailedCount),
	)
	return nil
}

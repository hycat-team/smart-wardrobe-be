package worker

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"smart-wardrobe-be/internal/pkg/utils/sliceutils"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

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

		fashionIDs := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
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
		for _, item := range items {
			run.AddTotal(1)
			imagePublicID := ""
			if item.FashionItemID != nil {
				if fi, ok := fashionItemMap[*item.FashionItemID]; ok && fi != nil {
					imagePublicID = fi.ImagePublicID
				}
			}
			if imagePublicID != "" {
				if err := uc.mediaService.DeleteImage(ctx, imagePublicID); err != nil {
					run.ChildWarn(uc.logger, "Failed to delete image from Cloudinary during failed item cleanup",
						zap.String("itemId", item.ID.String()),
						zap.String("publicId", imagePublicID),
						zap.Error(err),
					)
				}
			}

			err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
				payload := wardrobeDto.WardrobeEventPayload{
					ItemID: item.ID,
					UserID: item.UserID,
					Action: eventconstants.ActionDeleted,
				}
				return uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeDeleted, payload)
			})
			if err != nil {
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

// RecoverStaleProcessingItems republishes stale processing items or marks them failed when retries are exhausted.
func (uc *FashionWorkerUseCase) RecoverStaleProcessingItems(ctx context.Context, run *workerlog.Run) error {
	staleBefore := time.Now().UTC().Add(-time.Duration(uc.cfg.Wardrobe.StaleMinutes) * time.Minute)
	items, err := uc.wardrobeContract.GetStaleProcessingItems(ctx, staleBefore, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch stale processing items: %w", err)
	}
	run.AddTotal(len(items))
	republishedCount := 0
	markedFailedCount := 0

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
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
	for _, item := range items {
		if item.FashionItemID == nil {
			run.AddSkipped(1)
			continue
		}
		fi, ok := fashionItemMap[*item.FashionItemID]
		if !ok || fi == nil {
			run.AddSkipped(1)
			continue
		}
		if fi.ProcessingRetryCount >= uc.cfg.Wardrobe.MaxRetryCount {
			payload := dto.FashionProcessingExhaustedPayload{
				WardrobeItemID: item.ID,
				FashionItemID:  fi.ID,
				Reason:         autoRetryExceededMessage,
				UserID:         item.UserID,
			}
			err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
				if _, err := uc.fashionItemRepo.MarkProcessingFailed(txCtx, fi.ID, fi.ProcessingVersion, autoRetryExceededMessage, nil); err != nil {
					return err
				}
				return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionProcessingExhausted, payload)
			})
			if err == nil {
				markedFailedCount++
			}
			run.ChildInfo(uc.logger, "Stale processing item marked failed after retries exhausted",
				zap.String("itemId", item.ID.String()),
				zap.Int("processingVersion", fi.ProcessingVersion),
			)
			continue
		}

		if err := uc.fashionItemRepo.IncrementProcessingRetryCount(ctx, fi.ID); err != nil {
			run.ChildError(uc.logger, "Failed to increment retry count for stale processing item",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			continue
		}
		if err := uc.publishProcessingRetry(ctx, item.ID, &fi.ID, item.UserID); err != nil {
			run.ChildError(uc.logger, "Failed to republish stale processing item",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			continue
		}
		republishedCount++
		run.AddSuccess(1)
	}

	run.AddSummaryFields(
		zap.Int("staleItemsFoundCount", len(items)),
		zap.Int("republishedCount", republishedCount),
		zap.Int("markedFailedCount", markedFailedCount),
	)
	return nil
}

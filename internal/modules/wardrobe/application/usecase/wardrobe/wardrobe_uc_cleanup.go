package wardrobe

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"

	"go.uber.org/zap"
)

func (uc *WardrobeWorkerUseCase) CleanupFailedItems(ctx context.Context) error {
	limit := 100
	totalDeleted := 0

	for {
		items, err := uc.wardrobeRepo.GetFailedItemsForCleanup(ctx, limit)
		if err != nil {
			return fmt.Errorf("failed to fetch items for cleanup: %w", err)
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if item.ImagePublicID != "" {
				if err := uc.mediaService.DeleteImage(ctx, item.ImagePublicID); err != nil {
					uc.logger.Warn("[CleanupFailedItems] Failed to delete image from Cloudinary",
						zap.String("item_id", item.ID.String()),
						zap.String("public_id", item.ImagePublicID),
						zap.Error(err),
					)
				}
			}

			if err := uc.wardrobeRepo.Delete(ctx, item.ID); err != nil {
				uc.logger.Error("[CleanupFailedItems] Failed to delete item from database",
					zap.String("item_id", item.ID.String()),
					zap.Error(err),
				)
				return fmt.Errorf("failed to delete item %s: %w", item.ID.String(), err)
			}

			payload := dto.WardrobeEventPayload{
				ItemID: item.ID,
				UserID: item.UserID,
				Action: eventconstants.ActionDeleted,
			}
			_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeDeleted, payload)

			totalDeleted++
		}

		if len(items) < limit {
			break
		}
	}

	uc.logger.Info("[CleanupFailedItems] Successfully cleaned up failed items", zap.Int("count", totalDeleted))
	return nil
}

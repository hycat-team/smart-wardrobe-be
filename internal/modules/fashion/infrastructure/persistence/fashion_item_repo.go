package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FashionItemRepository struct {
	shared_persist.GenericRepository[entities.FashionItem, uuid.UUID]
}

func NewFashionItemRepository(db *gorm.DB) repositories.IFashionItemRepository {
	relations := []string{"Category"}
	return &FashionItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.FashionItem, uuid.UUID](db, relations),
	}
}

func (r *FashionItemRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	var items []*entities.FashionItem
	err := r.GetDB(ctx).Where("id IN ?", ids).Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FashionItemRepository) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.FashionItem, error) {
	var items []*entities.FashionItem
	err := r.GetDB(ctx).
		Where("review_reason IS NOT NULL AND created_at < NOW() - INTERVAL '7 days'").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FashionItemRepository) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.FashionItem, error) {
	var items []*entities.FashionItem
	err := r.GetDB(ctx).
		Where("processing_started_at < ? AND last_processing_attempt_at IS NULL", staleBefore).
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FashionItemRepository) ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.FashionItem, bool, error) {
	var claimed entities.FashionItem
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entities.FashionItem{}).
			Where("id = ?", itemID).
			Updates(map[string]any{
				"processing_started_at": now,
				"updated_at":            now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("id = ?", itemID).First(&claimed).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &claimed, true, nil
}

func (r *FashionItemRepository) ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.FashionItem, bool, error) {
	var claimed entities.FashionItem
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entities.FashionItem{}).
			Where("id = ? AND processing_version = ?", itemID, processingVersion).
			Updates(map[string]any{
				"processing_started_at":      now,
				"last_processing_attempt_at": now,
				"processing_version":         processingVersion + 1,
				"updated_at":                 now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("id = ?", itemID).First(&claimed).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &claimed, true, nil
}

func (r *FashionItemRepository) MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
	result := r.GetDB(ctx).Model(&entities.FashionItem{}).
		Where("id = ? AND processing_version = ?", itemID, processingVersion).
		Updates(map[string]any{
			"processing_error_reason: ": reason,
			"review_reason":             reviewReason,
			"updated_at":                time.Now().UTC(),
		})
	return result.RowsAffected > 0, result.Error
}

func (r *FashionItemRepository) MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error) {
	result := r.GetDB(ctx).Model(&entities.FashionItem{}).
		Where("id = ? AND processing_version = ?", itemID, processingVersion).
		Updates(map[string]any{
			"review_reason": reviewReason,
			"updated_at":    time.Now().UTC(),
		})
	return result.RowsAffected > 0, result.Error
}

func (r *FashionItemRepository) CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	result := r.GetDB(ctx).Model(&entities.FashionItem{}).
		Where("id = ? AND processing_version = ?", itemID, processingVersion).
		Updates(updates)
	return result.RowsAffected > 0, result.Error
}

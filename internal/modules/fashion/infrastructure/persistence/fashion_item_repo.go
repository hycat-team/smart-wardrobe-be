package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
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
	err := r.GetQueryWithPreload(ctx).Where("id IN ?", ids).Find(&items).Error
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

// SearchFashionItemIDs returns fashion_item ids matching the keyword query and/or category slug.
func (r *FashionItemRepository) SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error) {
	db := r.GetDB(ctx).Model(&entities.FashionItem{})
	if categorySlug != nil && *categorySlug != "" {
		db = db.Joins("JOIN categories ON categories.id = fashion_items.category_id").
			Where("categories.slug = ?", *categorySlug)
	}
	if query != nil && *query != "" {
		queryStr := strings.ToLower(*query)
		db = db.Where("(LOWER(fashion_items.color) LIKE ? OR LOWER(fashion_items.style) LIKE ? OR LOWER(fashion_items.material) LIKE ? OR LOWER(fashion_items.pattern) LIKE ? OR LOWER(fashion_items.fit) LIKE ? OR LOWER(fashion_items.seasonality) LIKE ? OR LOWER(fashion_items.description) LIKE ? OR LOWER(categories.name) LIKE ?)",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}
	var ids []uuid.UUID
	if err := db.Pluck("fashion_items.id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// GetFashionItemIDsByCategory returns fashion_item ids belonging to the given category.
func (r *FashionItemRepository) GetFashionItemIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.GetDB(ctx).
		Model(&entities.FashionItem{}).
		Where("category_id = ?", categoryID).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *FashionItemRepository) GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error) {
	if len(fashionItemIDs) == 0 {
		return nil, nil
	}
	var rows []dto.DashboardFashionItemCategoryDTO
	err := r.GetDB(ctx).
		Model(&entities.FashionItem{}).
		Select("id AS fashion_item_id, category_id AS category_id").
		Where("id IN ?", fashionItemIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
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

func (r *FashionItemRepository) IncrementProcessingRetryCount(ctx context.Context, itemID uuid.UUID) error {
	return r.GetDB(ctx).
		Model(&entities.FashionItem{}).
		Where("id = ?", itemID).
		Updates(map[string]any{
			"processing_retry_count":     gorm.Expr("processing_retry_count + 1"),
			"last_processing_attempt_at": time.Now().UTC(),
			"updated_at":                 time.Now().UTC(),
		}).Error
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

func (r *FashionItemRepository) ReassignCategory(ctx context.Context, fashionItemIDs []uuid.UUID, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error {
	if len(fashionItemIDs) == 0 {
		return nil
	}
	return r.GetDB(ctx).
		Model(&entities.FashionItem{}).
		Where("id IN ? AND category_id = ?", fashionItemIDs, fromCategoryID).
		Update("category_id", toCategoryID).Error
}

func (r *FashionItemRepository) ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error {
	return r.GetDB(ctx).
		Model(&entities.FashionItem{}).
		Where("id = ?", fashionItemID).
		Updates(map[string]any{
			"review_reason":              nil,
			"processing_error_reason":    nil,
			"processing_retry_count":     0,
			"processing_started_at":      nil,
			"last_processing_attempt_at": nil,
			"processing_version":         gorm.Expr("processing_version + 1"),
			"updated_at":                 time.Now().UTC(),
		}).Error
}

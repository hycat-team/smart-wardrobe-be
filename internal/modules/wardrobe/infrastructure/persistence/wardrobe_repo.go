package persistence

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WardrobeItemRepository struct {
	shared_persist.GenericRepository[entities.WardrobeItem, uuid.UUID]
}

func NewWardrobeItemRepository(db *gorm.DB) repositories.IWardrobeItemRepository {
	relations := []string{}
	return &WardrobeItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.WardrobeItem, uuid.UUID](db, relations),
	}
}

func (r *WardrobeItemRepository) Create(ctx context.Context, item *entities.WardrobeItem) error {
	if item == nil {
		return nil
	}
	if err := r.ensureFashionItemBeforeCreate(ctx, item); err != nil {
		return err
	}
	return r.GetDB(ctx).Create(item).Error
}

func (r *WardrobeItemRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.WardrobeItem{}).
		Where("user_id = ? AND is_deleted = ? AND status = ?", userID, false, wardrobestatus.InWardrobe).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) CountByUserIDAndFilters(ctx context.Context, userID uuid.UUID, categorySlug *string, statuses []wardrobestatus.WardrobeItemStatus) (int64, error) {
	var count int64
	query := r.buildUserWardrobeFilterQuery(ctx, userID, categorySlug, statuses, false)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) GetByUserID(ctx context.Context, userID uuid.UUID, categorySlug *string) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetDB(ctx).
		Where("user_id = ? AND is_deleted = ? AND status = ?", userID, false, wardrobestatus.InWardrobe)
	if categorySlug != nil && *categorySlug != "" {
		query = query.
			Where("fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id IN (SELECT id FROM categories WHERE slug = ?))", *categorySlug)
	}
	err := query.Order("wardrobe_items.created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetByUserIDAndFiltersPaginated(ctx context.Context, userID uuid.UUID, categorySlug *string, statuses []wardrobestatus.WardrobeItemStatus, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.buildUserWardrobeFilterQuery(ctx, userID, categorySlug, statuses, true)
	db := shared_persist.ApplyPagination(query, pagination)
	err := db.Order("wardrobe_items.created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) BulkCreate(ctx context.Context, items []*entities.WardrobeItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := r.ensureFashionItemBeforeCreateWithDB(tx, item); err != nil {
				return err
			}
		}
		return tx.Create(&items).Error
	})
}

func (r *WardrobeItemRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []*entities.WardrobeItem
	err := r.GetQueryWithPreload(ctx).Where("id IN ?", ids).Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetByUserIDAndFashionItemIDs(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID) ([]*entities.WardrobeItem, error) {
	if len(fashionItemIDs) == 0 {
		return nil, nil
	}

	var items []*entities.WardrobeItem
	err := r.GetQueryWithPreload(ctx).
		Where("user_id = ? AND fashion_item_id IN ? AND is_deleted = ?", userID, fashionItemIDs, false).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error) {
	return r.getByIDWithDB(r.GetDB(ctx), id)
}

func (r *WardrobeItemRepository) getByIDWithDB(db *gorm.DB, id uuid.UUID) (*entities.WardrobeItem, error) {
	var item entities.WardrobeItem
	err := db.Where("wardrobe_items.id = ?", id).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *WardrobeItemRepository) Update(ctx context.Context, item *entities.WardrobeItem) error {
	if item == nil {
		return nil
	}
	return r.GetDB(ctx).Save(item).Error
}

func (r *WardrobeItemRepository) CountItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) (int64, error) {
	var count int64
	db := r.GetDB(ctx).Model(&entities.WardrobeItem{})

	db = db.Where("item_type = ? AND wardrobe_items.is_deleted = ?", itemType, false)
	if categorySlug != nil && *categorySlug != "" {
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id IN (SELECT id FROM categories WHERE slug = ?))", *categorySlug)
	}

	if query != nil && *query != "" {
		queryStr := strings.ToLower(*query)
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE LOWER(color) LIKE ? OR LOWER(style) LIKE ? OR LOWER(material) LIKE ? OR LOWER(pattern) LIKE ? OR LOWER(fit) LIKE ? OR LOWER(seasonality) LIKE ? OR LOWER(description) LIKE ? OR category_id IN (SELECT id FROM categories WHERE LOWER(name) LIKE ?))",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) GetItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	db := r.GetDB(ctx)

	db = db.Where("item_type = ? AND wardrobe_items.is_deleted = ?", itemType, false)
	if categorySlug != nil && *categorySlug != "" {
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id IN (SELECT id FROM categories WHERE slug = ?))", *categorySlug)
	}

	if query != nil && *query != "" {
		queryStr := strings.ToLower(*query)
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE LOWER(color) LIKE ? OR LOWER(style) LIKE ? OR LOWER(material) LIKE ? OR LOWER(pattern) LIKE ? OR LOWER(fit) LIKE ? OR LOWER(seasonality) LIKE ? OR LOWER(description) LIKE ? OR category_id IN (SELECT id FROM categories WHERE LOWER(name) LIKE ?))",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	err := db.Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetItemsPaginated(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	db := r.GetDB(ctx)

	db = db.Where("item_type = ? AND wardrobe_items.is_deleted = ?", itemType, false)
	if categorySlug != nil && *categorySlug != "" {
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id IN (SELECT id FROM categories WHERE slug = ?))", *categorySlug)
	}

	if query != nil && *query != "" {
		queryStr := strings.ToLower(*query)
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE LOWER(color) LIKE ? OR LOWER(style) LIKE ? OR LOWER(material) LIKE ? OR LOWER(pattern) LIKE ? OR LOWER(fit) LIKE ? OR LOWER(seasonality) LIKE ? OR LOWER(description) LIKE ? OR category_id IN (SELECT id FROM categories WHERE LOWER(name) LIKE ?))",
			"%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%", "%"+queryStr+"%")
	}

	db = shared_persist.ApplyPagination(db, pagination)
	err := db.Order("wardrobe_items.created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	err := r.GetDB(ctx).
		Where("wardrobe_items.status = ? AND wardrobe_items.created_at < NOW() - INTERVAL '7 days'", wardrobestatus.Failed).
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	err := r.GetDB(ctx).
		Where("wardrobe_items.status = ? AND wardrobe_items.is_deleted = ? AND (SELECT COALESCE(last_processing_attempt_at, processing_started_at, ?::timestamp) FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) < ?", wardrobestatus.Processing, false, staleBefore, staleBefore).
		Order("wardrobe_items.created_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.WardrobeItem, bool, error) {
	var claimed *entities.WardrobeItem
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entities.WardrobeItem{}).
			Where("id = ? AND user_id = ? AND is_deleted = ? AND status IN ?", itemID, userID, false, []wardrobestatus.WardrobeItemStatus{
				wardrobestatus.Failed,
				wardrobestatus.NeedsReview,
			}).
			Updates(map[string]any{
				"status":     wardrobestatus.Processing,
				"updated_at": now,
			})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}

		var item entities.WardrobeItem
		if err := tx.Where("id = ?", itemID).First(&item).Error; err != nil {
			return err
		}
		if item.FashionItemID == nil {
			item.FashionItemID = &item.ID
		}
		if err := tx.Exec(`UPDATE fashion_items SET review_reason = NULL, processing_error_reason = NULL, processing_retry_count = 0, processing_started_at = ?, last_processing_attempt_at = ?, processing_version = processing_version + 1, updated_at = ? WHERE id = ?`,
			now, now, now, *item.FashionItemID).Error; err != nil {
			return err
		}
		var err error
		claimed, err = r.getByIDWithDB(tx, itemID)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return claimed, claimed != nil, nil
}

func (r *WardrobeItemRepository) ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.WardrobeItem, bool, error) {
	var claimed *entities.WardrobeItem
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var item entities.WardrobeItem
		err := tx.
			Where("wardrobe_items.id = ? AND wardrobe_items.is_deleted = ? AND wardrobe_items.status = ? AND (SELECT processing_version FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) = ? AND (SELECT COALESCE(last_processing_attempt_at, processing_started_at, ?::timestamp) FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) < ?",
				itemID, false, wardrobestatus.Processing, processingVersion, staleBefore, staleBefore).
			First(&item).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if item.FashionItemID == nil {
			item.FashionItemID = &item.ID
		}
		result := tx.Exec(`UPDATE fashion_items SET processing_retry_count = processing_retry_count + 1, last_processing_attempt_at = ?, processing_version = processing_version + 1, updated_at = ? WHERE id = ? AND processing_version = ?`,
			now, now, *item.FashionItemID, processingVersion)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		claimed, err = r.getByIDWithDB(tx, itemID)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return claimed, claimed != nil, nil
}

func (r *WardrobeItemRepository) MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
	updates := map[string]any{
		"status":                     wardrobestatus.Failed,
		"processing_error_reason":    reason,
		"review_reason":              reviewReason,
		"processing_started_at":      nil,
		"last_processing_attempt_at": nil,
		"updated_at":                 time.Now().UTC(),
	}

	return r.updateProcessingState(ctx, itemID, processingVersion, updates)
}

func (r *WardrobeItemRepository) MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error) {
	updates := map[string]any{
		"status":                     wardrobestatus.NeedsReview,
		"review_reason":              reviewReason,
		"processing_error_reason":    nil,
		"processing_started_at":      nil,
		"last_processing_attempt_at": nil,
		"processing_retry_count":     0,
		"updated_at":                 time.Now().UTC(),
	}

	return r.updateProcessingState(ctx, itemID, processingVersion, updates)
}

func (r *WardrobeItemRepository) CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	if updates == nil {
		updates = make(map[string]any)
	}
	updates["status"] = wardrobestatus.InWardrobe
	updates["review_reason"] = nil
	updates["processing_error_reason"] = nil
	updates["processing_started_at"] = nil
	updates["last_processing_attempt_at"] = nil
	updates["processing_retry_count"] = 0
	updates["updated_at"] = time.Now().UTC()

	return r.updateProcessingState(ctx, itemID, processingVersion, updates)
}

func (r *WardrobeItemRepository) updateProcessingState(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	if updates == nil {
		updates = make(map[string]any)
	}

	returnValue := false
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var item entities.WardrobeItem
		err := tx.
			Where("wardrobe_items.id = ? AND wardrobe_items.is_deleted = ? AND wardrobe_items.status = ? AND (SELECT processing_version FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) = ?",
				itemID, false, wardrobestatus.Processing, processingVersion).
			First(&item).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if item.FashionItemID == nil {
			item.FashionItemID = &item.ID
		}

		wardrobeUpdates := map[string]any{}
		if status, ok := updates["status"]; ok {
			wardrobeUpdates["status"] = status
		}
		if updatedAt, ok := updates["updated_at"]; ok {
			wardrobeUpdates["updated_at"] = updatedAt
		}
		if len(wardrobeUpdates) > 0 {
			result := tx.Model(&entities.WardrobeItem{}).
				Where("id = ? AND is_deleted = ? AND status = ?", itemID, false, wardrobestatus.Processing).
				Updates(wardrobeUpdates)
			if result.Error != nil || result.RowsAffected == 0 {
				return result.Error
			}
		}

		fashionUpdates := make(map[string]any, len(updates))
		for key, value := range updates {
			if key == "status" {
				continue
			}
			fashionUpdates[key] = value
		}
		if len(fashionUpdates) > 0 {
			setClauses := make([]string, 0, len(fashionUpdates))
			args := make([]any, 0, len(fashionUpdates)+2)
			for k, v := range fashionUpdates {
				setClauses = append(setClauses, k+" = ?")
				args = append(args, v)
			}
			args = append(args, *item.FashionItemID, processingVersion)
			result := tx.Exec("UPDATE fashion_items SET "+strings.Join(setClauses, ", ")+" WHERE id = ? AND processing_version = ?", args...)
			if result.Error != nil {
				return result.Error
			}
			returnValue = result.RowsAffected > 0
		}
		return nil
	})
	return returnValue, err
}

func (r *WardrobeItemRepository) TouchLastUsedAt(ctx context.Context, ids []uuid.UUID, usedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}

	return r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Where("id IN ?", ids).
		Update("last_used_at", usedAt).Error
}

func (r *WardrobeItemRepository) buildUserWardrobeFilterQuery(ctx context.Context, userID uuid.UUID, categorySlug *string, statuses []wardrobestatus.WardrobeItemStatus, withPreload bool) *gorm.DB {
	var query *gorm.DB
	if withPreload {
		query = r.GetQueryWithPreload(ctx)
	} else {
		query = r.GetDB(ctx).Model(&entities.WardrobeItem{})
	}

	query = query.Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = ?", userID, false)
	if categorySlug != nil && *categorySlug != "" {
		query = query.
			Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id IN (SELECT id FROM categories WHERE slug = ?))", *categorySlug)
	}
	if len(statuses) > 0 {
		query = query.Where("wardrobe_items.status IN ?", statuses)
	}

	return query
}

func (r *WardrobeItemRepository) ensureFashionItemBeforeCreate(ctx context.Context, item *entities.WardrobeItem) error {
	return r.ensureFashionItemBeforeCreateWithDB(r.GetDB(ctx), item)
}

func (r *WardrobeItemRepository) ensureFashionItemBeforeCreateWithDB(db *gorm.DB, item *entities.WardrobeItem) error {
	if item == nil {
		return nil
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.FashionItemID == nil {
		item.FashionItemID = &item.ID
	}
	return nil
}

func (r *WardrobeItemRepository) CountByCategoryIDAndItemType(ctx context.Context, categoryID uuid.UUID, itemType itemtype.ItemType) (int64, error) {
	var count int64
	err := r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Where("fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id = ?) AND item_type = ? AND is_deleted = ?", categoryID, itemType, false).
		Count(&count).Error
	return count, err
}

func (r *WardrobeItemRepository) ReassignCategoryForCatalogItems(ctx context.Context, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error {
	return r.GetDB(ctx).
		Exec("UPDATE fashion_items SET category_id = ? WHERE id IN (SELECT fashion_item_id FROM wardrobe_items WHERE fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id = ?) AND item_type = ? AND is_deleted = ?)",
			toCategoryID, fromCategoryID, itemtype.SystemCatalogItem, false).Error
}

func (r *WardrobeItemRepository) GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error) {
	db := r.GetDB(ctx)

	var totalItems int64
	if err := db.Model(&entities.WardrobeItem{}).Where("user_id = ? AND is_deleted = ?", userID, false).Count(&totalItems).Error; err != nil {
		return nil, err
	}

	var totalValue float64
	_ = db.Model(&entities.WardrobeItem{}).
		Where("user_id = ? AND is_deleted = ? AND purchase_price IS NOT NULL", userID, false).
		Select("COALESCE(SUM(purchase_price), 0)").Scan(&totalValue).Error

	staleTime := time.Now().AddDate(0, 0, -30)
	var underutilized int64
	_ = db.Model(&entities.WardrobeItem{}).
		Where("user_id = ? AND is_deleted = ? AND (last_used_at IS NULL OR last_used_at < ?)", userID, false, staleTime).
		Count(&underutilized).Error

	var cpw float64
	if totalItems > 0 {
		cpw = totalValue / float64(totalItems)
	}

	return &dto.UserWardrobeSummaryStatsDTO{
		UserID:                  userID,
		TotalItems:              int(totalItems),
		TotalWardrobeValueVND:   totalValue,
		AverageCostPerWearVND:   cpw,
		UnderutilizedItemsCount: int(underutilized),
	}, nil
}

func (r *WardrobeItemRepository) GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.UnderutilizedItemInfoDTO, error) {
	staleTime := time.Now().AddDate(0, 0, -30)
	var items []*entities.WardrobeItem
	err := r.GetDB(ctx).
		Where("user_id = ? AND is_deleted = ? AND (last_used_at IS NULL OR last_used_at < ?)", userID, false, staleTime).
		Order("last_used_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	result := make([]*dto.UnderutilizedItemInfoDTO, 0, len(items))
	now := time.Now()
	for _, item := range items {
		wearCount := 0
		price := 0.0
		if item.Price != nil {
			price = *item.Price
		}
		cpw := price
		if wearCount > 0 {
			cpw = price / float64(wearCount)
		}

		daysAgo := 30
		if item.LastUsedAt != nil {
			daysAgo = int(now.Sub(*item.LastUsedAt).Hours() / 24)
		}

		result = append(result, &dto.UnderutilizedItemInfoDTO{
			ItemID:           item.ID,
			Name:             "Trang phục",
			CategoryName:     "Chưa phân loại",
			PurchasePriceVND: price,
			WearCount:        wearCount,
			CostPerWearVND:   cpw,
			LastWornDaysAgo:  daysAgo,
			ImageURL:         "",
		})
	}
	return result, nil
}

package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

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

// activeWardrobeFilter áp dụng điều kiện "active" (InWardrobe, chưa xóa mềm) cho query
// — tập trung 1 chỗ để tránh lặp điều kiện ở nhiều method.
func activeWardrobeFilter(db *gorm.DB) *gorm.DB {
	return db.Where("is_deleted = ? AND status = ?", false, wardrobestatus.InWardrobe)
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
		Scopes(activeWardrobeFilter).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) CountByUserIDAndFilters(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filterByFashionItems bool) (int64, error) {
	var count int64
	query := r.buildUserWardrobeFilterQuery(ctx, userID, fashionItemIDs, statuses, filterByFashionItems, false)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) GetByUserID(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetDB(ctx).Scopes(activeWardrobeFilter).
		Where("user_id = ?", userID)
	if len(fashionItemIDs) > 0 {
		query = query.Where("fashion_item_id IN ?", fashionItemIDs)
	}
	err := query.Order("wardrobe_items.created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetActiveByUserID returns active (InWardrobe, not deleted) wardrobe items for a user,
// limited at the database level to avoid loading the whole wardrobe into memory.
func (r *WardrobeItemRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetDB(ctx).Scopes(activeWardrobeFilter).
		Where("user_id = ?", userID).
		Order("wardrobe_items.created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetByUserIDAndFiltersPaginated(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filterByFashionItems bool, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.buildUserWardrobeFilterQuery(ctx, userID, fashionItemIDs, statuses, filterByFashionItems, true)
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

// GetByFashionItemIDs returns all (non-deleted) wardrobe items referencing the
// given fashion items, across users. Used by stale-processing recovery (FR-035).
func (r *WardrobeItemRepository) GetByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*entities.WardrobeItem, error) {
	if len(fashionItemIDs) == 0 {
		return nil, nil
	}
	var items []*entities.WardrobeItem
	err := r.GetDB(ctx).
		Where("fashion_item_id IN ? AND is_deleted = ?", fashionItemIDs, false).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetUserFashionItemCounts returns, per fashion_item_id, the number of non-deleted
// wardrobe items owned by the user. Every status is counted and duplicate
// references are preserved, so the result matches the wardrobe list default
// (GET /me/wardrobe-items) rather than a de-duplicated fashion item set.
func (r *WardrobeItemRepository) GetUserFashionItemCounts(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error) {
	rows := []struct {
		FashionItemID uuid.UUID
		Count         int64
	}{}
	err := r.GetDB(ctx).Model(&entities.WardrobeItem{}).
		Select("fashion_item_id, COUNT(*) AS count").
		Where("user_id = ? AND is_deleted = ? AND fashion_item_id IS NOT NULL", userID, false).
		Group("fashion_item_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		counts[row.FashionItemID] = row.Count
	}
	return counts, nil
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

func (r *WardrobeItemRepository) CountItems(ctx context.Context, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType) (int64, error) {
	var count int64
	db := r.GetDB(ctx).Model(&entities.WardrobeItem{})
	db = buildWardrobeItemSearchQuery(db, fashionItemIDs, filterByFashionItems, itemType)
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountItemsGroupedByFashionIDs counts non-deleted wardrobe items of every item
// type grouped by fashion_item_id, optionally excluding the given wardrobe item
// IDs. Used to resolve how many owners still reference a fashion item before its
// shared image is removed, and to guard category deletion.
func (r *WardrobeItemRepository) CountItemsGroupedByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID, excludeItemIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	result := make(map[uuid.UUID]int64)
	if len(fashionItemIDs) == 0 {
		return result, nil
	}
	query := r.GetDB(ctx).Model(&entities.WardrobeItem{}).
		Select("fashion_item_id, COUNT(*) AS count").
		Where("fashion_item_id IN ? AND is_deleted = ?", fashionItemIDs, false)
	if len(excludeItemIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeItemIDs)
	}
	rows := []struct {
		FashionItemID uuid.UUID
		Count         int64
	}{}
	if err := query.Group("fashion_item_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.FashionItemID] = row.Count
	}
	return result, nil
}

func (r *WardrobeItemRepository) GetItems(ctx context.Context, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	db := buildWardrobeItemSearchQuery(r.GetDB(ctx), fashionItemIDs, filterByFashionItems, itemType)
	err := db.Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetItemsPaginated(ctx context.Context, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	db := buildWardrobeItemSearchQuery(r.GetDB(ctx), fashionItemIDs, filterByFashionItems, itemType)
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
		Where("wardrobe_items.status = ? AND wardrobe_items.is_deleted = ? AND wardrobe_items.updated_at < ?", wardrobestatus.Processing, false, staleBefore).
		Order("wardrobe_items.updated_at ASC").
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
		result := tx.Model(&entities.WardrobeItem{}).
			Where("id = ? AND is_deleted = ? AND status = ?", itemID, false, wardrobestatus.Processing).
			Updates(map[string]any{
				"updated_at": now,
			})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
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

func (r *WardrobeItemRepository) MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
	return r.updateProcessingState(ctx, itemID, processingVersion, map[string]any{
		"status":     wardrobestatus.Failed,
		"updated_at": time.Now().UTC(),
	})
}

func (r *WardrobeItemRepository) MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error) {
	return r.updateProcessingState(ctx, itemID, processingVersion, map[string]any{
		"status":     wardrobestatus.NeedsReview,
		"updated_at": time.Now().UTC(),
	})
}

func (r *WardrobeItemRepository) CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	if updates == nil {
		updates = make(map[string]any)
	}
	updates["status"] = wardrobestatus.InWardrobe
	updates["updated_at"] = time.Now().UTC()

	return r.updateProcessingState(ctx, itemID, processingVersion, updates)
}

func (r *WardrobeItemRepository) updateProcessingState(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	if updates == nil {
		updates = make(map[string]any)
	}

	returnValue := false
	err := r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		wardrobeUpdates := map[string]any{}
		if status, ok := updates["status"]; ok {
			wardrobeUpdates["status"] = status
		}
		if updatedAt, ok := updates["updated_at"]; ok {
			wardrobeUpdates["updated_at"] = updatedAt
		}
		if len(wardrobeUpdates) == 0 {
			return nil
		}
		result := tx.Model(&entities.WardrobeItem{}).
			Where("id = ? AND is_deleted = ? AND status = ?", itemID, false, wardrobestatus.Processing).
			Updates(wardrobeUpdates)
		if result.Error != nil {
			return result.Error
		}
		returnValue = result.RowsAffected > 0
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

func (r *WardrobeItemRepository) buildUserWardrobeFilterQuery(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filterByFashionItems bool, withPreload bool) *gorm.DB {
	var query *gorm.DB
	if withPreload {
		query = r.GetQueryWithPreload(ctx)
	} else {
		query = r.GetDB(ctx).Model(&entities.WardrobeItem{})
	}

	query = query.Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = ?", userID, false)
	if filterByFashionItems {
		query = query.Where("wardrobe_items.fashion_item_id IN ?", fashionItemIDs)
	}
	if len(statuses) > 0 {
		query = query.Where("wardrobe_items.status IN ?", statuses)
	}

	return query
}

// buildWardrobeItemSearchQuery builds a base query for searching wardrobe items
// filtered by the given fashion_item_ids and item type (no cross-module joins).
// When filterByFashionItems is true the fashion filter is always applied — even
// for an empty list — so "no match" yields zero rows instead of "no filter".
func buildWardrobeItemSearchQuery(db *gorm.DB, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType) *gorm.DB {
	db = db.Where("wardrobe_items.item_type = ? AND wardrobe_items.is_deleted = ?", itemType, false)
	if filterByFashionItems {
		db = db.Where("wardrobe_items.fashion_item_id IN ?", fashionItemIDs)
	}
	return db
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

func (r *WardrobeItemRepository) GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*entities.WardrobeItem, error) {
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
	return items, nil
}

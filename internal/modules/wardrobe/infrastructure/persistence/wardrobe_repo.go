package persistence

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WardrobeItemRepository struct {
	shared_persist.GenericRepository[entities.WardrobeItem, uuid.UUID]
}

func NewWardrobeItemRepository(db *gorm.DB) repositories.IWardrobeItemRepository {
	relations := []string{"User", "Category"}
	return &WardrobeItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.WardrobeItem, uuid.UUID](db, relations),
	}
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

func (r *WardrobeItemRepository) GetByUserID(ctx context.Context, userID uuid.UUID, categorySlug *string) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetQueryWithPreload(ctx).
		Where("user_id = ? AND is_deleted = ? AND status = ?", userID, false, wardrobestatus.InWardrobe)
	if categorySlug != nil && *categorySlug != "" {
		query = query.Joins("JOIN categories ON categories.id = wardrobe_items.category_id").Where("categories.slug = ?", *categorySlug)
	}
	err := query.Order("created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, categorySlug *string, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetQueryWithPreload(ctx).
		Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = ? AND wardrobe_items.status = ?", userID, false, wardrobestatus.InWardrobe)
	if categorySlug != nil && *categorySlug != "" {
		query = query.Joins("JOIN categories ON categories.id = wardrobe_items.category_id").Where("categories.slug = ?", *categorySlug)
	}
	db := shared_persist.ApplyPagination(query, pagination)
	err := db.Order("wardrobe_items.created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetPendingByUserIDPaginated(ctx context.Context, userID uuid.UUID, status *wardrobestatus.WardrobeItemStatus, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetQueryWithPreload(ctx).Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = ?", userID, false)
	if status != nil {
		query = query.Where("wardrobe_items.status = ?", *status)
	} else {
		query = query.Where("wardrobe_items.status IN ?", []wardrobestatus.WardrobeItemStatus{
			wardrobestatus.Processing,
			wardrobestatus.Failed,
			wardrobestatus.NeedsReview,
		})
	}
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
	return r.GetDB(ctx).Create(&items).Error
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

func (r *WardrobeItemRepository) GetItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	db := r.GetQueryWithPreload(ctx)

	db = db.Where("item_type = ? AND wardrobe_items.is_deleted = ?", itemType, false)
	if categorySlug != nil && *categorySlug != "" {
		db = db.Joins("JOIN categories ON categories.id = wardrobe_items.category_id").Where("categories.slug = ?", *categorySlug)
	}

	if query != nil && *query != "" {
		queryStr := strings.ToLower(*query)

		db = db.Where(r.GetDB(ctx).
			Where("category_id IN (SELECT id FROM categories WHERE LOWER(name) LIKE ?)", "%"+queryStr+"%").
			Or("LOWER(color) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(style) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(material) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(pattern) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(fit) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(seasonality) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(description) LIKE ?", "%"+queryStr+"%"),
		)
	}

	err := db.Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetItemsPaginated(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	db := r.GetQueryWithPreload(ctx)

	db = db.Where("item_type = ? AND wardrobe_items.is_deleted = ?", itemType, false)
	if categorySlug != nil && *categorySlug != "" {
		db = db.Joins("JOIN categories ON categories.id = wardrobe_items.category_id").Where("categories.slug = ?", *categorySlug)
	}

	if query != nil && *query != "" {
		queryStr := strings.ToLower(*query)

		db = db.Where(r.GetDB(ctx).
			Where("category_id IN (SELECT id FROM categories WHERE LOWER(name) LIKE ?)", "%"+queryStr+"%").
			Or("LOWER(color) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(style) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(material) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(pattern) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(fit) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(seasonality) LIKE ?", "%"+queryStr+"%").
			Or("LOWER(description) LIKE ?", "%"+queryStr+"%"),
		)
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
		Where("status = ? AND created_at < NOW() - INTERVAL '7 days'", wardrobestatus.Failed).
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	err := r.GetQueryWithPreload(ctx).
		Where("status = ? AND is_deleted = ? AND COALESCE(last_processing_attempt_at, processing_started_at, created_at) < ?", wardrobestatus.Processing, false, staleBefore).
		Order("created_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.WardrobeItem, bool, error) {
	updates := map[string]any{
		"status":                     wardrobestatus.Processing,
		"review_reason":              nil,
		"processing_error_reason":    nil,
		"processing_retry_count":     0,
		"processing_started_at":      now,
		"last_processing_attempt_at": now,
		"processing_version":         gorm.Expr("processing_version + 1"),
		"updated_at":                 now,
	}

	result := r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Where("id = ? AND user_id = ? AND is_deleted = ? AND status IN ?", itemID, userID, false, []wardrobestatus.WardrobeItemStatus{
			wardrobestatus.Failed,
			wardrobestatus.NeedsReview,
		}).
		Updates(updates)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}

	item, err := r.GetByID(ctx, itemID)
	return item, true, err
}

func (r *WardrobeItemRepository) ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.WardrobeItem, bool, error) {
	updates := map[string]any{
		"processing_retry_count":     gorm.Expr("processing_retry_count + 1"),
		"last_processing_attempt_at": now,
		"processing_version":         gorm.Expr("processing_version + 1"),
		"updated_at":                 now,
	}

	result := r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Where("id = ? AND is_deleted = ? AND status = ? AND processing_version = ? AND COALESCE(last_processing_attempt_at, processing_started_at, created_at) < ?",
			itemID, false, wardrobestatus.Processing, processingVersion, staleBefore).
		Updates(updates)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}

	item, err := r.GetByID(ctx, itemID)
	return item, true, err
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
	result := r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Where("id = ? AND is_deleted = ? AND status = ? AND processing_version = ?",
			itemID, false, wardrobestatus.Processing, processingVersion).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
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

// GetHybridCandidates performs a multi-stage hybrid search combining:
// - Vector search (cosine similarity on pgvector embeddings)
// - Lexical search (full-text search GIN index on text attributes)
// - Fallback options (if one of them is empty or fails)
// The results are fused using a weighted combination:
// Score = 0.7 * (1.0 - Cosine Distance) + 0.3 * (Text Search Rank)
func (r *WardrobeItemRepository) GetHybridCandidates(
	ctx context.Context,
	userID uuid.UUID,
	semanticVector entities.Vector,
	keywords []string,
	limit int,
) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	// Base query filtering active, non-deleted wardrobe items owned by the specified user
	db := r.GetQueryWithPreload(ctx).
		Where("wardrobe_items.user_id = ? AND wardrobe_items.status = ? AND wardrobe_items.is_deleted = ?",
			userID, wardrobestatus.InWardrobe, false)

	hasVector := len(semanticVector) > 0
	hasKeywords := len(keywords) > 0

	// Case 1: Both vector and keyword search parameters are available (Hybrid Search)
	if hasVector && hasKeywords {
		kwStr := strings.Join(keywords, " ")
		// Weighted score: 70% weight for semantic vector similarity, 30% for full-text search rank (ts_rank_cd)
		// <=> operator calculates cosine distance, so (1.0 - distance) yields cosine similarity
		// simple config is utilized for fast matching without language-specific stemming
		err := db.Order(gorm.Expr(
			"0.7 * (1.0 - (wardrobe_items.embedding <=> ?)) + 0.3 * ts_rank_cd(to_tsvector('simple', coalesce(wardrobe_items.color, '') || ' ' || coalesce(wardrobe_items.style, '') || ' ' || coalesce(wardrobe_items.material, '') || ' ' || coalesce(wardrobe_items.pattern, '') || ' ' || coalesce(wardrobe_items.fit, '') || ' ' || coalesce(wardrobe_items.description, '')), plainto_tsquery('simple', ?)) DESC",
			semanticVector, kwStr,
		)).
		Limit(limit).
		Find(&items).Error
		return items, err
	} else if hasVector {
		// Case 2: Only vector search is available (pure semantic retrieval ordered by closest cosine distance)
		err := db.Order(gorm.Expr("wardrobe_items.embedding <=> ?", semanticVector)).
			Limit(limit).
			Find(&items).Error
		return items, err
	} else if hasKeywords {
		// Case 3: Only keywords are available (pure lexical retrieval utilizing GIN index matching with @@)
		kwStr := strings.Join(keywords, " ")
		err := db.Where(gorm.Expr(
			"to_tsvector('simple', coalesce(wardrobe_items.color, '') || ' ' || coalesce(wardrobe_items.style, '') || ' ' || coalesce(wardrobe_items.material, '') || ' ' || coalesce(wardrobe_items.pattern, '') || ' ' || coalesce(wardrobe_items.fit, '') || ' ' || coalesce(wardrobe_items.description, '')) @@ plainto_tsquery('simple', ?)",
			kwStr,
		)).
		Limit(limit).
		Find(&items).Error
		return items, err
	}

	// Case 4: Fallback retrieval (default to most recently created items)
	err := db.Order("wardrobe_items.created_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}


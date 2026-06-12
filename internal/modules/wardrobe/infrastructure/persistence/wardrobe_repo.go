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
		Where("user_id = ? AND status NOT IN (?, ?)", userID, wardrobestatus.Processing, wardrobestatus.Failed).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) GetByUserID(ctx context.Context, userID uuid.UUID, categorySlug *string) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	query := r.GetQueryWithPreload(ctx).Where("user_id = ? AND is_deleted = ?", userID, false)
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
	query := r.GetQueryWithPreload(ctx).Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = ?", userID, false)
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

func (r *WardrobeItemRepository) CountByUserIDAndCategory(ctx context.Context, userID uuid.UUID, categorySlug *string) (int64, error) {
	var count int64
	query := r.GetDB(ctx).Model(&entities.WardrobeItem{}).Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = ?", userID, false)
	if categorySlug != nil && *categorySlug != "" {
		query = query.Joins("JOIN categories ON categories.id = wardrobe_items.category_id").Where("categories.slug = ?", *categorySlug)
	}
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
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

func (r *WardrobeItemRepository) CountItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) (int64, error) {
	var count int64
	db := r.GetDB(ctx).Model(&entities.WardrobeItem{})

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

	err := db.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
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

func (r *WardrobeItemRepository) TouchLastUsedAt(ctx context.Context, ids []uuid.UUID, usedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}

	return r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Where("id IN ?", ids).
		Update("last_used_at", usedAt).Error
}

func (r *WardrobeItemRepository) GetSimilarItemsByVectorAndCategory(
	ctx context.Context,
	userID uuid.UUID,
	categoryID uuid.UUID,
	vector entities.Vector,
	limit int,
) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	err := r.GetQueryWithPreload(ctx).
		Where("user_id = ? AND category_id = ? AND status = ? AND is_deleted = ?", userID, categoryID, wardrobestatus.InWardrobe, false).
		Order(gorm.Expr("embedding <=> ?", vector)).
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) GetRecentlyActiveItemsByCategory(
	ctx context.Context,
	userID uuid.UUID,
	categoryID uuid.UUID,
	limit int,
) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	err := r.GetQueryWithPreload(ctx).
		Where("user_id = ? AND category_id = ? AND status = ? AND is_deleted = ?", userID, categoryID, wardrobestatus.InWardrobe, false).
		Order("last_used_at DESC NULLS LAST, created_at DESC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
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


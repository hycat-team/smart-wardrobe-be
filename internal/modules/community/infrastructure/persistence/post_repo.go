package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/community/domain/dto"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type feedRow struct {
	entities.Post
	GlobalHotnessScore float64 `gorm:"column:global_hotness_score"`
}

type PostRepository struct {
	shared_persist.GenericRepository[entities.Post, uuid.UUID]
}

func NewPostRepository(db *gorm.DB) repositories.IPostRepository {
	return &PostRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Post, uuid.UUID](db, []string{"User"}),
	}
}

func (r *PostRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Post, error) {
	var post entities.Post
	err := r.GetQueryWithPreload(ctx).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.id = ? AND posts.is_deleted = ? AND users.is_deleted = ?", id, false, false).
		First(&post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) GetByPublicID(ctx context.Context, publicID string) (*entities.Post, error) {
	var post entities.Post
	err := r.GetQueryWithPreload(ctx).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.public_id = ? AND posts.is_deleted = ? AND users.is_deleted = ?", publicID, false, false).
		First(&post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) baseFeedQuery(ctx context.Context) *gorm.DB {
	return r.GetDB(ctx).
		Model(&entities.Post{}).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.is_deleted = ? AND users.is_deleted = ?", false, false).
		Where("posts.post_type <> ? OR EXISTS (SELECT 1 FROM post_items WHERE post_items.post_id = posts.id AND post_items.status <> ?)", "SALE", postitemstatus.Hidden)
}

func (r *PostRepository) GetFeed(ctx context.Context, query dto.FeedQuery) (*dto.FeedResult, error) {
	pagination := shared_persist.NormalizePagination(shared_dto.PaginationQuery{
		Page:  query.Page,
		Limit: query.Limit,
	})
	baseQuery := r.baseFeedQuery(ctx)

	if query.Username != nil {
		baseQuery = baseQuery.Where("users.username = ?", *query.Username)
	}
	if query.PostType != nil {
		baseQuery = baseQuery.Where("posts.post_type = ?", *query.PostType)
	}

	var totalItems int64
	if err := baseQuery.Count(&totalItems).Error; err != nil {
		return nil, err
	}

	items, err := r.listFeed(ctx, query, false, query.Limit)
	if err != nil {
		return nil, err
	}

	return &dto.FeedResult{
		Items:    items,
		Metadata: shared_persist.BuildPaginationMetadata(pagination, totalItems),
	}, nil
}

func (r *PostRepository) GetHotFeedCandidates(ctx context.Context, query dto.FeedQuery, limit int) ([]*dto.FeedPostRecord, error) {
	return r.listFeed(ctx, query, true, limit)
}

func (r *PostRepository) listFeed(ctx context.Context, query dto.FeedQuery, forceHot bool, limit int) ([]*dto.FeedPostRecord, error) {
	dbQuery := r.baseFeedQuery(ctx).
		Select("posts.*, COALESCE(post_score_snapshots.global_hotness_score, 0) AS global_hotness_score").
		Joins("LEFT JOIN post_score_snapshots ON post_score_snapshots.post_id = posts.id")

	if query.Username != nil {
		dbQuery = dbQuery.Where("users.username = ?", *query.Username)
	}
	if query.PostType != nil {
		dbQuery = dbQuery.Where("posts.post_type = ?", *query.PostType)
	}

	if forceHot || query.Sort == "hot" {
		dbQuery = dbQuery.Order("COALESCE(post_score_snapshots.global_hotness_score, 0) DESC").Order("posts.created_at DESC")
	} else {
		dbQuery = dbQuery.Order("posts.created_at DESC")
	}

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}
	if !forceHot {
		dbQuery = shared_persist.ApplyPagination(dbQuery, shared_dto.PaginationQuery{
			Page:  query.Page,
			Limit: query.Limit,
		})
	}

	var rows []*feedRow
	if err := dbQuery.Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*dto.FeedPostRecord, 0, len(rows))
	for _, row := range rows {
		post := row.Post
		result = append(result, &dto.FeedPostRecord{
			Post:               &post,
			GlobalHotnessScore: row.GlobalHotnessScore,
		})
	}
	return result, nil
}

func (r *PostRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Post, error) {
	var items []*entities.Post
	err := r.GetQueryWithPreload(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *PostRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var posts []*entities.Post
	err := r.GetQueryWithPreload(ctx).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.id IN ? AND posts.is_deleted = ? AND users.is_deleted = ?", ids, false, false).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *PostRepository) GetDetail(ctx context.Context, postPublicID string) (*entities.Post, []*entities.PostItem, []*entities.PostMedia, error) {
	post, err := r.GetByPublicID(ctx, postPublicID)
	if err != nil || post == nil {
		return post, nil, nil, err
	}

	var items []*entities.PostItem
	if err := r.GetDB(ctx).
		Preload("WardrobeItem").
		Preload("WardrobeItem.Category").
		Where("post_id = ? AND status <> ?", post.ID, postitemstatus.Hidden).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, nil, nil, err
	}

	if post.PostType == "SALE" && len(items) == 0 {
		return nil, nil, nil, nil
	}

	var media []*entities.PostMedia
	if err := r.GetDB(ctx).
		Where("post_id = ?", post.ID).
		Order("sort_order ASC, created_at ASC").
		Find(&media).Error; err != nil {
		return nil, nil, nil, err
	}

	return post, items, media, nil
}

func (r *PostRepository) GetDirtyPostIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	var postIDs []uuid.UUID
	query := r.GetDB(ctx).
		Model(&entities.Post{}).
		Where("hotness_dirty_at IS NOT NULL AND is_deleted = ?", false).
		Order("hotness_dirty_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Pluck("id", &postIDs).Error; err != nil {
		return nil, err
	}
	return postIDs, nil
}

func (r *PostRepository) GetDecayRefreshPostIDs(ctx context.Context, since time.Time, limit int) ([]uuid.UUID, error) {
	var postIDs []uuid.UUID
	query := r.GetDB(ctx).
		Model(&entities.Post{}).
		Where("created_at >= ? AND is_deleted = ?", since, false).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Pluck("id", &postIDs).Error; err != nil {
		return nil, err
	}
	return postIDs, nil
}

func (r *PostRepository) GetHighScoreStalePostIDs(ctx context.Context, before time.Time, minScore float64, limit int) ([]uuid.UUID, error) {
	var postIDs []uuid.UUID
	query := r.GetDB(ctx).
		Model(&entities.Post{}).
		Select("posts.id").
		Joins("JOIN post_score_snapshots ON post_score_snapshots.post_id = posts.id").
		Where("posts.created_at < ? AND posts.is_deleted = ?", before, false).
		Where("post_score_snapshots.global_hotness_score >= ?", minScore).
		Order("post_score_snapshots.global_hotness_score DESC").
		Order("posts.created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Pluck("posts.id", &postIDs).Error; err != nil {
		return nil, err
	}
	return postIDs, nil
}

func (r *PostRepository) MarkHotnessDirty(ctx context.Context, postID uuid.UUID) error {
	now := time.Now()
	return r.GetDB(ctx).
		Model(&entities.Post{}).
		Where("id = ? AND is_deleted = ?", postID, false).
		Update("hotness_dirty_at", now).Error
}

func (r *PostRepository) ClearHotnessDirty(ctx context.Context, postIDs []uuid.UUID) error {
	if len(postIDs) == 0 {
		return nil
	}

	return r.GetDB(ctx).
		Model(&entities.Post{}).
		Where("id IN ? AND is_deleted = ?", postIDs, false).
		Update("hotness_dirty_at", nil).Error
}

func (r *PostRepository) SoftDelete(ctx context.Context, postID uuid.UUID) error {
	return r.GetDB(ctx).
		Model(&entities.Post{}).
		Where("id = ? AND is_deleted = ?", postID, false).
		Update("is_deleted", true).Error
}

func (r *PostRepository) Restore(ctx context.Context, postID uuid.UUID) error {
	return r.GetDB(ctx).
		Model(&entities.Post{}).
		Where("id = ? AND is_deleted = ?", postID, true).
		Update("is_deleted", false).Error
}

func (r *PostRepository) GetPostsForAdmin(ctx context.Context, filter repositories.AdminPostFilter) (*repositories.AdminPostListResult, error) {
	db := r.GetDB(ctx).Model(&entities.Post{}).Preload("User")

	if filter.IsDeleted != nil {
		db = db.Where("posts.is_deleted = ?", *filter.IsDeleted)
	}

	if filter.PostType != nil && *filter.PostType != "" {
		db = db.Where("posts.post_type = ?", *filter.PostType)
	}

	if filter.Query != nil && *filter.Query != "" {
		qStr := "%" + strings.ToLower(*filter.Query) + "%"
		db = db.Where("LOWER(posts.content) LIKE ? OR LOWER(posts.title) LIKE ?", qStr, qStr)
	}

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if offset < 0 {
		offset = 0
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	var posts []*entities.Post
	err := db.Order("posts.created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return &repositories.AdminPostListResult{
		Posts:      posts,
		TotalCount: totalCount,
	}, nil
}

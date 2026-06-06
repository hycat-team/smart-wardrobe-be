package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
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

func (r *PostRepository) GetFeed(ctx context.Context, query repositories.FeedQuery) (*repositories.FeedResult, error) {
	pagination := shared_persist.NormalizePagination(shared_dto.PaginationQuery{
		Page:  query.Page,
		Limit: query.Limit,
	})
	baseQuery := r.GetDB(ctx).
		Model(&entities.Post{})

	if query.UserID != nil {
		baseQuery = baseQuery.Where("posts.user_id = ?", *query.UserID)
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

	return &repositories.FeedResult{
		Items:    items,
		Metadata: shared_persist.BuildPaginationMetadata(pagination, totalItems),
	}, nil
}

func (r *PostRepository) GetHotFeedCandidates(ctx context.Context, query repositories.FeedQuery, limit int) ([]*repositories.FeedPostRecord, error) {
	return r.listFeed(ctx, query, true, limit)
}

func (r *PostRepository) listFeed(ctx context.Context, query repositories.FeedQuery, forceHot bool, limit int) ([]*repositories.FeedPostRecord, error) {
	dbQuery := r.GetDB(ctx).
		Model(&entities.Post{}).
		Select("posts.*, COALESCE(post_score_snapshots.global_hotness_score, 0) AS global_hotness_score").
		Joins("LEFT JOIN post_score_snapshots ON post_score_snapshots.post_id = posts.id")

	if query.UserID != nil {
		dbQuery = dbQuery.Where("posts.user_id = ?", *query.UserID)
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

	result := make([]*repositories.FeedPostRecord, 0, len(rows))
	for _, row := range rows {
		post := row.Post
		result = append(result, &repositories.FeedPostRecord{
			Post:               &post,
			GlobalHotnessScore: row.GlobalHotnessScore,
		})
	}
	return result, nil
}

func (r *PostRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Post, error) {
	var items []*entities.Post
	err := r.GetQueryWithPreload(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *PostRepository) GetDetail(ctx context.Context, postID uuid.UUID) (*entities.Post, []*entities.PostItem, []*entities.PostMedia, error) {
	post, err := r.GetByID(ctx, postID)
	if err != nil || post == nil {
		return post, nil, nil, err
	}

	var items []*entities.PostItem
	if err := r.GetDB(ctx).
		Preload("WardrobeItem").
		Preload("WardrobeItem.Category").
		Where("post_id = ?", postID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, nil, nil, err
	}

	var media []*entities.PostMedia
	if err := r.GetDB(ctx).Where("post_id = ?", postID).Order("sort_order ASC, created_at ASC").Find(&media).Error; err != nil {
		return nil, nil, nil, err
	}

	return post, items, media, nil
}

package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedsort"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedtype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/hiddenreason"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepository struct {
	shared_persist.GenericRepository[entities.Post, uuid.UUID]
}

func NewPostRepository(db *gorm.DB) repositories.IPostRepository {
	return &PostRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Post, uuid.UUID](db, []string{"Media"}),
	}
}

// mediaOrder keeps post media in the sequence supplied by the author.
func mediaOrder(db *gorm.DB) *gorm.DB {
	return db.Order("post_media.sort_order ASC, post_media.id ASC")
}

func (r *PostRepository) GetByPublicID(ctx context.Context, publicID string) (*entities.Post, error) {
	var post entities.Post
	if err := r.GetDB(ctx).Preload("Media", mediaOrder).Where("public_id = ?", publicID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) ListFeed(ctx context.Context, filter repositories.PostFilter) (*repositories.PostListResult, error) {
	db := r.GetDB(ctx).Model(&entities.Post{}).Where("posts.status = ?", poststatus.Published)

	if filter.PostType != nil {
		db = db.Where("posts.post_type = ?", *filter.PostType)
	}
	if filter.AuthorUserID != nil {
		db = db.Where("posts.user_id = ?", *filter.AuthorUserID)
	}
	if filter.Type == feedtype.Following && filter.FollowingOf != nil {
		sub := r.GetDB(ctx).Model(&entities.Follow{}).Select("followee_id").Where("follower_id = ?", *filter.FollowingOf)
		db = db.Where("posts.user_id IN (?)", sub)
	}
	if filter.Sort == feedsort.Hot {
		db = db.Where("posts.created_at >= now() - make_interval(days => ?)", filter.HotWindowDays)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	db = shared_persist.ApplyPagination(db, shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit})

	var posts []*entities.Post
	query := db.Preload("Media", mediaOrder).Order(postOrderClause(filter)).Find(&posts)
	if query.Error != nil {
		return nil, query.Error
	}
	return &repositories.PostListResult{Items: posts, TotalCount: total}, nil
}

// postOrderClause builds the SQL ordering. Hot uses a query-time time-decay
// score; weights/gravity come from validated numeric config, so inlining them
// via fmt is safe (no user input).
func postOrderClause(filter repositories.PostFilter) string {
	if filter.Sort != feedsort.Hot {
		return "posts.created_at DESC, posts.id DESC"
	}
	return fmt.Sprintf(
		"((posts.like_count * %d + posts.comment_count * %d + 1) / POWER(EXTRACT(EPOCH FROM (now() - posts.created_at)) / 3600 + 2, %.6f)) DESC, posts.id DESC",
		filter.LikeWeight, filter.CommentWeight, filter.HotGravity,
	)
}

func (r *PostRepository) ListByAuthorPaginated(ctx context.Context, authorID uuid.UUID, viewerUserID *uuid.UUID, pagination shared_dto.PaginationQuery) (*repositories.PostListResult, error) {
	db := r.GetDB(ctx).Model(&entities.Post{}).Where("posts.user_id = ?", authorID)
	if viewerUserID != nil && *viewerUserID == authorID {
		db = db.Where("posts.status IN ?", []poststatus.PostStatus{poststatus.Published, poststatus.Hidden})
	} else {
		db = db.Where("posts.status = ?", poststatus.Published)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	db = shared_persist.ApplyPagination(db, pagination)

	var posts []*entities.Post
	if err := db.Preload("Media", mediaOrder).Order("posts.created_at DESC, posts.id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return &repositories.PostListResult{Items: posts, TotalCount: total}, nil
}

func (r *PostRepository) Search(ctx context.Context, filter repositories.SearchPostsFilter) (*repositories.PostListResult, error) {
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	if keyword == "" {
		return &repositories.PostListResult{Items: []*entities.Post{}, TotalCount: 0}, nil
	}
	prefix := filter.ContentPrefixChars
	if prefix <= 0 {
		prefix = 200
	}
	pattern := "%" + escapeLike(keyword) + "%"

	db := r.GetDB(ctx).Model(&entities.Post{}).
		Where("posts.status = ?", poststatus.Published).
		Where("LOWER(left(posts.content, ?)) LIKE ? ESCAPE '\\'", prefix, pattern).
		Where("posts.created_at >= now() - make_interval(days => ?)", filter.HotWindowDays)
	if filter.PostType != nil {
		db = db.Where("posts.post_type = ?", *filter.PostType)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	db = shared_persist.ApplyPagination(db, shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit})

	order := postOrderClause(repositories.PostFilter{
		Sort:          feedsort.Hot,
		LikeWeight:    filter.LikeWeight,
		CommentWeight: filter.CommentWeight,
		HotGravity:    filter.HotGravity,
	})
	var posts []*entities.Post
	if err := db.Preload("Media", mediaOrder).Order(order).Find(&posts).Error; err != nil {
		return nil, err
	}
	return &repositories.PostListResult{Items: posts, TotalCount: total}, nil
}

// escapeLike escapes LIKE wildcards so user input is matched literally.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (r *PostRepository) AdminList(ctx context.Context, filter repositories.AdminPostFilter) (*repositories.PostListResult, error) {
	db := r.GetDB(ctx).Model(&entities.Post{})
	if filter.Status != nil {
		db = db.Where("posts.status = ?", *filter.Status)
	}
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	if keyword != "" {
		prefix := filter.ContentPrefixChars
		if prefix <= 0 {
			prefix = 200
		}
		pattern := "%" + escapeLike(keyword) + "%"
		db = db.Where(
			"(LOWER(left(posts.content, ?)) LIKE ? ESCAPE '\\' OR LOWER(COALESCE(posts.title, '')) LIKE ? ESCAPE '\\')",
			prefix, pattern, pattern,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	db = shared_persist.ApplyPagination(db, shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit})

	var posts []*entities.Post
	if err := db.Preload("Media", mediaOrder).Order("posts.created_at DESC, posts.id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return &repositories.PostListResult{Items: posts, TotalCount: total}, nil
}

func (r *PostRepository) IncrementLikeCount(ctx context.Context, postID uuid.UUID, delta int) error {
	return r.GetDB(ctx).Model(&entities.Post{}).Where("id = ?", postID).
		UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", delta)).Error
}

func (r *PostRepository) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error {
	return r.GetDB(ctx).Model(&entities.Post{}).Where("id = ?", postID).
		UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error
}

// RecountCommentCount rebuilds comment_count from the comments table. It is
// used by admin moderation, where several rows change state at once (a root
// plus its replies) and relative deltas would have to be bookkept per
// transition. The post row is locked first so a concurrent relative increment
// (a user adding or deleting a comment) either commits before the count, or
// waits for the lock and applies on top of the recounted value. Counting
// without the lock would let one of the two writes overwrite the other.
func (r *PostRepository) RecountCommentCount(ctx context.Context, postID uuid.UUID) error {
	db := r.GetDB(ctx)
	var lock entities.Post
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").Where("id = ?", postID).Take(&lock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return db.Exec(
		"UPDATE posts SET comment_count = (SELECT COUNT(*) FROM post_comments WHERE post_id = ? AND status = ?) WHERE id = ?",
		postID, poststatus.Published, postID,
	).Error
}

// UpdateEditableFields writes only the author-editable columns. A whole-entity
// Save would also write like_count/comment_count from the copy read before the
// transaction, silently discarding a like or comment committed in between.
func (r *PostRepository) UpdateEditableFields(ctx context.Context, postID uuid.UUID, title *string, content string, outfitID *uuid.UUID) error {
	return r.GetDB(ctx).Model(&entities.Post{}).Where("id = ?", postID).Updates(map[string]any{
		"title":     title,
		"content":   content,
		"outfit_id": outfitID,
	}).Error
}

// ReconcileCounters rebuilds the denormalized counters from their source
// tables (feature 022 R12). Only drifted rows are written, so a healthy
// registry costs reads and no updates; updated_at is deliberately not touched
// because it describes the post content, not its counters.
const reconcilePostCountersSQL = `
WITH like_counts AS (
    SELECT post_id, COUNT(*) AS c FROM post_likes GROUP BY post_id
), comment_counts AS (
    SELECT post_id, COUNT(*) AS c FROM post_comments WHERE status = ? GROUP BY post_id
)
UPDATE posts p
SET like_count = COALESCE(l.c, 0),
    comment_count = COALESCE(cc.c, 0)
FROM posts p2
LEFT JOIN like_counts l ON l.post_id = p2.id
LEFT JOIN comment_counts cc ON cc.post_id = p2.id
WHERE p.id = p2.id
  AND (p.like_count IS DISTINCT FROM COALESCE(l.c, 0)
    OR p.comment_count IS DISTINCT FROM COALESCE(cc.c, 0))`

func (r *PostRepository) ReconcileCounters(ctx context.Context) error {
	return r.GetDB(ctx).Exec(reconcilePostCountersSQL, poststatus.Published).Error
}

func (r *PostRepository) UpdateModerationStatus(ctx context.Context, postID uuid.UUID, status poststatus.PostStatus, moderatedBy *uuid.UUID, hiddenReason *hiddenreason.HiddenReason) error {
	updates := map[string]any{
		"status":        status,
		"moderated_by":  moderatedBy,
		"hidden_reason": hiddenReason,
		"updated_at":    time.Now(),
	}
	return r.GetDB(ctx).Model(&entities.Post{}).Where("id = ?", postID).Updates(updates).Error
}

func (r *PostRepository) SoftDeleteByOutfitID(ctx context.Context, outfitID uuid.UUID) (int64, error) {
	res := r.GetDB(ctx).Model(&entities.Post{}).
		Where("outfit_id = ? AND status <> ?", outfitID, poststatus.Deleted).
		Updates(map[string]any{"status": poststatus.Deleted, "updated_at": time.Now()})
	return res.RowsAffected, res.Error
}

func (r *PostRepository) HideAllByAuthor(ctx context.Context, authorID uuid.UUID) (int64, error) {
	res := r.GetDB(ctx).Model(&entities.Post{}).
		Where("user_id = ? AND status = ?", authorID, poststatus.Published).
		Updates(map[string]any{
			"status":        poststatus.Hidden,
			"hidden_reason": hiddenreason.AuthorDisabled,
			"updated_at":    time.Now(),
		})
	return res.RowsAffected, res.Error
}

func (r *PostRepository) RestoreAllByAuthor(ctx context.Context, authorID uuid.UUID) (int64, error) {
	res := r.GetDB(ctx).Model(&entities.Post{}).
		Where("user_id = ? AND status = ? AND hidden_reason = ?", authorID, poststatus.Hidden, hiddenreason.AuthorDisabled).
		Updates(map[string]any{
			"status":        poststatus.Published,
			"hidden_reason": nil,
			"updated_at":    time.Now(),
		})
	return res.RowsAffected, res.Error
}

func (r *PostRepository) CountByAuthor(ctx context.Context, authorID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.Post{}).
		Where("user_id = ? AND status = ?", authorID, poststatus.Published).
		Count(&count).Error
	return count, err
}

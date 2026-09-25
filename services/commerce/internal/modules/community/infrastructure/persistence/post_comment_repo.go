package persistence

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostCommentRepository struct {
	shared_persist.GenericRepository[entities.PostComment, uuid.UUID]
}

func NewPostCommentRepository(db *gorm.DB) repositories.IPostCommentRepository {
	return &PostCommentRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.PostComment, uuid.UUID](db, nil),
	}
}

// GetRoots returns top-level comments (newest first). Deleted roots are
// included so the use case can render a "deleted" placeholder when they still
// have replies; hidden/deleted-without-replies are filtered by the use case.
func (r *PostCommentRepository) GetRoots(ctx context.Context, postID uuid.UUID) ([]*entities.PostComment, error) {
	var comments []*entities.PostComment
	err := r.GetDB(ctx).
		Where("post_id = ? AND parent_comment_id IS NULL AND status IN ?", postID, []poststatus.PostStatus{poststatus.Published, poststatus.Deleted}).
		Order("created_at DESC").
		Find(&comments).Error
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *PostCommentRepository) GetReplies(ctx context.Context, rootID uuid.UUID) ([]*entities.PostComment, error) {
	var replies []*entities.PostComment
	err := r.GetDB(ctx).
		Where("root_comment_id = ? AND parent_comment_id IS NOT NULL AND status = ?", rootID, poststatus.Published).
		Order("created_at ASC").
		Find(&replies).Error
	if err != nil {
		return nil, err
	}
	return replies, nil
}

func (r *PostCommentRepository) CountRepliesByRootIDs(ctx context.Context, rootIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	result := make(map[uuid.UUID]int64, len(rootIDs))
	if len(rootIDs) == 0 {
		return result, nil
	}
	type row struct {
		RootCommentID uuid.UUID
		Total         int64
	}
	var rows []row
	err := r.GetDB(ctx).Model(&entities.PostComment{}).
		Select("root_comment_id, COUNT(*) AS total").
		Where("root_comment_id IN ? AND parent_comment_id IS NOT NULL AND status = ?", rootIDs, poststatus.Published).
		Group("root_comment_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.RootCommentID] = r.Total
	}
	return result, nil
}

func (r *PostCommentRepository) AdminList(ctx context.Context, filter repositories.AdminCommentFilter) (*repositories.CommentListResult, error) {
	db := r.GetDB(ctx).Model(&entities.PostComment{})
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	if keyword != "" {
		pattern := "%" + escapeLike(keyword) + "%"
		db = db.Where("LOWER(content) LIKE ? ESCAPE '\\'", pattern)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	db = shared_persist.ApplyPagination(db, shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit})

	var comments []*entities.PostComment
	if err := db.Order("created_at DESC").Find(&comments).Error; err != nil {
		return nil, err
	}
	return &repositories.CommentListResult{Items: comments, TotalCount: total}, nil
}

// UpdateStatusByRoot cascades a moderation status to every reply of a root
// comment (FR-025 / US6 AC3).
func (r *PostCommentRepository) UpdateStatusByRoot(ctx context.Context, rootID uuid.UUID, status poststatus.PostStatus, moderatedBy *uuid.UUID) (int64, error) {
	res := r.GetDB(ctx).Model(&entities.PostComment{}).
		Where("parent_comment_id = ?", rootID).
		Updates(map[string]any{
			"status":       status,
			"moderated_by": moderatedBy,
			"updated_at":   time.Now(),
		})
	return res.RowsAffected, res.Error
}

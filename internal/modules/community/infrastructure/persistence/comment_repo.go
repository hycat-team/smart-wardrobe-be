package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommentRepository struct {
	shared_persist.GenericRepository[entities.Comment, uuid.UUID]
}

func NewCommentRepository(db *gorm.DB) repositories.ICommentRepository {
	return &CommentRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Comment, uuid.UUID](db, []string{"User"}),
	}
}

func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Comment, error) {
	var item entities.Comment
	err := r.GetQueryWithPreload(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *CommentRepository) GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.Comment, error) {
	var items []*entities.Comment
	err := r.GetQueryWithPreload(ctx).
		Joins("JOIN users ON users.id = comments.user_id").
		Where("comments.post_id = ? AND comments.is_deleted = ? AND users.is_deleted = ?", postID, false, false).
		Order("comments.created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *CommentRepository) GetTopLevelByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.Comment, error) {
	var items []*entities.Comment
	err := r.GetQueryWithPreload(ctx).
		Joins("JOIN users ON users.id = comments.user_id").
		Where("comments.post_id = ? AND comments.parent_comment_id IS NULL AND comments.is_deleted = ? AND users.is_deleted = ?", postID, false, false).
		Order("comments.created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *CommentRepository) GetRepliesByParentID(ctx context.Context, postID uuid.UUID, parentCommentID uuid.UUID) ([]*entities.Comment, error) {
	var items []*entities.Comment
	err := r.GetQueryWithPreload(ctx).
		Joins("JOIN users ON users.id = comments.user_id").
		Where("comments.post_id = ? AND comments.parent_comment_id = ? AND comments.is_deleted = ? AND users.is_deleted = ?", postID, parentCommentID, false, false).
		Order("comments.created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *CommentRepository) GetByIDAndPostID(ctx context.Context, commentID uuid.UUID, postID uuid.UUID) (*entities.Comment, error) {
	var item entities.Comment
	err := r.GetQueryWithPreload(ctx).
		Where("id = ? AND post_id = ? AND is_deleted = ?", commentID, postID, false).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *CommentRepository) SoftDelete(ctx context.Context, commentID uuid.UUID) error {
	return r.GetDB(ctx).
		Model(&entities.Comment{}).
		Where("id = ? AND is_deleted = ?", commentID, false).
		Update("is_deleted", true).Error
}

func (r *CommentRepository) SoftDeleteByParentID(ctx context.Context, parentCommentID uuid.UUID) error {
	return r.GetDB(ctx).
		Model(&entities.Comment{}).
		Where("parent_comment_id = ? AND is_deleted = ?", parentCommentID, false).
		Update("is_deleted", true).Error
}

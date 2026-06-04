package persistence

import (
	"context"

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
		GenericRepository: *shared_persist.NewGenericRepository[entities.Comment, uuid.UUID](db, nil),
	}
}

func (r *CommentRepository) GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.Comment, error) {
	var items []*entities.Comment
	err := r.GetDB(ctx).Where("post_id = ?", postID).Order("created_at ASC").Find(&items).Error
	return items, err
}

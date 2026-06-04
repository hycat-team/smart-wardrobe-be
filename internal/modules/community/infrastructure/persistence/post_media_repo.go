package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostMediaRepository struct {
	shared_persist.GenericRepository[entities.PostMedia, uuid.UUID]
}

func NewPostMediaRepository(db *gorm.DB) repositories.IPostMediaRepository {
	return &PostMediaRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.PostMedia, uuid.UUID](db, nil),
	}
}

func (r *PostMediaRepository) GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostMedia, error) {
	var items []*entities.PostMedia
	err := r.GetDB(ctx).Where("post_id = ?", postID).Order("sort_order ASC, created_at ASC").Find(&items).Error
	return items, err
}

func (r *PostMediaRepository) BulkCreate(ctx context.Context, items []*entities.PostMedia) error {
	if len(items) == 0 {
		return nil
	}
	return r.GetDB(ctx).Create(&items).Error
}

func (r *PostMediaRepository) DeleteByPostID(ctx context.Context, postID uuid.UUID) error {
	return r.GetDB(ctx).Where("post_id = ?", postID).Delete(&entities.PostMedia{}).Error
}

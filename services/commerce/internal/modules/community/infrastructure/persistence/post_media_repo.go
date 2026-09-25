package persistence

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostMediaRepository struct {
	db *gorm.DB
}

func NewPostMediaRepository(db *gorm.DB) repositories.IPostMediaRepository {
	return &PostMediaRepository{db: db}
}

func (r *PostMediaRepository) BulkCreate(ctx context.Context, medias []*entities.PostMedia) error {
	if len(medias) == 0 {
		return nil
	}
	return getDB(ctx, r.db).Create(&medias).Error
}

func (r *PostMediaRepository) DeleteByPostID(ctx context.Context, postID uuid.UUID) error {
	return getDB(ctx, r.db).Where("post_id = ?", postID).Delete(&entities.PostMedia{}).Error
}

func (r *PostMediaRepository) GetByPostIDs(ctx context.Context, postIDs []uuid.UUID) ([]*entities.PostMedia, error) {
	if len(postIDs) == 0 {
		return []*entities.PostMedia{}, nil
	}
	var medias []*entities.PostMedia
	if err := getDB(ctx, r.db).Where("post_id IN ?", postIDs).Order("post_id, sort_order").Find(&medias).Error; err != nil {
		return nil, err
	}
	return medias, nil
}

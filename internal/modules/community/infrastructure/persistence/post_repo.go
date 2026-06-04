package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository struct {
	shared_persist.GenericRepository[entities.Post, uuid.UUID]
}

func NewPostRepository(db *gorm.DB) repositories.IPostRepository {
	return &PostRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Post, uuid.UUID](db, []string{"User"}),
	}
}

func (r *PostRepository) GetFeed(ctx context.Context) ([]*entities.Post, error) {
	var items []*entities.Post
	err := r.GetQueryWithPreload(ctx).Order("created_at DESC").Find(&items).Error
	return items, err
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
	if err := r.GetDB(ctx).Preload("WardrobeItem").Preload("WardrobeItem.Category").Where("post_id = ?", postID).Find(&items).Error; err != nil {
		return nil, nil, nil, err
	}

	var media []*entities.PostMedia
	if err := r.GetDB(ctx).Where("post_id = ?", postID).Order("sort_order ASC, created_at ASC").Find(&media).Error; err != nil {
		return nil, nil, nil, err
	}

	return post, items, media, nil
}

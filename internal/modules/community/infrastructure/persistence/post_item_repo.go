package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostItemRepository struct {
	shared_persist.GenericRepository[entities.PostItem, uuid.UUID]
}

func NewPostItemRepository(db *gorm.DB) repositories.IPostItemRepository {
	return &PostItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.PostItem, uuid.UUID](db, []string{"WardrobeItem", "WardrobeItem.Category"}),
	}
}

func (r *PostItemRepository) GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostItem, error) {
	var items []*entities.PostItem
	err := r.GetQueryWithPreload(ctx).Where("post_id = ?", postID).Find(&items).Error
	return items, err
}

func (r *PostItemRepository) GetPendingByBuyerID(ctx context.Context, buyerUserID uuid.UUID) ([]*entities.PostItem, error) {
	var items []*entities.PostItem
	err := r.GetQueryWithPreload(ctx).Preload("Post").Where("buyer_user_id = ? AND transfer_state = ?", buyerUserID, transferstate.Pending).Find(&items).Error
	return items, err
}

func (r *PostItemRepository) DeleteByPostAndIDs(ctx context.Context, postID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return r.GetDB(ctx).Where("post_id = ? AND id IN ?", postID, ids).Delete(&entities.PostItem{}).Error
}

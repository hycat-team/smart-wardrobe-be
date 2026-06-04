package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LikeRepository struct {
	shared_persist.GenericRepository[entities.Like, uuid.UUID]
}

func NewLikeRepository(db *gorm.DB) repositories.ILikeRepository {
	return &LikeRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Like, uuid.UUID](db, nil),
	}
}

func (r *LikeRepository) GetPostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (*entities.Like, error) {
	var item entities.Like
	err := r.GetDB(ctx).Where("user_id = ? AND post_id = ?", userID, postID).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

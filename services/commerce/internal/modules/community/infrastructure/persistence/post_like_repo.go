package persistence

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostLikeRepository struct {
	db *gorm.DB
}

func NewPostLikeRepository(db *gorm.DB) repositories.IPostLikeRepository {
	return &PostLikeRepository{db: db}
}

func (r *PostLikeRepository) Create(ctx context.Context, like *entities.PostLike) error {
	return getDB(ctx, r.db).Create(like).Error
}

// Delete removes the like row and returns the number of affected rows so the
// caller only decrements the post counter when a row was actually deleted.
// Two concurrent unlikes both pass the exist check, but only one DELETE can
// remove the row; the other reports zero and must not decrement again.
func (r *PostLikeRepository) Delete(ctx context.Context, postID, userID uuid.UUID) (int64, error) {
	res := getDB(ctx, r.db).Where("post_id = ? AND user_id = ?", postID, userID).Delete(&entities.PostLike{})
	return res.RowsAffected, res.Error
}

func (r *PostLikeRepository) Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	var count int64
	err := getDB(ctx, r.db).Model(&entities.PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostLikeRepository) GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool, len(postIDs))
	if userID == uuid.Nil || len(postIDs) == 0 {
		return result, nil
	}
	var liked []uuid.UUID
	if err := getDB(ctx, r.db).Model(&entities.PostLike{}).
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Pluck("post_id", &liked).Error; err != nil {
		return nil, err
	}
	for _, id := range liked {
		result[id] = true
	}
	return result, nil
}

func (r *PostLikeRepository) GetByPostIDPaginated(ctx context.Context, postID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.PostLike, int64, error) {
	db := getDB(ctx, r.db).Model(&entities.PostLike{}).Where("post_id = ?", postID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	db = shared_persist.ApplyPagination(db, pagination)

	var likes []*entities.PostLike
	if err := db.Order("created_at DESC").Find(&likes).Error; err != nil {
		return nil, 0, err
	}
	return likes, total, nil
}

package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
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

func (r *PostItemRepository) GetTransferItemsBySellerID(ctx context.Context, sellerUserID uuid.UUID) ([]*entities.PostItem, error) {
	var items []*entities.PostItem
	err := r.GetQueryWithPreload(ctx).
		Preload("Post").
		Where("post_id IN (?)",
			r.GetDB(ctx).
				Model(&entities.Post{}).
				Select("id").
				Where("user_id = ?", sellerUserID),
		).
		Where("(transfer_state IN ? OR status = ?)", []transferstate.TransferState{
			transferstate.Pending,
			transferstate.Accepted,
			transferstate.Declined,
		}, postitemstatus.Sold).
		Order("post_id ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *PostItemRepository) GetByItemID(ctx context.Context, itemID uuid.UUID) ([]*entities.PostItem, error) {
	var items []*entities.PostItem
	err := r.GetQueryWithPreload(ctx).Preload("Post").Where("item_id = ?", itemID).Find(&items).Error
	return items, err
}

func (r *PostItemRepository) GetSiblingItems(ctx context.Context, itemID uuid.UUID, excludePostItemID uuid.UUID) ([]*entities.PostItem, error) {
	var items []*entities.PostItem
	err := r.GetQueryWithPreload(ctx).
		Preload("Post").
		Where("item_id = ? AND id <> ?", itemID, excludePostItemID).
		Find(&items).Error
	return items, err
}

func (r *PostItemRepository) HasActiveTransfer(ctx context.Context, itemID uuid.UUID, excludePostItemID *uuid.UUID) (bool, error) {
	query := r.GetDB(ctx).
		Model(&entities.PostItem{}).
		Where("item_id = ?", itemID).
		Where("(status = ? OR transfer_state = ?)", postitemstatus.Sold, transferstate.Pending)

	if excludePostItemID != nil {
		query = query.Where("id <> ?", *excludePostItemID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostItemRepository) DeleteByPostAndIDs(ctx context.Context, postID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return r.GetDB(ctx).Where("post_id = ? AND id IN ?", postID, ids).Delete(&entities.PostItem{}).Error
}

func (r *PostItemRepository) SumVisiblePriceByPostID(ctx context.Context, postID uuid.UUID) (float64, error) {
	var total float64
	if err := r.GetDB(ctx).
		Model(&entities.PostItem{}).
		Select("COALESCE(SUM(price), 0)").
		Where("post_id = ? AND status <> ?", postID, postitemstatus.Hidden).
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *PostItemRepository) GetPostItemsForAdmin(ctx context.Context, filter repositories.AdminPostItemFilter) (*repositories.AdminPostItemListResult, error) {
	db := r.GetDB(ctx).Model(&entities.PostItem{}).
		Preload("Post").
		Preload("WardrobeItem").
		Preload("WardrobeItem.Category")

	if filter.Status != nil {
		db = db.Where("post_items.status = ?", *filter.Status)
	}

	if filter.TransferState != nil {
		db = db.Where("post_items.transfer_state = ?", *filter.TransferState)
	}

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if offset < 0 {
		offset = 0
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	var items []*entities.PostItem
	err := db.Order("post_items.created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	if err != nil {
		return nil, err
	}

	return &repositories.AdminPostItemListResult{
		PostItems:  items,
		TotalCount: totalCount,
	}, nil
}


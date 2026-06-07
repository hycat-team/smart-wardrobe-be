package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/requeststatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransferRequestRepository struct {
	shared_persist.GenericRepository[entities.TransferRequest, uuid.UUID]
}

func NewTransferRequestRepository(db *gorm.DB) repositories.ITransferRequestRepository {
	return &TransferRequestRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.TransferRequest, uuid.UUID](db, []string{"Buyer", "PostItem", "PostItem.WardrobeItem"}),
	}
}

func (r *TransferRequestRepository) GetByPostItemID(ctx context.Context, postItemID uuid.UUID) ([]*entities.TransferRequest, error) {
	var items []*entities.TransferRequest
	err := r.GetQueryWithPreload(ctx).Where("post_item_id = ?", postItemID).Find(&items).Error
	return items, err
}

func (r *TransferRequestRepository) GetPendingByBuyerAndItems(ctx context.Context, buyerID uuid.UUID, postItemIDs []uuid.UUID) ([]*entities.TransferRequest, error) {
	if len(postItemIDs) == 0 {
		return nil, nil
	}
	var items []*entities.TransferRequest
	err := r.GetQueryWithPreload(ctx).
		Where("buyer_id = ? AND post_item_id IN ? AND status = ?", buyerID, postItemIDs, requeststatus.Pending).
		Find(&items).Error
	return items, err
}

func (r *TransferRequestRepository) GetByBuyerAndPostItem(ctx context.Context, buyerID uuid.UUID, postItemID uuid.UUID) (*entities.TransferRequest, error) {
	var item entities.TransferRequest
	err := r.GetQueryWithPreload(ctx).
		Where("buyer_id = ? AND post_item_id = ?", buyerID, postItemID).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *TransferRequestRepository) GetByBuyerAndPostItems(ctx context.Context, buyerID uuid.UUID, postItemIDs []uuid.UUID) ([]*entities.TransferRequest, error) {
	if len(postItemIDs) == 0 {
		return nil, nil
	}

	var items []*entities.TransferRequest
	err := r.GetQueryWithPreload(ctx).
		Where("buyer_id = ? AND post_item_id IN ?", buyerID, postItemIDs).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

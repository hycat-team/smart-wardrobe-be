package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type IItemTransferUseCase interface {
	MarkPostItemSold(ctx context.Context, userID uuid.UUID, postItemID uuid.UUID, buyerUserID uuid.UUID) error
	GetPendingTransfers(ctx context.Context, buyerUserID uuid.UUID) ([]*dto.PendingTransferRes, error)
	GetSellerTransferPosts(ctx context.Context, sellerUserID uuid.UUID) ([]*dto.SellerTransferPostRes, error)
	AcceptTransfer(ctx context.Context, buyerUserID uuid.UUID, postItemID uuid.UUID) (*wardrobe_dto.WardrobeItemRes, error)
	DeclineTransfer(ctx context.Context, buyerUserID uuid.UUID, postItemID uuid.UUID) error
}

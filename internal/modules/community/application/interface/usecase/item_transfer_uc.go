package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type IItemTransferUseCase interface {
	MarkPostItemsSold(ctx context.Context, sellerUserID uuid.UUID, postItemIDs []uuid.UUID, buyerID uuid.UUID) error
	GetPendingTransfers(ctx context.Context, buyerUserID uuid.UUID) ([]*dto.PendingTransferRes, error)
	GetSellerTransferPosts(ctx context.Context, sellerUserID uuid.UUID) ([]*dto.SellerTransferPostRes, error)
	AcceptTransfers(ctx context.Context, buyerUserID uuid.UUID, postItemIDs []uuid.UUID) ([]*wardrobe_dto.WardrobeItemRes, error)
	DeclineTransfers(ctx context.Context, buyerUserID uuid.UUID, postItemIDs []uuid.UUID) error
	CreateTransferRequests(ctx context.Context, buyerUserID uuid.UUID, postItemIDs []uuid.UUID) error
	GetTransferRequestsForSeller(ctx context.Context, sellerUserID uuid.UUID, postItemID uuid.UUID) ([]*dto.TransferRequestRes, error)
}

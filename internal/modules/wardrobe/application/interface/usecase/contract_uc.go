package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type IWardrobeContractUseCase interface {
	CopyItemToUser(ctx context.Context, sourceItemID uuid.UUID, targetUserID uuid.UUID) (*dto.WardrobeItemRes, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error
	VerifyItemsForPost(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error
	GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error)
	SetFashionItemID(ctx context.Context, itemID uuid.UUID, fashionItemID uuid.UUID) error
}

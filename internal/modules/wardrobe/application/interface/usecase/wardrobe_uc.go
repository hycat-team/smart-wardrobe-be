package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type IWardrobeUseCase interface {
	GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error)
	CreateWardrobeItem(ctx context.Context, userID uuid.UUID, input dto.CreateWardrobeItemReq) (*dto.WardrobeItemRes, error)
}

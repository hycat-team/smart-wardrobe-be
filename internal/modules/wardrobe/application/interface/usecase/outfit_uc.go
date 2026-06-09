package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type IOutfitUseCase interface {
	GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error)
	SaveOutfit(ctx context.Context, userID uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error)
	UpdateOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error)
	GetOutfits(ctx context.Context, userID uuid.UUID, query dto.GetOutfitsQueryReq) (*shared_dto.PaginationResult[*dto.OutfitRes], error)
	GetOutfitByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.OutfitRes, error)
	DeleteOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
}

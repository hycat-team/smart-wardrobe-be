package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// IOutfitContractUseCase is the pure in-service contract use case for outfits.
// It only accesses the wardrobe module's own repositories.
type IOutfitContractUseCase interface {
	GetOutfitContractByID(ctx context.Context, id uuid.UUID) (*dto.OutfitContractDTO, error)
	GetOutfitContractsByIDs(ctx context.Context, ids []uuid.UUID) ([]*dto.OutfitContractDTO, error)
}

package contract

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"

	"github.com/google/uuid"
)

// IOutfitContract exposes outfit lookups to other in-service modules (e.g.
// community) using pure DTOs only.
type IOutfitContract interface {
	GetOutfitByID(ctx context.Context, id uuid.UUID) (*dto.OutfitContractDTO, error)
	GetOutfitsByIDs(ctx context.Context, ids []uuid.UUID) ([]*dto.OutfitContractDTO, error)
}

type OutfitContract struct {
	outfitContractUC uc_interfaces.IOutfitContractUseCase
}

func NewOutfitContract(outfitContractUC uc_interfaces.IOutfitContractUseCase) IOutfitContract {
	return &OutfitContract{outfitContractUC: outfitContractUC}
}

func (c *OutfitContract) GetOutfitByID(ctx context.Context, id uuid.UUID) (*dto.OutfitContractDTO, error) {
	return c.outfitContractUC.GetOutfitContractByID(ctx, id)
}

func (c *OutfitContract) GetOutfitsByIDs(ctx context.Context, ids []uuid.UUID) ([]*dto.OutfitContractDTO, error) {
	return c.outfitContractUC.GetOutfitContractsByIDs(ctx, ids)
}

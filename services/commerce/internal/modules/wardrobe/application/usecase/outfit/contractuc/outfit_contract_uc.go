package contractuc

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

type OutfitContractUseCase struct {
	outfitRepo repositories.IOutfitRepository
}

func NewOutfitContractUseCase(outfitRepo repositories.IOutfitRepository) uc_interfaces.IOutfitContractUseCase {
	return &OutfitContractUseCase{outfitRepo: outfitRepo}
}

func (uc *OutfitContractUseCase) GetOutfitContractByID(ctx context.Context, id uuid.UUID) (*dto.OutfitContractDTO, error) {
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapper.MapToOutfitContractDTO(outfit), nil
}

func (uc *OutfitContractUseCase) GetOutfitContractsByIDs(ctx context.Context, ids []uuid.UUID) ([]*dto.OutfitContractDTO, error) {
	if len(ids) == 0 {
		return []*dto.OutfitContractDTO{}, nil
	}
	outfits, err := uc.outfitRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.OutfitContractDTO, 0, len(outfits))
	for _, outfit := range outfits {
		result = append(result, mapper.MapToOutfitContractDTO(outfit))
	}
	return result, nil
}

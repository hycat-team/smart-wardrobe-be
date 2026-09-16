package loyalty

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"

	"github.com/google/uuid"
)

func (uc *BrandLoyaltyUseCase) GetLoyaltyProgramForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) (*dto.LoyaltyProgramRes, error) {
	program, err := uc.programRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if program == nil {
		return nil, branderrors.ErrActiveLoyaltyProgramRequired()
	}
	return mapper.MapLoyaltyProgram(program), nil
}

func (uc *BrandLoyaltyUseCase) UpsertLoyaltyProgram(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.UpsertLoyaltyProgramReq) (*dto.LoyaltyProgramRes, error) {
	program, err := uc.programRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	if program == nil {
		program = &entities.LoyaltyProgram{
			BrandID:         brandID,
			Name:            input.Name,
			AmountPerPoint:  input.AmountPerPoint,
			PointExpiryDays: input.PointExpiryDays,
			RoundingMode:    input.RoundingMode,
			IsActive:        true,
		}
		if input.IsActive != nil {
			program.IsActive = *input.IsActive
		}
		if err := uc.programRepo.Create(ctx, program); err != nil {
			return nil, err
		}
	} else {
		program.Name = input.Name
		program.AmountPerPoint = input.AmountPerPoint
		program.PointExpiryDays = input.PointExpiryDays
		program.RoundingMode = input.RoundingMode
		if input.IsActive != nil {
			program.IsActive = *input.IsActive
		}
		if err := uc.programRepo.Update(ctx, program); err != nil {
			return nil, err
		}
	}

	return mapper.MapLoyaltyProgram(program), nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTiersForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyTierRes, error) {
	tiers, err := uc.tierRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTiers(tiers), nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTierDetailForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, tierID uuid.UUID) (*dto.LoyaltyTierDetailRes, error) {
	tier, err := uc.tierRepo.GetByID(ctx, tierID)
	if err != nil {
		return nil, err
	}
	if tier == nil || tier.BrandID != brandID {
		return nil, branderrors.ErrTierNotFound()
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandIDAndTierID(ctx, brandID, tierID)
	if err != nil {
		return nil, err
	}

	return &dto.LoyaltyTierDetailRes{
		LoyaltyTierRes: *mapper.MapLoyaltyTier(tier),
		Benefits:       mapper.MapBrandBenefits(benefits),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTierDetailForUser(ctx context.Context, brandID uuid.UUID, tierID uuid.UUID) (*dto.LoyaltyTierDetailRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	tier, err := uc.tierRepo.GetByID(ctx, tierID)
	if err != nil {
		return nil, err
	}
	if tier == nil || tier.BrandID != brandID {
		return nil, branderrors.ErrTierNotFound()
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandIDAndTierID(ctx, brandID, tierID)
	if err != nil {
		return nil, err
	}

	return &dto.LoyaltyTierDetailRes{
		LoyaltyTierRes: *mapper.MapLoyaltyTier(tier),
		Benefits:       mapper.MapBrandBenefits(benefits),
	}, nil
}

func (uc *BrandLoyaltyUseCase) CreateLoyaltyTier(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateLoyaltyTierReq) (*dto.LoyaltyTierRes, error) {
	existing, err := uc.tierRepo.GetByBrandAndName(ctx, brandID, input.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrTierNameExists(input.Name)
	}

	existing, err = uc.tierRepo.GetByBrandAndRank(ctx, brandID, input.Rank)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrTierRankExists(input.Rank)
	}

	tier := &entities.LoyaltyTier{
		BrandID:       brandID,
		Name:          input.Name,
		Rank:          input.Rank,
		MinTotalSpend: input.MinTotalSpend,
		Description:   input.Description,
	}
	if err := uc.tierRepo.Create(ctx, tier); err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTier(tier), nil
}

func (uc *BrandLoyaltyUseCase) UpdateLoyaltyTier(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, tierID uuid.UUID, input dto.UpdateLoyaltyTierReq) (*dto.LoyaltyTierRes, error) {
	tier, err := uc.tierRepo.GetByID(ctx, tierID)
	if err != nil {
		return nil, err
	}
	if tier == nil || tier.BrandID != brandID {
		return nil, branderrors.ErrTierNotFound()
	}

	if input.Name != nil {
		existing, err := uc.tierRepo.GetByBrandAndName(ctx, brandID, *input.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != tierID {
			return nil, branderrors.ErrTierNameExists(*input.Name)
		}
		tier.Name = *input.Name
	}

	if input.Rank != nil {
		existing, err := uc.tierRepo.GetByBrandAndRank(ctx, brandID, *input.Rank)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != tierID {
			return nil, branderrors.ErrTierRankExists(*input.Rank)
		}
		tier.Rank = *input.Rank
	}

	if input.MinTotalSpend != nil {
		tier.MinTotalSpend = *input.MinTotalSpend
	}

	if input.Description != nil {
		tier.Description = input.Description
	}

	if err := uc.tierRepo.Update(ctx, tier); err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTier(tier), nil
}

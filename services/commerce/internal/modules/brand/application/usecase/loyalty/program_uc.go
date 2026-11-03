package loyalty

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitunlocktype"
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

	res := mapper.MapLoyaltyTiers(tiers)
	if len(res) == 0 {
		return res, nil
	}

	tierIDs := make([]uuid.UUID, len(res))
	for i, tier := range res {
		tierIDs[i] = tier.ID
	}
	counts, err := uc.benefitRepo.CountActiveByTierIDs(ctx, brandID, tierIDs)
	if err != nil {
		return nil, err
	}
	for _, tier := range res {
		tier.BenefitsCount = int(counts[tier.ID])
	}
	return res, nil
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

	benefits, err := uc.activeTierPrivilegeBenefits(ctx, brandID)
	if err != nil {
		return nil, err
	}

	return &dto.LoyaltyTierDetailRes{
		LoyaltyTierRes: *mapper.MapLoyaltyTier(tier),
		Benefits:       mapper.MapBrandBenefits(cumulativeTierBenefits(benefits, tier.Rank)),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTiersForUser(ctx context.Context, brandID uuid.UUID) ([]*dto.LoyaltyTierDetailRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	tiers, err := uc.tierRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	benefits, err := uc.activeTierPrivilegeBenefits(ctx, brandID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.LoyaltyTierDetailRes, 0, len(tiers))
	for _, tier := range tiers {
		result = append(result, &dto.LoyaltyTierDetailRes{
			LoyaltyTierRes: *mapper.MapLoyaltyTier(tier),
			Benefits:       mapper.MapBrandBenefits(cumulativeTierBenefits(benefits, tier.Rank)),
		})
	}
	return result, nil
}

// activeTierPrivilegeBenefits returns the active benefits that are unlocked by a
// membership tier (including their required tier rank, preloaded by the repo).
func (uc *BrandLoyaltyUseCase) activeTierPrivilegeBenefits(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	benefits, err := uc.benefitRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	tierBenefits := make([]*entities.BrandBenefit, 0, len(benefits))
	for _, benefit := range benefits {
		if benefit.UnlockType != benefitunlocktype.TierPrivilege || benefit.RequiredTier == nil {
			continue
		}
		tierBenefits = append(tierBenefits, benefit)
	}
	return tierBenefits, nil
}

// cumulativeTierBenefits keeps every tier benefit whose required rank is at or
// below the given rank, matching the rank-based access rule.
func cumulativeTierBenefits(benefits []*entities.BrandBenefit, maxRank int) []*entities.BrandBenefit {
	result := make([]*entities.BrandBenefit, 0, len(benefits))
	for _, benefit := range benefits {
		if benefit.RequiredTier != nil && benefit.RequiredTier.Rank <= maxRank {
			result = append(result, benefit)
		}
	}
	return result
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

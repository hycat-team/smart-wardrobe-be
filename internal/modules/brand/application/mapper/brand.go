package mapper

import (
	"encoding/json"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapBrand(brand *entities.Brand) *dto.BrandRes {
	if brand == nil {
		return nil
	}
	return &dto.BrandRes{
		ID:               brand.ID,
		Slug:             brand.Slug,
		Name:             brand.Name,
		Description:      brand.Description,
		LogoURL:          brand.LogoURL,
		Status:           brand.Status,
		CreatedByUserID:  brand.CreatedByUserID,
		ApprovedByUserID: brand.ApprovedByUserID,
		ApprovedAt:       brand.ApprovedAt,
		CreatedAt:        brand.CreatedAt,
		UpdatedAt:        brand.UpdatedAt,
	}
}

func MapBrands(brands []*entities.Brand) []*dto.BrandRes {
	res := make([]*dto.BrandRes, len(brands))
	for idx, brand := range brands {
		res[idx] = MapBrand(brand)
	}
	return res
}

func MapBrandMember(member *entities.BrandMember) *dto.BrandMemberRes {
	if member == nil {
		return nil
	}
	return &dto.BrandMemberRes{
		ID:        member.ID,
		BrandID:   member.BrandID,
		UserID:    member.UserID,
		Role:      member.Role,
		Status:    member.Status,
		CreatedAt: member.CreatedAt,
		UpdatedAt: member.UpdatedAt,
	}
}

func MapBrandMembers(members []*entities.BrandMember) []*dto.BrandMemberRes {
	res := make([]*dto.BrandMemberRes, len(members))
	for idx, member := range members {
		res[idx] = MapBrandMember(member)
	}
	return res
}

func MapBrandCustomer(customer *entities.BrandCustomer) *dto.BrandCustomerRes {
	if customer == nil {
		return nil
	}
	return &dto.BrandCustomerRes{
		ID:                   customer.ID,
		BrandID:              customer.BrandID,
		UserID:               customer.UserID,
		CustomerName:         customer.CustomerName,
		PhoneE164:            customer.PhoneE164,
		ExternalCustomerCode: customer.ExternalCustomerCode,
		JoinedSource:         customer.JoinedSource,
		Status:               customer.Status,
		JoinedAt:             customer.JoinedAt,
		ClaimedAt:            customer.ClaimedAt,
		CreatedByMemberID:    customer.CreatedByMemberID,
		CreatedAt:            customer.CreatedAt,
		UpdatedAt:            customer.UpdatedAt,
	}
}

func MapBrandCustomers(customers []*entities.BrandCustomer) []*dto.BrandCustomerRes {
	res := make([]*dto.BrandCustomerRes, len(customers))
	for idx, customer := range customers {
		res[idx] = MapBrandCustomer(customer)
	}
	return res
}

func MapBrandBenefit(benefit *entities.BrandBenefit) *dto.BrandBenefitRes {
	if benefit == nil {
		return nil
	}
	var rawConfig interface{}
	if len(benefit.FeatureConfig) > 0 {
		_ = json.Unmarshal(benefit.FeatureConfig, &rawConfig)
	}
	var featCode *string
	if benefit.FeatureCode != nil {
		str := string(*benefit.FeatureCode)
		featCode = &str
	}
	return &dto.BrandBenefitRes{
		ID:             benefit.ID,
		BrandID:        benefit.BrandID,
		Name:           benefit.Name,
		Description:    benefit.Description,
		BenefitType:    string(benefit.BenefitType),
		UnlockType:     string(benefit.UnlockType),
		RequiredPoints: benefit.RequiredPoints,
		RequiredTierID: benefit.RequiredTierID,
		FeatureCode:    featCode,
		FeatureConfig:  rawConfig,
		Status:         string(benefit.Status),
		CreatedAt:      benefit.CreatedAt,
		UpdatedAt:      benefit.UpdatedAt,
	}
}

func MapBrandBenefits(benefits []*entities.BrandBenefit) []*dto.BrandBenefitRes {
	res := make([]*dto.BrandBenefitRes, len(benefits))
	for idx, benefit := range benefits {
		res[idx] = MapBrandBenefit(benefit)
	}
	return res
}

func MapBenefitRedemption(red *entities.BenefitRedemption) *dto.BenefitRedemptionRes {
	if red == nil {
		return nil
	}
	return &dto.BenefitRedemptionRes{
		ID:              red.ID,
		BenefitID:       red.BenefitID,
		BrandID:         red.BrandID,
		BrandCustomerID: red.BrandCustomerID,
		UserID:          red.UserID,
		PointsSpent:     red.PointsSpent,
		Status:          string(red.Status),
		RedeemedAt:      red.RedeemedAt,
		UsedAt:          red.UsedAt,
		ExpiresAt:       red.ExpiresAt,
		CreatedAt:       red.CreatedAt,
		UpdatedAt:       red.UpdatedAt,
	}
}

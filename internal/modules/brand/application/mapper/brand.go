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
		LogoPublicID:     brand.LogoPublicID,
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

func MapPortalBrand(member *entities.BrandMember) *dto.PortalBrandRes {
	if member == nil || member.Brand == nil {
		return nil
	}
	brand := MapBrand(member.Brand)
	if brand == nil {
		return nil
	}
	return &dto.PortalBrandRes{
		BrandRes:     *brand,
		MemberID:     member.ID,
		MemberRole:   member.Role,
		MemberStatus: member.Status,
	}
}

func MapPortalBrands(members []*entities.BrandMember) []*dto.PortalBrandRes {
	res := make([]*dto.PortalBrandRes, 0, len(members))
	for _, member := range members {
		if mapped := MapPortalBrand(member); mapped != nil {
			res = append(res, mapped)
		}
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

func MapBrandConversation(conv *entities.BrandConversation, customerName *string, userDisplayName *string) *dto.BrandConversationRes {
	if conv == nil {
		return nil
	}
	return &dto.BrandConversationRes{
		ID:              conv.ID,
		BrandID:         conv.BrandID,
		UserID:          conv.UserID,
		CustomerName:    customerName,
		UserDisplayName: userDisplayName,
		Status:          string(conv.Status),
		LastMessageAt:   conv.LastMessageAt,
		UserLastReadAt:  conv.UserLastReadAt,
		StaffLastReadAt: conv.StaffLastReadAt,
		CreatedAt:       conv.CreatedAt,
		UpdatedAt:       conv.UpdatedAt,
	}
}

func MapBrandConversationMessage(msg *entities.BrandConversationMessage) *dto.BrandConversationMessageRes {
	if msg == nil {
		return nil
	}
	return &dto.BrandConversationMessageRes{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderRole:     string(msg.SenderRole),
		SenderUserID:   msg.SenderUserID,
		Message:        msg.Message,
		CreatedAt:      msg.CreatedAt,
	}
}

func MapBrandConversationMessages(messages []*entities.BrandConversationMessage) []*dto.BrandConversationMessageRes {
	res := make([]*dto.BrandConversationMessageRes, len(messages))
	for idx, msg := range messages {
		res[idx] = MapBrandConversationMessage(msg)
	}
	return res
}

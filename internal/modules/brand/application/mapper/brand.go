package mapper

import (
	"encoding/json"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

func MapBrand(brand *entities.Brand, totalCustomer int) *dto.BrandRes {
	if brand == nil {
		return nil
	}
	res := &dto.BrandRes{
		ID:               brand.ID,
		Slug:             brand.Slug,
		Name:             brand.Name,
		Description:      brand.Description,
		LogoURL:          brand.LogoURL,
		BackgroundURL:    brand.BackgroundURL,
		Status:           brand.Status,
		CreatedByUserID:  brand.CreatedByUserID,
		ApprovedByUserID: brand.ApprovedByUserID,
		ApprovedAt:       brand.ApprovedAt,
		CreatedAt:        brand.CreatedAt,
		UpdatedAt:        brand.UpdatedAt,
	}
	if totalCustomer >= 0 {
		res.TotalCustomer = &totalCustomer
	}
	return res
}

func MapBrands(brands []*entities.Brand, totalCustomerByBrandID map[uuid.UUID]int) []*dto.BrandRes {
	res := make([]*dto.BrandRes, len(brands))
	for idx, brand := range brands {
		count := -1
		if totalCustomerByBrandID != nil {
			if c, ok := totalCustomerByBrandID[brand.ID]; ok {
				count = c
			}
		}
		res[idx] = MapBrand(brand, count)
	}
	return res
}

func MapPortalBrand(member *entities.BrandMember, totalCustomer int) *dto.PortalBrandRes {
	if member == nil || member.Brand == nil {
		return nil
	}
	brand := MapBrand(member.Brand, totalCustomer)
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

func MapPortalBrands(members []*entities.BrandMember, totalCustomerByBrandID map[uuid.UUID]int) []*dto.PortalBrandRes {
	res := make([]*dto.PortalBrandRes, 0, len(members))
	for _, member := range members {
		count := -1
		if totalCustomerByBrandID != nil {
			if c, ok := totalCustomerByBrandID[member.BrandID]; ok {
				count = c
			}
		}
		if mapped := MapPortalBrand(member, count); mapped != nil {
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

func MapBrandCustomer(customer *entities.BrandCustomer, user *identity_dto.UserRes) *dto.BrandCustomerRes {
	if customer == nil {
		return nil
	}

	var customerUser *dto.BrandCustomerUserRes
	if user != nil {
		customerUser = &dto.BrandCustomerUserRes{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Gender:    user.Gender,
			AvatarUrl: user.AvatarUrl,
		}
	}

	var loyaltyAccount *dto.CustomerLoyaltyAccountRes
	if customer.LoyaltyAccount != nil {
		var currentTier *dto.LoyaltyTierBriefRes
		if customer.LoyaltyAccount.CurrentTier != nil {
			currentTier = &dto.LoyaltyTierBriefRes{
				ID:   customer.LoyaltyAccount.CurrentTier.ID,
				Name: customer.LoyaltyAccount.CurrentTier.Name,
			}
		}
		loyaltyAccount = &dto.CustomerLoyaltyAccountRes{
			ID:             customer.LoyaltyAccount.ID,
			CurrentPoints:  customer.LoyaltyAccount.CurrentPoints,
			LifetimePoints: customer.LoyaltyAccount.LifetimePoints,
			TotalSpend:     customer.LoyaltyAccount.TotalSpend,
			CurrentTier:    currentTier,
		}
	}

	return &dto.BrandCustomerRes{
		ID:                   customer.ID,
		BrandID:              customer.BrandID,
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
		User:                 customerUser,
		LoyaltyAccount:       loyaltyAccount,
	}
}

func MapBrandCustomers(customers []*entities.BrandCustomer) []*dto.BrandCustomerRes {
	res := make([]*dto.BrandCustomerRes, len(customers))
	for idx, customer := range customers {
		res[idx] = MapBrandCustomer(customer, nil)
	}
	return res
}

func MapBrandCustomersWithUsers(customers []*entities.BrandCustomer, userMap map[uuid.UUID]*identity_dto.UserRes) []*dto.BrandCustomerRes {
	res := make([]*dto.BrandCustomerRes, 0, len(customers))
	for _, customer := range customers {
		var user *identity_dto.UserRes
		if customer.UserID != nil {
			user = userMap[*customer.UserID]
		}
		res = append(res, MapBrandCustomer(customer, user))
	}
	return res
}

func MapBrandBenefit(benefit *entities.BrandBenefit) *dto.BrandBenefitRes {
	if benefit == nil {
		return nil
	}
	var rawConfig any
	if len(benefit.FeatureConfig) > 0 {
		_ = json.Unmarshal(benefit.FeatureConfig, &rawConfig)
	}
	var featCode *string
	if benefit.FeatureCode != nil {
		str := string(*benefit.FeatureCode)
		featCode = &str
	}
	var requiredTierName *string
	if benefit.RequiredTier != nil {
		requiredTierName = &benefit.RequiredTier.Name
	}
	return &dto.BrandBenefitRes{
		ID:               benefit.ID,
		BrandID:          benefit.BrandID,
		Name:             benefit.Name,
		Description:      benefit.Description,
		BenefitType:      string(benefit.BenefitType),
		UnlockType:       string(benefit.UnlockType),
		RequiredPoints:   benefit.RequiredPoints,
		RequiredTierID:   benefit.RequiredTierID,
		RequiredTierName: requiredTierName,
		FeatureCode:      featCode,
		FeatureConfig:    rawConfig,
		Status:           string(benefit.Status),
		CreatedAt:        benefit.CreatedAt,
		UpdatedAt:        benefit.UpdatedAt,
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

func MapBrandConversation(conv *entities.BrandConversation, customerId uuid.UUID, customerName *string, userDisplayName *string) *dto.BrandConversationRes {
	if conv == nil {
		return nil
	}
	return &dto.BrandConversationRes{
		ID:              conv.ID,
		BrandID:         conv.BrandID,
		UserID:          conv.UserID,
		CustomerId:      customerId,
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

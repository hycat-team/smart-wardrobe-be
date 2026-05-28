package mapper

import (
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// MapToUserRes maps user identity and subscription plan metrics into a coherent response DTO
func MapToUserRes(user *entities.User, sub *subscription_contract.UserSubscriptionOverviewDTO) *dto.UserRes {
	if user == nil {
		return nil
	}

	var firstNameStr, lastNameStr, addressStr string
	var genderVal gender.Gender

	if user.FirstName != nil {
		firstNameStr = *user.FirstName
	}
	if user.LastName != nil {
		lastNameStr = *user.LastName
	}
	if user.Address != nil {
		addressStr = *user.Address
	}
	if user.Gender != nil {
		genderVal = *user.Gender
	}

	res := &dto.UserRes{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		RoleSlug:  user.RoleSlug,
		FirstName: firstNameStr,
		LastName:  lastNameStr,
		Address:   addressStr,
		Gender:    genderVal,
		Status:    int(user.Status),
		CreatedAt: user.CreatedAt,
	}

	if sub != nil {
		res.Subscription = dto.UserSubscriptionRes{
			PlanID:             sub.PlanID,
			PlanName:           sub.PlanName,
			ExpiresAt:          sub.ExpiresAt,
			MaxWardrobeItems:   sub.MaxWardrobeItems,
			MaxOutfits:         sub.MaxOutfits,
			AiOutfitDailyQuota: sub.AiOutfitDailyQuota,
			AiChatDailyQuota:   sub.AiChatDailyQuota,
		}
	}

	if user.BodyProfile != nil {
		res.BodyProfile = &dto.UserBodyProfileRes{
			Height:             user.BodyProfile.Height,
			Weight:             user.BodyProfile.Weight,
			BodyType:           user.BodyProfile.BodyType,
			FitPreference:      user.BodyProfile.FitPreference,
			SkinTone:           user.BodyProfile.SkinTone,
			EstimatedBodyShape: user.BodyProfile.EstimatedBodyShape,
			RecommendedSize:    user.BodyProfile.RecommendedSize,
			StylingNotes:       user.BodyProfile.StylingNotes,
		}
	}

	return res
}

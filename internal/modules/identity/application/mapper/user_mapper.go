package mapper

import (
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/domain/entities"
	subscription_dto "smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"
)

// MapToUserBrief maps user identity into a minimal display response DTO.
// Lightweight: does not fetch subscription or body profile.
func MapToUserBrief(user *entities.User) *dto.UserBriefRes {
	if user == nil {
		return nil
	}

	var firstNameStr, lastNameStr string
	var genderVal gender.Gender

	if user.FirstName != nil {
		firstNameStr = *user.FirstName
	}
	if user.LastName != nil {
		lastNameStr = *user.LastName
	}
	if user.Gender != nil {
		genderVal = *user.Gender
	}

	return &dto.UserBriefRes{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: firstNameStr,
		LastName:  lastNameStr,
		Gender:    genderVal,
		AvatarUrl: user.AvatarUrl,
	}
}

// MapToUserBriefs maps a batch of users into brief response DTOs preserving input order.
func MapToUserBriefs(users []*entities.User) []*dto.UserBriefRes {
	if len(users) == 0 {
		return []*dto.UserBriefRes{}
	}
	res := make([]*dto.UserBriefRes, 0, len(users))
	for _, user := range users {
		if brief := MapToUserBrief(user); brief != nil {
			res = append(res, brief)
		}
	}
	return res
}

// MapToUserRes maps user identity and subscription plan metrics into a coherent response DTO
func MapToUserRes(user *entities.User, sub *subscription_dto.UserSubscriptionOverviewDTO) *dto.UserRes {
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

	var dobStr *string
	if user.DateOfBirth != nil {
		formatted := user.DateOfBirth.Format("2006-01-02")
		dobStr = &formatted
	}

	res := &dto.UserRes{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		RoleSlug:    user.RoleSlug,
		FirstName:   firstNameStr,
		LastName:    lastNameStr,
		DateOfBirth: dobStr,
		Address:     addressStr,
		Gender:      genderVal,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		AvatarUrl:   user.AvatarUrl,
	}

	if sub != nil {
		res.Subscription = dto.UserSubscriptionRes{
			PlanID:             sub.PlanID,
			PlanName:           sub.PlanName,
			PlanSlug:           sub.PlanSlug,
			ExpiresAt:          sub.ExpiresAt,
			MaxWardrobeItems:   sub.MaxWardrobeItems,
			MaxOutfits:         sub.MaxOutfits,
			AiOutfitDailyQuota: sub.AiOutfitDailyQuota,
			AiChatDailyQuota:   sub.AiChatDailyQuota,
		}
	}

	if user.BodyProfile != nil {
		var measurements *dto.UserBodyMeasurementsRes
		if user.BodyProfile.Measurements != nil {
			measurements = &dto.UserBodyMeasurementsRes{
				ChestCM: user.BodyProfile.Measurements.ChestCM,
				WaistCM: user.BodyProfile.Measurements.WaistCM,
				HipCM:   user.BodyProfile.Measurements.HipCM,
			}
		}

		var inferredByAI *dto.UserInferredBodyRes
		if user.BodyProfile.InferredByAI != nil {
			inferredByAI = &dto.UserInferredBodyRes{
				BodyShape:       user.BodyProfile.InferredByAI.BodyShape,
				ConfidenceScore: user.BodyProfile.InferredByAI.ConfidenceScore,
			}
		}

		effectiveProfile := user.GetEffectiveBodyProfile()
		res.BodyProfile = &dto.UserBodyProfileRes{
			HeightCM:       user.BodyProfile.HeightCM,
			WeightKG:       user.BodyProfile.WeightKG,
			BodyShape:      effectiveProfile.BodyShape,
			Measurements:   measurements,
			InferredByAI:   inferredByAI,
			VerifiedByUser: user.BodyProfile.VerifiedByUser,
			LastUpdatedAt:  user.BodyProfile.LastUpdatedAt,
		}
	}

	return res
}

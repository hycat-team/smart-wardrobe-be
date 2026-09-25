package mapper

import (
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
)

// mapToCommunityUser builds the canonical user projection. Gender is included
// only when known (non-zero).
func mapToCommunityUser(user *identity_dto.UserBriefContractDTO) *dto.CommunityUserRes {
	if user == nil {
		return nil
	}
	res := &dto.CommunityUserRes{
		UserID:    user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		AvatarURL: user.AvatarUrl,
	}
	if user.Gender != 0 {
		gender := int(user.Gender)
		res.Gender = &gender
	}
	return res
}

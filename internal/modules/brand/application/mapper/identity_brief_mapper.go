package mapper

import identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"

// MapIdentityUserBrief converts the identity cross-module contract payload into
// the API response shape embedded in brand responses (contract payload and API
// response evolve independently per the DTO Convention).
func MapIdentityUserBrief(in *identity_dto.UserBriefContractDTO) *identity_dto.UserBriefRes {
	if in == nil {
		return nil
	}
	return &identity_dto.UserBriefRes{
		ID:        in.ID,
		Username:  in.Username,
		Email:     in.Email,
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Gender:    in.Gender,
		Status:    in.Status,
		AvatarUrl: in.AvatarUrl,
	}
}

package mapper

import (
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// MapToUserRes converts a domain User entity to a UserRes DTO
func MapToUserRes(user *entities.User) *dto.UserRes {
	if user == nil {
		return nil
	}

	var firstNameVal, lastNameVal, addressVal string
	if user.FirstName != nil {
		firstNameVal = *user.FirstName
	}
	if user.LastName != nil {
		lastNameVal = *user.LastName
	}
	if user.Address != nil {
		addressVal = *user.Address
	}

	var genVal gender.Gender
	if user.Gender != nil {
		genVal = *user.Gender
	}

	return &dto.UserRes{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		RoleSlug:  user.RoleSlug,
		FirstName: firstNameVal,
		LastName:  lastNameVal,
		Address:   addressVal,
		Gender:    genVal,
		CreatedAt: user.CreatedAt,
	}
}

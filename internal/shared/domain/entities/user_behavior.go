package entities

import (
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"time"
)

func (u *User) UpdateProfile(firstName string, lastName *string, dateOfBirth time.Time, gen gender.Gender) {
	u.FirstName = &firstName
	u.LastName = lastName
	u.DateOfBirth = &dateOfBirth
	u.Gender = &gen
}

func (u *User) ChangeAddress(newAddress string) {
	u.Address = &newAddress
}

func (u *User) ChangePasswordHash(newPasswordHash string) {
	u.PasswordHash = newPasswordHash
}

func (u *User) UpdateAvatar(avatarUrl string, avatarPublicID string) {
	u.AvatarUrl = &avatarUrl
	u.AvatarPublicID = &avatarPublicID
}

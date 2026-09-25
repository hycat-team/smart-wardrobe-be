package entities

import (
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"
	"time"
)

func (u *User) UpdateProfile(firstName string, lastName *string, dateOfBirth time.Time, gen gender.Gender) {
	u.FirstName = &firstName
	u.LastName = lastName
	u.DateOfBirth = &dateOfBirth
	u.Gender = &gen
}

func (u *User) ChangeAddress(newAddress *string) {
	u.Address = newAddress
}

func (u *User) ChangePasswordHash(newPasswordHash string) {
	u.PasswordHash = &newPasswordHash
}

// HasPassword reports whether the account currently has a password hash set.
// Google-only accounts have no password until the user sets one.
func (u *User) HasPassword() bool {
	return u != nil && u.PasswordHash != nil && *u.PasswordHash != ""
}

// LinkGoogleIdentity attaches a Google identity to the account.
func (u *User) LinkGoogleIdentity(providerID string) {
	provider := authprovider.Google
	u.Provider = provider
	u.ProviderID = &providerID
	u.EmailVerified = true
}

func (u *User) UpdateAvatar(avatarUrl string, avatarPublicID string) {
	u.AvatarUrl = &avatarUrl
	u.AvatarPublicID = &avatarPublicID
}

func (u *User) UpdateBodyProfile(profile *BodyProfile) {
	u.BodyProfile = profile
}

func (u *User) UpdateStatus(status userstatus.UserStatus) {
	u.Status = status
}

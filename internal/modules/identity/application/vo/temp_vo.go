package vo

import "smart-wardrobe-be/internal/shared/domain/constants/shared/gender"

type TempUserCacheModel struct {
	Username     string        `json:"Username"`
	Email        string        `json:"Email"`
	PasswordHash string        `json:"PasswordHash"`
	FirstName    string        `json:"FirstName"`
	LastName     string        `json:"LastName"`
	Address      string        `json:"Address"`
	DateOfBirth  string        `json:"DateOfBirth"`
	Gender       gender.Gender `json:"Gender"`
}

type TempOtpData struct {
	UserId string `json:"UserId"`
}

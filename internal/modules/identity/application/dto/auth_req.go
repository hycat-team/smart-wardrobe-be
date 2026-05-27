package dto

import "smart-wardrobe-be/internal/shared/domain/constants/gender"

type RegisterReq struct {
	Username        string         `json:"username" binding:"required,username"`
	Email           string         `json:"email" binding:"required,email"`
	Password        string         `json:"password" binding:"required,min=6"`
	ConfirmPassword string         `json:"confirmPassword" binding:"required,eqfield=Password"`
	FirstName       string         `json:"firstName" binding:"required"`
	LastName        *string        `json:"lastName" binding:"omitempty"`
	Address         string         `json:"address" binding:"required"`
	DateOfBirth     string         `json:"dateOfBirth" binding:"required,datetime=2006-01-02"`
	Gender          *gender.Gender `json:"gender" binding:"omitempty,oneof=0 1 2 3"`
}

type ConfirmRegisterOtpReq struct {
	Email   string `json:"email" binding:"required,email"`
	OtpCode string `json:"otpCode" binding:"required,len=6,numeric"`
}

type LoginReq struct {
	LoginName string `json:"loginName" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type RefreshTokenReq struct {
	OldRefreshToken string `json:"oldRefreshToken" binding:"required"`
}

type LogoutReq struct {
	AccessToken  string `json:"accessToken" binding:"required"`
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type SendForgotPasswordOtpReq struct {
	Email string `json:"email" binding:"required,email"`
}

type ConfirmForgotPasswordOtpReq struct {
	Email   string `json:"email" binding:"required,email"`
	OtpCode string `json:"otpCode" binding:"required,len=6,numeric"`
}

package dto

import "smart-wardrobe-be/internal/shared/domain/constants/gender"

type RegisterReq struct {
	Username        string         `json:"username" binding:"required,username" label:"tên đăng nhập"`
	Email           string         `json:"email" binding:"required,email" label:"email"`
	Password        string         `json:"password" binding:"required,min=6" label:"mật khẩu"`
	ConfirmPassword string         `json:"confirmPassword" binding:"required,eqfield=Password" label:"xác nhận mật khẩu"`
	FirstName       string         `json:"firstName" binding:"required" label:"tên"`
	LastName        *string        `json:"lastName" binding:"omitempty" label:"họ"`
	Address         string         `json:"address" binding:"required" label:"địa chỉ"`
	DateOfBirth     string         `json:"dateOfBirth" binding:"required,datetime=2006-01-02" label:"ngày sinh"`
	Gender          *gender.Gender `json:"gender" binding:"omitempty,oneof=0 1 2 3" label:"giới tính"`
}

type ConfirmRegisterOtpReq struct {
	Email   string `json:"email" binding:"required,email" label:"email"`
	OtpCode string `json:"otpCode" binding:"required,len=6,numeric" label:"mã OTP"`
}

type LoginReq struct {
	LoginName string `json:"loginName" binding:"required" label:"tên đăng nhập hoặc email"`
	Password  string `json:"password" binding:"required" label:"mật khẩu"`
}

type RefreshTokenReq struct {
	OldRefreshToken string `json:"oldRefreshToken" binding:"required" label:"refresh token cũ"`
}

type LogoutReq struct {
	AccessToken  string `json:"accessToken" binding:"required" label:"access token"`
	RefreshToken string `json:"refreshToken" binding:"required" label:"refresh token"`
}

type SendForgotPasswordOtpReq struct {
	Email string `json:"email" binding:"required,email" label:"email"`
}

type ConfirmForgotPasswordOtpReq struct {
	Email   string `json:"email" binding:"required,email" label:"email"`
	OtpCode string `json:"otpCode" binding:"required,len=6,numeric" label:"mã OTP"`
}

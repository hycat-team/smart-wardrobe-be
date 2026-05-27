package dto

type ResetPasswordReq struct {
	NewPassword      string `json:"newPassword" binding:"required,min=6"`
	ConfirmPassword  string `json:"confirmPassword" binding:"required,eqfield=NewPassword"`
	LogoutAllDevices bool   `json:"logoutAllDevices"`
}

type ChangePasswordReq struct {
	OldPassword      string `json:"oldPassword" binding:"required"`
	NewPassword      string `json:"newPassword" binding:"required,min=6"`
	ConfirmPassword  string `json:"confirmPassword" binding:"required,eqfield=NewPassword"`
	LogoutAllDevices bool   `json:"logoutAllDevices"`
}

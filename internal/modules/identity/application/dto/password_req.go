package dto

type ResetPasswordReq struct {
	NewPassword      string `json:"newPassword" binding:"required,min=6" label:"mật khẩu mới"`
	ConfirmPassword  string `json:"confirmPassword" binding:"required,eqfield=NewPassword" label:"xác nhận mật khẩu mới"`
	LogoutAllDevices bool   `json:"logoutAllDevices"`
}

type ChangePasswordReq struct {
	OldPassword      string `json:"oldPassword" binding:"required" label:"mật khẩu cũ"`
	NewPassword      string `json:"newPassword" binding:"required,min=6" label:"mật khẩu mới"`
	ConfirmPassword  string `json:"confirmPassword" binding:"required,eqfield=NewPassword" label:"xác nhận mật khẩu mới"`
	LogoutAllDevices bool   `json:"logoutAllDevices"`
}

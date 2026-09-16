package dto

type ResetPasswordReq struct {
	ResetToken       *string `json:"resetToken,omitempty" label:"mã khôi phục mật khẩu"`
	NewPassword      string  `json:"newPassword" binding:"required,password_complexity" label:"mật khẩu mới"`
	ConfirmPassword  string  `json:"confirmPassword" binding:"required,eqfield=NewPassword" label:"xác nhận mật khẩu mới"`
	LogoutAllDevices bool    `json:"logoutAllDevices"`
}

type ConfirmForgotPasswordOtpRes struct {
	ResetToken string `json:"resetToken"`
}

type ChangePasswordReq struct {
	OldPassword      string `json:"oldPassword" binding:"required" label:"mật khẩu cũ"`
	NewPassword      string `json:"newPassword" binding:"required,password_complexity" label:"mật khẩu mới"`
	ConfirmPassword  string `json:"confirmPassword" binding:"required,eqfield=NewPassword" label:"xác nhận mật khẩu mới"`
	LogoutAllDevices bool   `json:"logoutAllDevices"`
}

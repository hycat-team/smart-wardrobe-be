package identityerrors

import (
	"fmt"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

// Thông điệp liên hệ chăm sóc khách hàng dùng chung
const contactSupportMessage = "Tài khoản đã bị khoá hoặc vô hiệu hoá. Vui lòng liên hệ CSKH."

// Lỗi động (cần định dạng tham số)
func ErrUsernameExists(username string) *apperror.Error {
	return apperror.NewConflict(fmt.Sprintf("Tài khoản '%s' đã tồn tại.", username))
}

func ErrEmailExists(email string) *apperror.Error {
	return apperror.NewConflict(fmt.Sprintf("Email '%s' đã tồn tại.", email))
}

// Lỗi tĩnh
func ErrOtpCooldown() *apperror.Error {
	return apperror.NewTooManyRequest("Vui lòng đợi 1 phút trước khi yêu cầu OTP mới.")
}

func ErrOtpGenericCooldown() *apperror.Error {
	return apperror.NewBadRequest("Vui lòng đợi một lát trước khi yêu cầu mã mới.")
}

func ErrOtpEmailSendFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi gửi email xác nhận OTP")
}

func ErrRecoveryEmailFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi gửi email khôi phục mật khẩu")
}

func ErrOtpVerificationFail() *apperror.Error {
	return apperror.NewInternalError("Lấy thông tin đăng ký thất bại")
}

func ErrOtpDataInvalid() *apperror.Error {
	return apperror.NewInternalError("Thông tin đăng ký không hợp lệ.")
}

func ErrOtpValidationInvalid() *apperror.Error {
	return apperror.NewBadRequest("Dữ liệu xác thực không hợp lệ.")
}

func ErrOtpNotFound() *apperror.Error {
	return apperror.NewBadRequest("Mã OTP không hợp lệ hoặc đã hết hạn.")
}

func ErrOtpTooManyAttempts() *apperror.Error {
	return apperror.NewTooManyRequest("Vui lòng thử lại sau. Bạn đã nhập sai mã OTP quá nhiều lần.")
}

func ErrInvalidDob() *apperror.Error {
	return apperror.NewBadRequest("Ngày sinh không hợp lệ. Vui lòng định dạng yyyy-mm-dd.")
}

func ErrJsonConvertFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi chuyển đổi thông tin người dùng")
}

func ErrTempConvertFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi chuyển đổi thông tin tạm thời")
}

func ErrResetTokenGenFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi cấp mã khôi phục mật khẩu")
}

func ErrSessionIssueFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi cấp phiên làm việc")
}

func ErrSessionCreateFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi cấp phiên làm việc.")
}

func ErrAccountCreationFailed() *apperror.Error {
	return apperror.NewInternalError("Lỗi khi khởi tạo tài khoản mới")
}

func ErrInvalidCredentials() *apperror.Error {
	return apperror.NewBadRequest("Sai tài khoản hoặc mật khẩu.")
}

func ErrInvalidOldPassword() *apperror.Error {
	return apperror.NewBadRequest("Mật khẩu cũ không chính xác.")
}

func ErrInvalidToken() *apperror.Error {
	return apperror.NewUnauthorized("Token không hợp lệ.")
}

func ErrTokenExpired() *apperror.Error {
	return apperror.NewUnauthorized("Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.")
}

func ErrInvalidSession() *apperror.Error {
	return apperror.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
}

func ErrInvalidSessionGeneric() *apperror.Error {
	return apperror.NewUnauthorized("Phiên làm việc không hợp lệ.")
}

func ErrTokenExpiredRecovery() *apperror.Error {
	return apperror.NewUnauthorized("Phiên làm việc đã hết hạn hoặc không hợp lệ. Vui lòng thực hiện lại yêu cầu.")
}

func ErrUserNotFound() *apperror.Error {
	return apperror.NewUnauthorized("Người dùng không tồn tại.")
}

func ErrUserNotFoundGeneric() *apperror.Error {
	return apperror.NewUnauthorized("Không tìm thấy người dùng.")
}

func ErrUserNotFoundDetailed() *apperror.Error {
	return apperror.NewUnauthorized("Không tìm thấy người dùng này.")
}

func ErrUserProfileNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thông tin người dùng.")
}

func ErrEmailNotRegistered() *apperror.Error {
	return apperror.NewNotFound("Email này chưa được đăng kí trong hệ thống.")
}

func ErrAdminAccountNotFound() *apperror.Error {
	return apperror.NewUnauthorized("Không tìm thấy thông tin tài khoản admin.")
}

func ErrAccountDisabled() *apperror.Error {
	return apperror.NewForbidden(contactSupportMessage)
}

func ErrAccountDisabledAuth() *apperror.Error {
	return apperror.NewUnauthorized(contactSupportMessage)
}

func ErrSelfStatusUpdate() *apperror.Error {
	return apperror.NewForbidden("Bạn không thể tự thay đổi trạng thái tài khoản admin của chính mình.")
}

func ErrUserStatusOnly() *apperror.Error {
	return apperror.NewForbidden("Chỉ có thể thay đổi trạng thái tài khoản user.")
}

func ErrCookieTokenMissing() *apperror.Error {
	return apperror.NewBadRequest("Thiếu token gia hạn phiên làm việc trong cookie.")
}

func ErrPasswordContainsUsername() *apperror.Error {
	return apperror.NewBadRequest("Mật khẩu không được trùng hoặc chứa tên đăng nhập.")
}

func ErrNewPasswordSameAsOld() *apperror.Error {
	return apperror.NewBadRequest("Mật khẩu mới không được trùng với mật khẩu cũ.")
}

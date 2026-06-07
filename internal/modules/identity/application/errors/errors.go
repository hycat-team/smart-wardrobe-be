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
var (
	// Lỗi OTP & Gửi Mail
	ErrOtpCooldown          = apperror.NewTooManyRequest("Vui lòng đợi 1 phút trước khi yêu cầu OTP mới.")
	ErrOtpGenericCooldown   = apperror.NewBadRequest("Vui lòng đợi một lát trước khi yêu cầu mã mới.")
	ErrOtpEmailSendFailed   = apperror.NewInternalError("Lỗi khi gửi email xác nhận OTP")
	ErrRecoveryEmailFailed  = apperror.NewInternalError("Lỗi khi gửi email khôi phục mật khẩu")
	ErrOtpVerificationFail  = apperror.NewInternalError("Lấy thông tin đăng ký thất bại")
	ErrOtpDataInvalid       = apperror.NewInternalError("Thông tin đăng ký không hợp lệ.")
	ErrOtpValidationInvalid = apperror.NewBadRequest("Dữ liệu xác thực không hợp lệ.")
	ErrOtpNotFound          = apperror.NewBadRequest("Mã OTP không hợp lệ hoặc đã hết hạn.")
	ErrOtpTooManyAttempts   = apperror.NewTooManyRequest("Vui lòng thử lại sau. Bạn đã nhập sai mã OTP quá nhiều lần.")

	// Lỗi Validation
	ErrInvalidDob = apperror.NewBadRequest("Ngày sinh không hợp lệ. Vui lòng định dạng yyyy-mm-dd.")

	// Lỗi Hệ thống & Mapping Nội bộ
	ErrJsonConvertFailed    = apperror.NewInternalError("Lỗi khi chuyển đổi thông tin người dùng")
	ErrTempConvertFailed    = apperror.NewInternalError("Lỗi khi chuyển đổi thông tin tạm thời")
	ErrResetTokenGenFailed  = apperror.NewInternalError("Lỗi khi cấp mã khôi phục mật khẩu")
	ErrSessionIssueFailed   = apperror.NewInternalError("Lỗi khi cấp phiên làm việc")
	ErrSessionCreateFailed  = apperror.NewInternalError("Lỗi khi cấp phiên làm việc.")
	ErrAccountCreationFailed = apperror.NewInternalError("Lỗi khi khởi tạo tài khoản mới")

	// Lỗi Xác thực & Thông tin tài khoản
	ErrInvalidCredentials     = apperror.NewBadRequest("Sai tài khoản hoặc mật khẩu.")
	ErrInvalidOldPassword     = apperror.NewBadRequest("Mật khẩu cũ không chính xác.")
	ErrInvalidToken           = apperror.NewUnauthorized("Token không hợp lệ.")
	ErrTokenExpired           = apperror.NewUnauthorized("Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.")
	ErrInvalidSession         = apperror.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	ErrInvalidSessionGeneric   = apperror.NewUnauthorized("Phiên làm việc không hợp lệ.")
	ErrTokenExpiredRecovery   = apperror.NewUnauthorized("Phiên làm việc đã hết hạn hoặc không hợp lệ. Vui lòng thực hiện lại yêu cầu.")
	ErrUserNotFound           = apperror.NewUnauthorized("Người dùng không tồn tại.")
	ErrUserNotFoundGeneric    = apperror.NewUnauthorized("Không tìm thấy người dùng.")
	ErrUserNotFoundDetailed   = apperror.NewUnauthorized("Không tìm thấy người dùng này.")
	ErrUserProfileNotFound    = apperror.NewNotFound("Không tìm thấy thông tin người dùng.")
	ErrEmailNotRegistered     = apperror.NewNotFound("Email này chưa được đăng kí trong hệ thống.")
	ErrAdminAccountNotFound   = apperror.NewUnauthorized("Không tìm thấy thông tin tài khoản admin.")

	// Lỗi Phân quyền & Trạng thái tài khoản
	ErrAccountDisabled       = apperror.NewForbidden(contactSupportMessage)
	ErrAccountDisabledAuth   = apperror.NewUnauthorized(contactSupportMessage)
	ErrSelfStatusUpdate      = apperror.NewForbidden("Bạn không thể tự thay đổi trạng thái tài khoản admin của chính mình.")
	ErrMemberStatusOnly      = apperror.NewForbidden("Chỉ có thể thay đổi trạng thái tài khoản member.")

	// Lỗi riêng biệt ở Presentation Layer (Cookie/HTTP)
	ErrCookieTokenMissing    = apperror.NewBadRequest("Thiếu token gia hạn phiên làm việc trong cookie.")
)

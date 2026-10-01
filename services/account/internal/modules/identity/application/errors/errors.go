package identityerrors

import (
	"errors"
	"fmt"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/shared/application/constants/apperror"
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

// The client-facing sentences for a failed account creation. Each one is FIXED:
// it names the KIND of failure in Vietnamese and never carries text produced by
// a dependency, because `apperror` states plainly that raw error text must not
// leak into `Message` (apperror/error.go:15-17, FR-005 Phase 9). The sibling
// factories that take a cause make the same choice
// (subscription/.../application/errors/errors.go, ErrAdminGrantResolveUsersFailed
// and ErrAdminGrantRecordAuditFailed).
const (
	accountCreationFailedMessage         = "Lỗi khi khởi tạo tài khoản mới kèm gói hội viên mặc định."
	accountCreationNoDefaultPlanMessage  = "Lỗi khi khởi tạo tài khoản mới: không đọc được cấu hình gói hội viên mặc định."
	accountCreationNoSubscriptionMessage = "Lỗi khi khởi tạo tài khoản mới: không tạo được gói hội viên cho tài khoản."
	accountCreationNoCallerTxMessage     = "Lỗi khi khởi tạo tài khoản mới: cấp gói hội viên mặc định không chạy trong cùng giao dịch."
)

// ErrAccountCreationFailedWithCause is the shape of ErrAccountCreationFailed used
// when the failure came from a dependency rather than from the request.
//
// 024 T197: the account and its default entitlement are ONE transaction
// (FR-013), so provisioning is the only place a broken default-plan
// configuration is ever reported - and returning that error unwrapped made
// `POST /admin/users` answer HTTP 404 carrying another module's wording, for an
// operation that neither creates nor looks up a resource.
//
// `apperror.Wrap` cannot fix that: it returns an existing `*apperror.Error`
// unchanged, so a NotFound raised by the subscription module would still reach
// the client with its 404 and its foreign message.
//
// 024 T209: it used to interpolate the cause into the message, which was a
// different defect of the same line and the more dangerous one. The wording a
// dependency produced is internal - English, sometimes a driver string, sometimes
// a stack of it - and this factory handed it to the client verbatim. It also made
// the response body vary per failure, so nothing could be matched on it.
//
// What the operator is owed is preserved, and it is preserved in three places
// that suit it better than `Message` did:
//
//   - `Message` names the KIND of cause, chosen from the dependency's own
//     factories, so "the default plan configuration is missing" still reaches the
//     administrator in Vietnamese while the raw text stays inside.
//   - `DebugMessage` carries the technical detail. The global error middleware
//     clears it and re-derives it from the raw error in non-production only
//     (error_handler.go:88-96), so it can never widen what a client sees.
//   - `Cause` keeps the original error object, which is the only thing the
//     middleware logs (error_handler.go:71-75) and the only thing
//     `errors.Is`/`errors.As` can match further down.
//
// The classification is a switch over the subscription factories because those
// are the only answers `EnsureDefaultSubscription` produces deliberately; anything
// else (a driver error, a repository fault) falls through to the generic
// sentence rather than inventing a diagnosis.
func ErrAccountCreationFailedWithCause(cause error) *apperror.Error {
	e := apperror.NewInternalError(accountCreationFailureMessage(cause))
	e.Cause = cause
	if cause != nil {
		e.DebugMessage = fmt.Sprintf("account creation failed while provisioning the default entitlement: %s", cause)
	}
	return e
}

// accountCreationFailureMessage maps the provisioning failure to the fixed
// Vietnamese sentence that describes its kind.
//
// `errors.Is` is the mechanism, and it is exact rather than textual: every one of
// these factories is a parameterless function returning a fixed
// `*apperror.Error`, and `apperror.Error.Is` compares status and message, so
// `errors.Is(cause, ErrDefaultPlanConfigNotFound())` is true only for that
// factory's own answer. A wrapped cause is still matched, because `errors.Is`
// walks the `Unwrap` chain.
func accountCreationFailureMessage(cause error) string {
	switch {
	case errors.Is(cause, subscriptionerrors.ErrDefaultPlanConfigNotFound()):
		return accountCreationNoDefaultPlanMessage
	case errors.Is(cause, subscriptionerrors.ErrUserSubscriptionNotFound()):
		return accountCreationNoSubscriptionMessage
	case errors.Is(cause, subscriptionerrors.ErrProvisioningRequiresCallerTransaction()):
		return accountCreationNoCallerTxMessage
	default:
		return accountCreationFailedMessage
	}
}

func ErrInvalidCredentials() *apperror.Error {
	return apperror.NewBadRequest("Sai tài khoản hoặc mật khẩu.")
}

// ErrPasswordNotSet is returned when the account exists but has never had a
// password hash (e.g. a Google-only account), so telling the user the password
// is wrong would be misleading. It reveals only that no password is set, not
// anything about the credentials themselves.
func ErrPasswordNotSet() *apperror.Error {
	return apperror.NewBadRequest("Tài khoản của bạn chưa có mật khẩu. Nếu bạn đã đăng ký bằng Google, vui lòng đăng nhập bằng Google. Nếu muốn đăng nhập bằng mật khẩu, hãy sử dụng chức năng Quên mật khẩu để thiết lập mật khẩu mới.")
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

func ErrRegistrationSessionExpired() *apperror.Error {
	return apperror.NewBadRequest("Phiên đăng ký đã hết hạn hoặc không tồn tại. Vui lòng thực hiện đăng ký lại.")
}

func ErrForgotPasswordSessionExpired() *apperror.Error {
	return apperror.NewBadRequest("Yêu cầu khôi phục mật khẩu đã hết hạn hoặc không tồn tại. Vui lòng gửi lại yêu cầu.")
}

func ErrAdminCannotPromoteAdmin() *apperror.Error {
	return apperror.NewForbidden("Chỉ tài khoản admin seed mới được quyền phong hoặc tạo tài khoản admin.")
}

func ErrSelfRoleUpdate() *apperror.Error {
	return apperror.NewForbidden("Bạn không thể tự thay đổi vai trò tài khoản của chính mình.")
}

func ErrCannotModifyAdminRole() *apperror.Error {
	return apperror.NewForbidden("Không thể thay đổi vai trò của tài khoản quản trị viên khác.")
}

func ErrSelfDeactivation() *apperror.Error {
	return apperror.NewForbidden("Bạn không thể tự vô hiệu hóa tài khoản của chính mình.")
}

// Google login errors.
func ErrGoogleTokenInvalid() *apperror.Error {
	return apperror.NewBadRequest("Token Google không hợp lệ hoặc đã hết hạn.")
}

func ErrGoogleEmailUnverified() *apperror.Error {
	return apperror.NewBadRequest("Email Google chưa được xác thực.")
}

func ErrEmailRegisteredLoginPassword() *apperror.Error {
	return apperror.NewBadRequest("Email đã được đăng ký. Vui lòng đăng nhập bằng mật khẩu.")
}

func ErrAccountLinkedOtherGoogle() *apperror.Error {
	return apperror.NewConflict("Email đã được liên kết với một tài khoản Google khác.")
}

func ErrGoogleConfigMissing() *apperror.Error {
	return apperror.NewInternalError("Cấu hình đăng nhập Google chưa được thiết lập.")
}

func ErrGoogleExchangeFailed() *apperror.Error {
	return apperror.NewUnauthorized("Không hoàn tất được đăng nhập Google. Vui lòng thử lại.")
}

func ErrOAuthStateInvalid() *apperror.Error {
	return apperror.NewBadRequest("Phiên đăng nhập Google không hợp lệ hoặc đã hết hạn. Vui lòng thử lại.")
}

func ErrGoogleRedirectInvalid() *apperror.Error {
	return apperror.NewBadRequest("Đường dẫn quay về không hợp lệ.")
}

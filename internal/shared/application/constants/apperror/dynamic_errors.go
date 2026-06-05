package apperror

import "net/http"

func NewBadRequest(detail string) *Error {
	return New(http.StatusBadRequest, "Thao tác không thành công", detail)
}

func NewNotFound(detail string) *Error {
	return New(http.StatusNotFound, "Không tìm thấy dữ liệu", detail)
}

func NewTooManyRequest(detail string) *Error {
	return New(http.StatusTooManyRequests, "Quá nhiều yêu cầu", detail)
}

func NewConflict(detail string) *Error {
	return New(http.StatusConflict, "Dữ liệu bị trùng lặp", detail)
}

func NewUnauthorized(detail string) *Error {
	return New(http.StatusUnauthorized, "Lỗi xác thực", detail)
}

func NewForbidden(detail string) *Error {
	return New(http.StatusForbidden, "Không có quyền truy cập", detail)
}

func NewInternalError(detail string) *Error {
	return New(http.StatusInternalServerError, "Lỗi hệ thống", detail)
}

func NewError(status int, title, detail string) *Error {
	return New(status, title, detail)
}

func ErrInvalidId() *Error {
	return NewBadRequest("Id không hợp lệ")
}

func ErrInvalidImageFormat() *Error {
	return NewBadRequest("Vui lòng chỉ upload ảnh định dạng JPG, PNG hoặc WebP")
}

func ErrImageRequired() *Error {
	return NewBadRequest("Vui lòng chọn ảnh để tải lên")
}

func ErrBusiness() *Error {
	return NewBadRequest("Thao tác không thành công")
}

func ErrInvalidToken() *Error {
	return NewUnauthorized("Token không hợp lệ")
}

func ErrInvalidAccessToken() *Error {
	return NewUnauthorized("4011")
}

func ErrMissingTokenInfo() *Error {
	return NewUnauthorized("Token thiếu thông tin yêu cầu")
}

func ErrUnauthorized() *Error {
	return NewUnauthorized("Vui lòng đăng nhập")
}

func ErrForbidden() *Error {
	return NewForbidden("Không có quyền truy cập")
}

func ErrTokenNotFound() *Error {
	return NewNotFound("không tìm thấy token")
}

func ErrSearchIndexNotFound() *Error {
	return NewNotFound("chỉ mục tìm kiếm chưa tồn tại hoặc đã bị xóa")
}

func ErrConflictDuplicate() *Error {
	return NewConflict("Dữ liệu bị trùng lặp")
}

func ErrTooManyRequests() *Error {
	return NewTooManyRequest("Quá nhiều yêu cầu")
}

func ErrOTPRateLimit() *Error {
	return NewTooManyRequest("Quá số lần gửi OTP")
}

func ErrOTPTooManyAttempts() *Error {
	return NewTooManyRequest("Quá số lần nhập OTP")
}

func ErrInternalServer() *Error {
	return NewInternalError("Lỗi không mong muốn. Vui lòng thử lại")
}

func ErrDbUpdate() *Error {
	return NewInternalError("Lỗi cập nhật dữ liệu")
}

func ErrExportFailed() *Error {
	return NewInternalError("Lỗi xuất file")
}

func ErrUnexpectedSigningToken() *Error {
	return NewInternalError("Token không hợp lệ")
}

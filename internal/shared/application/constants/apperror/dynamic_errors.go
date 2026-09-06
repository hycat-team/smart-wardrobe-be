package apperror

import "net/http"

func NewError(status int, title, detail string) *Error {
	return New(status, title, detail)
}

func NewBadRequest(detail string) *Error {
	return New(http.StatusBadRequest, "Thao tác không thành công", detail)
}

func NewNotFound(detail string) *Error {
	return New(http.StatusNotFound, "Không tìm thấy dữ liệu", detail)
}

func NewTooManyRequest(detail string) *Error {
	return New(http.StatusTooManyRequests, "Quá nhiều yêu cầu", detail)
}

func NewRequestEntityTooLarge(detail string) *Error {
	return New(http.StatusRequestEntityTooLarge, "Dữ liệu gửi lên quá lớn", detail)
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

func NewServiceUnavailable(detail string) *Error {
	return New(http.StatusServiceUnavailable, "Dịch vụ tạm thời gián đoạn", detail)
}

// NewUpstreamUnavailable maps a third-party upstream failure to a client-safe
// 503; the raw provider error travels only as Cause (never serialized) so
// server logs keep the detail.
func NewUpstreamUnavailable(cause error) *Error {
	e := NewServiceUnavailable("Dịch vụ đang bận, vui lòng thử lại sau ít phút.")
	e.Cause = cause
	return e
}

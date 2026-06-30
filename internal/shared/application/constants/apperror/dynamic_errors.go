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

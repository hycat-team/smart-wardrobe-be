package errorcode

import "net/http"

func NewBadRequest(detail string) error {
	return &ErrorResponse{
		Status: http.StatusBadRequest,
		Title:  "Thao tác không thành công",
		Detail: detail,
	}
}

func NewNotFound(detail string) error {
	return &ErrorResponse{
		Status: http.StatusNotFound,
		Title:  "Không tìm thấy dữ liệu",
		Detail: detail,
	}
}

func NewTooManyRequest(detail string) error {
	return &ErrorResponse{
		Status: http.StatusTooManyRequests,
		Title:  "Quá nhiều yêu cầu",
		Detail: detail,
	}
}

func NewConflict(detail string) error {
	return &ErrorResponse{
		Status: http.StatusConflict,
		Title:  "Dữ liệu bị trùng lặp",
		Detail: detail,
	}
}

func NewUnauthorized(detail string) error {
	return &ErrorResponse{
		Status: http.StatusUnauthorized,
		Title:  "Lỗi xác thực",
		Detail: detail,
	}
}

func NewForbidden(detail string) error {
	return &ErrorResponse{
		Status: http.StatusForbidden,
		Title:  "Không có quyền truy cập",
		Detail: detail,
	}
}

func NewInternalError(detail string) error {
	return &ErrorResponse{
		Status: http.StatusInternalServerError,
		Title:  "Lỗi hệ thống",
		Detail: detail,
	}
}

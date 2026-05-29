package errorcode

import (
	"errors"
	"net/http"
)

var (
	// --- 400 Bad Request ---
	ErrInvalidId          = errors.New("Id không hợp lệ")
	ErrInvalidImageFormat = errors.New("Vui lòng chỉ upload ảnh định dạng JPG, PNG hoặc WebP")
	ErrImageRequired      = errors.New("Vui lòng chọn ảnh để tải lên")
	ErrBusiness           = errors.New("Thao tác không thành công")

	// --- 401 Unauthorized ---
	ErrInvalidToken       = errors.New("Token không hợp lệ")
	ErrInvalidAccessToken = errors.New("4011")
	ErrMissingTokenInfo   = errors.New("Token thiếu thông tin yêu cầu")
	ErrUnauthorized       = errors.New("Vui lòng đăng nhập")

	// --- 403 Forbidden ---
	ErrForbidden = errors.New("Không có quyền truy cập")

	// --- 404 Not Found ---
	ErrTokenNotFound = errors.New("không tìm thấy token")

	// --- 409 Conflict ---
	ErrConflictDuplicate = errors.New("Dữ liệu bị trùng lặp")

	// --- 429 Too Many Requests ---
	ErrTooManyRequests    = errors.New("Quá nhiều yêu cầu")
	ErrOTPRateLimit       = errors.New("Quá số lần gửi OTP")
	ErrOTPTooManyAttempts = errors.New("Quá số lần nhập OTP")

	// --- 500 Internal Server Error ---
	ErrInternalServer         = errors.New("Lỗi không mong muốn. Vui lòng thử lại")
	ErrDbUpdate               = errors.New("Lỗi cập nhật dữ liệu")
	ErrExportFailed           = errors.New("Lỗi xuất file")
	ErrUnexpectedSigningToken = errors.New("Token không hợp lệ")
)

var InternalErrorMap map[error]ErrorResponse

func InitErrorMap() {
	InternalErrorMap = map[error]ErrorResponse{
		// --- 400 ---
		ErrInvalidId: {
			Status: http.StatusBadRequest,
			Title:  "Thao tác không thành công",
			Detail: ErrInvalidId.Error(),
		},
		ErrInvalidImageFormat: {
			Status: http.StatusBadRequest,
			Title:  "Thao tác không thành công",
			Detail: ErrInvalidImageFormat.Error(),
		},
		ErrImageRequired: {
			Status: http.StatusBadRequest,
			Title:  "Thao tác không thành công",
			Detail: ErrImageRequired.Error(),
		},
		ErrBusiness: {
			Status: http.StatusBadRequest,
			Title:  "Thao tác không thành công",
			Detail: ErrBusiness.Error(),
		},

		// --- 401 ---
		ErrInvalidToken: {
			Status: http.StatusUnauthorized,
			Title:  "Lỗi xác thực",
			Detail: ErrInvalidToken.Error(),
		},
		ErrInvalidAccessToken: {
			Status: http.StatusUnauthorized,
			Title:  "Lỗi xác thực",
			Detail: ErrInvalidAccessToken.Error(),
		},
		ErrMissingTokenInfo: {
			Status: http.StatusUnauthorized,
			Title:  "Lỗi xác thực",
			Detail: ErrMissingTokenInfo.Error(),
		},
		ErrUnauthorized: {
			Status: http.StatusUnauthorized,
			Title:  "Lỗi xác thực",
			Detail: ErrUnauthorized.Error(),
		},

		// --- 403 ---
		ErrForbidden: {
			Status: http.StatusForbidden,
			Title:  "Không có quyền truy cập",
			Detail: ErrForbidden.Error(),
		},

		// --- 404 ---
		ErrTokenNotFound: {
			Status: http.StatusNotFound,
			Title:  "Không tìm thấy",
			Detail: ErrTokenNotFound.Error(),
		},

		// --- 409 ---
		ErrConflictDuplicate: {
			Status: http.StatusConflict,
			Title:  "Dữ liệu bị trùng lặp",
			Detail: ErrConflictDuplicate.Error(),
		},

		// --- 429 ---
		ErrTooManyRequests: {
			Status: http.StatusTooManyRequests,
			Title:  "Quá nhiều yêu cầu",
			Detail: ErrTooManyRequests.Error(),
		},
		ErrOTPRateLimit: {
			Status: http.StatusTooManyRequests,
			Title:  "Quá nhiều yêu cầu",
			Detail: ErrOTPRateLimit.Error(),
		},
		ErrOTPTooManyAttempts: {
			Status: http.StatusTooManyRequests,
			Title:  "Quá nhiều yêu cầu",
			Detail: ErrOTPTooManyAttempts.Error(),
		},

		// --- 500 ---
		ErrInternalServer: {
			Status: http.StatusInternalServerError,
			Title:  "Lỗi hệ thống",
			Detail: ErrInternalServer.Error(),
		},
		ErrDbUpdate: {
			Status: http.StatusInternalServerError,
			Title:  "Lỗi hệ thống",
			Detail: ErrDbUpdate.Error(),
		},
		ErrExportFailed: {
			Status: http.StatusInternalServerError,
			Title:  "Lỗi hệ thống",
			Detail: ErrExportFailed.Error(),
		},
		ErrUnexpectedSigningToken: {
			Status: http.StatusInternalServerError,
			Title:  "Lỗi hệ thống",
			Detail: ErrUnexpectedSigningToken.Error(),
		},
	}
}

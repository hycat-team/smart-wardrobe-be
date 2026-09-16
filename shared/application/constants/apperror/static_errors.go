package apperror

func ErrInvalidToken() *Error {
	return NewUnauthorized("Token không hợp lệ")
}

func ErrInvalidAccessToken() *Error {
	return NewUnauthorized("4011")
}

func ErrUnauthorized() *Error {
	return NewUnauthorized("Vui lòng đăng nhập")
}

func ErrForbidden() *Error {
	return NewForbidden("Không có quyền truy cập")
}

func ErrSearchIndexNotFound() *Error {
	return NewNotFound("Chỉ mục tìm kiếm chưa tồn tại hoặc đã bị xóa")
}

func ErrUnexpectedSigningToken() *Error {
	return NewInternalError("Token không hợp lệ")
}

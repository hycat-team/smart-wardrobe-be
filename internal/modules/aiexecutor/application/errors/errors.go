package errors

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrPricingConfigNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cấu hình giá mô hình AI.")
}

func ErrGrantPolicyNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy chính sách chi phí AI.")
}

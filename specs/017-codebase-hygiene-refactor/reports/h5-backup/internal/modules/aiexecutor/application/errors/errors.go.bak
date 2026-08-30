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

func ErrEffectiveToBeforeEffectiveFrom() *apperror.Error {
	return apperror.NewBadRequest("Thời hạn hiệu lực kết thúc phải sau thời điểm bắt đầu hiệu lực.")
}

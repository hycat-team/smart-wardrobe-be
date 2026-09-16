package errors

import (
	"smart-wardrobe-be/shared/application/constants/apperror"
)

func ErrPricingConfigNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cấu hình giá mô hình AI.")
}

func ErrEffectiveToBeforeEffectiveFrom() *apperror.Error {
	return apperror.NewBadRequest("Thời hạn hiệu lực kết thúc phải sau thời điểm bắt đầu hiệu lực.")
}

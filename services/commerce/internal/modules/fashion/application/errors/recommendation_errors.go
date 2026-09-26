package fashioerrors

import (
	"smart-wardrobe-be/shared/application/constants/apperror"
)

func ErrInsufficientOutfitComponents() *apperror.Error {
	return apperror.NewBadRequest("Chưa ghép được bộ hoàn chỉnh. Thử nới bộ lọc hoặc thêm trang phục nhé!")
}

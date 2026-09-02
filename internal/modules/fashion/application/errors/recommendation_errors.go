package fashioerrors

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrInsufficientOutfitComponents() *apperror.Error {
	return apperror.NewBadRequest("Tủ đồ của bạn không đủ trang phục phù hợp (cần có ít nhất 1 Áo và 1 Quần/Chân váy, hoặc 1 Đầm) để phối thành bộ hoàn chỉnh theo yêu cầu. Hãy đăng tải thêm đồ hoặc thay đổi bộ lọc nhé!")
}

package fashioerrors

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrChatNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cuộc trò chuyện này.")
}

func ErrMinimumWardrobeItemsRequired() *apperror.Error {
	return apperror.NewBadRequest("Tủ đồ của bạn cần có ít nhất 5 trang phục để hệ thống có thể tiến hành phối đồ. Hãy đăng tải thêm đồ nhé!")
}

func ErrAiOutfitQuotaExceeded() *apperror.Error {
	return apperror.NewBadRequest("Bạn đã dùng hết lượt tạo trang phục bằng AI trong hôm nay.")
}

func ErrAiChatQuotaExceeded() *apperror.Error {
	return apperror.NewBadRequest("Bạn đã dùng hết lượt trò chuyện với AI Chatbot trong hôm nay.")
}

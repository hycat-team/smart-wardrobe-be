package wardrobeerrors

import (
	"fmt"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

// Dynamic errors (dynamic messages with parameters)
func ErrItemNotFoundWithID(itemID any) *apperror.Error {
	return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy trang phục (Mã: %s).", itemID))
}

func ErrItemForbiddenWithID(itemID any) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Trang phục (Mã: %s) không thuộc tủ đồ của bạn.", itemID))
}

func ErrItemSoldWithID(itemID any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang phục (Mã: %s) đã được bán nên không thể đăng bài.", itemID))
}

func ErrWardrobeLimitExceededForClone(currentCount, maxItems, quantity int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Số lượng bản sao vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, maxItems, quantity))
}

func ErrWardrobeLimitExceededForCatalog(currentCount, maxItems, requestedCount int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Số lượng trang phục khởi tạo vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, maxItems, requestedCount))
}

func ErrWardrobeLimitExceededForUpload(currentCount, maxItems, requestedCount int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Số lượng trang phục tải lên vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, maxItems, requestedCount))
}

func ErrItemLockedDueToLimit(maxItems int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Trang phục này đã bị khóa vì tủ đồ vượt quá giới hạn tối đa của gói dịch vụ (Tối đa %d món).", maxItems))
}

func ErrOutfitLimitReached(currentOutfits, maxOutfits int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Bạn đã đạt giới hạn số bộ phối đồ tối đa của gói dịch vụ hiện tại (%d/%d bộ).", currentOutfits, maxOutfits))
}

func ErrOutfitItemSold(itemID any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang phục (Mã: %s) đã được bán nên không thể dùng để phối đồ.", itemID))
}

func ErrOutfitItemInvalidOrForbidden(itemID any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang phục (Mã: %s) không tồn tại hoặc không thuộc tủ đồ của bạn.", itemID))
}

// Static errors
var (
	// Presentation / Validation
	ErrInvalidChatIDFormat   = apperror.NewBadRequest("Định dạng mã cuộc trò chuyện không hợp lệ.")
	ErrInvalidOutfitIDFormat = apperror.NewBadRequest("Định dạng mã bộ phối đồ không hợp lệ.")

	// Search
	ErrSearchItemsFailed = apperror.NewInternalError("Đã xảy ra lỗi hệ thống trong quá trình tìm kiếm trang phục.")

	// Items & Categories
	ErrItemNotFound             = apperror.NewNotFound("Không tìm thấy trang phục này.")
	ErrUpdateItemForbidden      = apperror.NewForbidden("Bạn không được phép cập nhật thông tin trang phục này.")
	ErrManualClassifySoldItem   = apperror.NewBadRequest("Không thể phân loại thủ công trang phục đã bán.")
	ErrCategoryNotFound         = apperror.NewBadRequest("Danh mục trang phục không tồn tại.")
	ErrProcessFashionTextFailed = apperror.NewInternalError("Hệ thống không thể xử lý nội dung văn bản thời trang lúc này.")

	// Clone & Catalog & Upload
	ErrItemToCloneNotFound         = apperror.NewNotFound("Không tìm thấy trang phục cần sao chép.")
	ErrInvalidCloneQuantity        = apperror.NewBadRequest("Bạn chỉ có thể tạo từ 1 đến 5 bản sao của trang phục này.")
	ErrOriginalItemToCloneNotFound = apperror.NewNotFound("Không tìm thấy trang phục gốc để sao chép.")
	ErrCloneOtherUserItemForbidden = apperror.NewForbidden("Bạn không thể sao chép trang phục của người dùng khác.")
	ErrCloneSoldItem               = apperror.NewBadRequest("Không thể sao chép trang phục đã được bán.")
	ErrCatalogItemIDsEmpty         = apperror.NewBadRequest("Danh sách trang phục mẫu không được để trống.")
	ErrCatalogItemNotFound         = apperror.NewNotFound("Không tìm thấy trang phục mẫu phù hợp.")
	ErrUploadImagesEmpty           = apperror.NewBadRequest("Danh sách hình ảnh tải lên không được để trống.")

	// AI & Chat
	ErrNoSuitableItemsForOutfit = apperror.NewBadRequest("Tủ đồ của bạn chưa có trang phục thích hợp để phối đồ.")
	ErrMinimumWardrobeItemsRequired = apperror.NewBadRequest("Tủ đồ của bạn cần có ít nhất 5 trang phục để hệ thống có thể tiến hành phối đồ. Hãy đăng tải thêm đồ nhé!")
	ErrNoOutfitsFound           = apperror.NewBadRequest("Không tìm thấy bộ phối đồ nào phù hợp trong tủ đồ của bạn.")
	ErrChatNotFound             = apperror.NewNotFound("Không tìm thấy cuộc trò chuyện này.")
	ErrInvalidOutfitStructure   = apperror.NewInternalError("Cấu trúc bộ phối đồ từ AI không hợp lệ.")

	// Outfits
	ErrOutfitNotFound = apperror.NewNotFound("Không tìm thấy bộ phối đồ này.")
)

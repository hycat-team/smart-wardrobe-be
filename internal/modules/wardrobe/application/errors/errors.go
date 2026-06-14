package wardrobeerrors

import (
	"fmt"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

// ErrItemNotFoundWithID returns a not found error for a specific wardrobe item.
func ErrItemNotFoundWithID(itemID any) *apperror.Error {
	return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy trang phục (Mã: %s).", itemID))
}

// ErrItemForbiddenWithID returns a forbidden error for a specific wardrobe item.
func ErrItemForbiddenWithID(itemID any) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Trang phục (Mã: %s) không thuộc tủ đồ của bạn.", itemID))
}

// ErrItemSoldWithID returns a bad request error when a sold item is reused incorrectly.
func ErrItemSoldWithID(itemID any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang phục (Mã: %s) đã được bán nên không thể đăng bài.", itemID))
}

// ErrWardrobeLimitExceededForClone returns a quota error for clone requests.
func ErrWardrobeLimitExceededForClone(currentCount, maxItems, quantity int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Số lượng bản sao vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, maxItems, quantity))
}

// ErrWardrobeLimitExceededForCatalog returns a quota error for closet initialization.
func ErrWardrobeLimitExceededForCatalog(currentCount, maxItems, requestedCount int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Số lượng trang phục khởi tạo vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, maxItems, requestedCount))
}

// ErrWardrobeLimitExceededForUpload returns a quota error for upload requests.
func ErrWardrobeLimitExceededForUpload(currentCount, maxItems, requestedCount int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Số lượng trang phục tải lên vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, maxItems, requestedCount))
}

// ErrItemLockedDueToLimit returns a lock error when an item is outside the active wardrobe quota.
func ErrItemLockedDueToLimit(maxItems int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Trang phục này đã bị khóa vì tủ đồ vượt quá giới hạn tối đa của gói dịch vụ (Tối đa %d món).", maxItems))
}

// ErrOutfitLimitReached returns an error when the outfit quota is exhausted.
func ErrOutfitLimitReached(currentOutfits, maxOutfits int) *apperror.Error {
	return apperror.NewForbidden(fmt.Sprintf("Bạn đã đạt giới hạn số bộ phối đồ tối đa của gói dịch vụ hiện tại (%d/%d bộ).", currentOutfits, maxOutfits))
}

// ErrOutfitItemSold returns an error when a sold item is used for an outfit.
func ErrOutfitItemSold(itemID any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang phục (Mã: %s) đã được bán nên không thể dùng để phối đồ.", itemID))
}

// ErrOutfitItemInvalidOrForbidden returns an error when an outfit item is missing or inaccessible.
func ErrOutfitItemInvalidOrForbidden(itemID any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang phục (Mã: %s) không tồn tại hoặc không thuộc tủ đồ của bạn.", itemID))
}

var (
	ErrInvalidChatIDFormat   = apperror.NewBadRequest("Định dạng mã cuộc trò chuyện không hợp lệ.")
	ErrInvalidOutfitIDFormat = apperror.NewBadRequest("Định dạng mã bộ phối đồ không hợp lệ.")

	ErrSearchItemsFailed = apperror.NewInternalError("Đã xảy ra lỗi hệ thống trong quá trình tìm kiếm trang phục.")

	ErrItemNotFound                    = apperror.NewNotFound("Không tìm thấy trang phục này.")
	ErrUpdateItemForbidden             = apperror.NewForbidden("Bạn không được phép cập nhật thông tin trang phục này.")
	ErrManualClassifySoldItem          = apperror.NewBadRequest("Không thể phân loại thủ công trang phục đã bán.")
	ErrCategoryNotFound                = apperror.NewBadRequest("Danh mục trang phục không tồn tại.")
	ErrCategoryNameAlreadyExists       = apperror.NewConflict("Tên danh mục đã tồn tại trong hệ thống.")
	ErrCategorySlugAlreadyExists       = apperror.NewConflict("Slug danh mục đã tồn tại trong hệ thống.")
	ErrCategoryOtherImmutable          = apperror.NewBadRequest("Không thể xóa danh mục hệ thống 'other'.")
	ErrCategoryHasUserItems            = apperror.NewBadRequest("Không thể xóa danh mục này vì vẫn còn trang phục của người dùng đang liên kết.")
	ErrFallbackCategoryNotFound        = apperror.NewInternalError("Không tìm thấy danh mục hệ thống 'other' để chuyển dữ liệu.")
	ErrProcessFashionTextFailed        = apperror.NewInternalError("Hệ thống không thể xử lý nội dung văn bản thời trang lúc này.")
	ErrRetryWardrobeAnalysisForbidden  = apperror.NewBadRequest("Chỉ có thể thử phân tích lại trang phục đang lỗi hoặc cần rà soát.")
	ErrRetryWardrobeAnalysisInProgress = apperror.NewBadRequest("Trang phục này đang được xử lý hoặc không còn có thể thử phân tích lại.")
	ErrRetryWardrobeAnalysisCooldown   = apperror.NewBadRequest("Bạn vừa thử phân tích lại trang phục này. Vui lòng đợi một chút rồi thử lại.")

	ErrItemToCloneNotFound         = apperror.NewNotFound("Không tìm thấy trang phục cần sao chép.")
	ErrInvalidCloneQuantity        = apperror.NewBadRequest("Bạn chỉ có thể tạo từ 1 đến 5 bản sao của trang phục này.")
	ErrOriginalItemToCloneNotFound = apperror.NewNotFound("Không tìm thấy trang phục gốc để sao chép.")
	ErrCloneOtherUserItemForbidden = apperror.NewForbidden("Bạn không thể sao chép trang phục của người dùng khác.")
	ErrCloneSoldItem               = apperror.NewBadRequest("Không thể sao chép trang phục đã được bán.")
	ErrCatalogItemIDsEmpty         = apperror.NewBadRequest("Danh sách trang phục mẫu không được để trống.")
	ErrCatalogItemNotFound         = apperror.NewNotFound("Không tìm thấy trang phục mẫu phù hợp.")
	ErrUploadImagesEmpty           = apperror.NewBadRequest("Danh sách hình ảnh tải lên không được để trống.")

	ErrNoSuitableItemsForOutfit     = apperror.NewBadRequest("Tủ đồ của bạn chưa có trang phục thích hợp để phối đồ.")
	ErrMinimumWardrobeItemsRequired = apperror.NewBadRequest("Tủ đồ của bạn cần có ít nhất 5 trang phục để hệ thống có thể tiến hành phối đồ. Hãy đăng tải thêm đồ nhé!")
	ErrNoOutfitsFound               = apperror.NewBadRequest("Không tìm thấy bộ phối đồ nào phù hợp trong tủ đồ của bạn.")
	ErrChatNotFound                 = apperror.NewNotFound("Không tìm thấy cuộc trò chuyện này.")
	ErrInvalidOutfitStructure       = apperror.NewInternalError("Cấu trúc bộ phối đồ từ AI không hợp lệ.")

	ErrOutfitNotFound = apperror.NewNotFound("Không tìm thấy bộ phối đồ này.")
)

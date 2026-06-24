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

func ErrInvalidChatIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Định dạng mã cuộc trò chuyện không hợp lệ.")
}

func ErrInvalidOutfitIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Định dạng mã bộ phối đồ không hợp lệ.")
}

func ErrSearchItemsFailed() *apperror.Error {
	return apperror.NewInternalError("Đã xảy ra lỗi hệ thống trong quá trình tìm kiếm trang phục.")
}

func ErrInvalidWardrobeStatusFilter() *apperror.Error {
	return apperror.NewBadRequest("Giá trị bộ lọc trạng thái trang phục không hợp lệ.")
}

func ErrItemNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy trang phục này.")
}

func ErrUpdateItemForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không được phép cập nhật thông tin trang phục này.")
}

func ErrManualClassifySoldItem() *apperror.Error {
	return apperror.NewBadRequest("Không thể phân loại thủ công trang phục đã bán.")
}

func ErrCategoryNotFound() *apperror.Error {
	return apperror.NewBadRequest("Danh mục trang phục không tồn tại.")
}

func ErrCategoryNameAlreadyExists() *apperror.Error {
	return apperror.NewConflict("Tên danh mục đã tồn tại trong hệ thống.")
}

func ErrCategorySlugAlreadyExists() *apperror.Error {
	return apperror.NewConflict("Slug danh mục đã tồn tại trong hệ thống.")
}

func ErrCategoryLegacySlugForbidden() *apperror.Error {
	return apperror.NewBadRequest("Slug danh mục 'vay' đã ngừng hỗ trợ. Vui lòng dùng 'dam' hoặc 'chan-vay'.")
}

func ErrCategoryOtherImmutable() *apperror.Error {
	return apperror.NewBadRequest("Không thể xóa danh mục hệ thống 'other'.")
}

func ErrCategoryHasUserItems() *apperror.Error {
	return apperror.NewBadRequest("Không thể xóa danh mục này vì vẫn còn trang phục của người dùng đang liên kết.")
}

func ErrFallbackCategoryNotFound() *apperror.Error {
	return apperror.NewInternalError("Không tìm thấy danh mục hệ thống 'other' để chuyển dữ liệu.")
}

func ErrProcessFashionTextFailed() *apperror.Error {
	return apperror.NewInternalError("Hệ thống không thể xử lý nội dung văn bản thời trang lúc này.")
}

func ErrRetryWardrobeAnalysisForbidden() *apperror.Error {
	return apperror.NewBadRequest("Chỉ có thể thử phân tích lại trang phục đang lỗi hoặc cần rà soát.")
}

func ErrRetryWardrobeAnalysisInProgress() *apperror.Error {
	return apperror.NewBadRequest("Trang phục này đang được xử lý hoặc không còn có thể thử phân tích lại.")
}

func ErrRetryWardrobeAnalysisCooldown() *apperror.Error {
	return apperror.NewBadRequest("Bạn vừa thử phân tích lại trang phục này. Vui lòng đợi một chút rồi thử lại.")
}

func ErrItemToCloneNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy trang phục cần sao chép.")
}

func ErrInvalidCloneQuantity() *apperror.Error {
	return apperror.NewBadRequest("Bạn chỉ có thể tạo từ 1 đến 5 bản sao của trang phục này.")
}

func ErrOriginalItemToCloneNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy trang phục gốc để sao chép.")
}

func ErrCloneOtherUserItemForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không thể sao chép trang phục của người dùng khác.")
}

func ErrCloneSoldItem() *apperror.Error {
	return apperror.NewBadRequest("Không thể sao chép trang phục đã được bán.")
}

func ErrCatalogItemIDsEmpty() *apperror.Error {
	return apperror.NewBadRequest("Danh sách trang phục mẫu không được để trống.")
}

func ErrCatalogItemNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy trang phục mẫu phù hợp.")
}

func ErrUploadImagesEmpty() *apperror.Error {
	return apperror.NewBadRequest("Danh sách hình ảnh tải lên không được để trống.")
}

func ErrNoSuitableItemsForOutfit() *apperror.Error {
	return apperror.NewBadRequest("Tủ đồ của bạn chưa có trang phục thích hợp để phối đồ.")
}

func ErrMinimumWardrobeItemsRequired() *apperror.Error {
	return apperror.NewBadRequest("Tủ đồ của bạn cần có ít nhất 5 trang phục để hệ thống có thể tiến hành phối đồ. Hãy đăng tải thêm đồ nhé!")
}

func ErrNoOutfitsFound() *apperror.Error {
	return apperror.NewBadRequest("Không tìm thấy bộ phối đồ nào phù hợp trong tủ đồ của bạn.")
}

func ErrChatNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cuộc trò chuyện này.")
}

func ErrInvalidOutfitStructure() *apperror.Error {
	return apperror.NewInternalError("Cấu trúc bộ phối đồ từ AI không hợp lệ.")
}

func ErrOutfitNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy bộ phối đồ này.")
}

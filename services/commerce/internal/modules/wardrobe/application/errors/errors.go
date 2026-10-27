package wardrobeerrors

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

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

func ErrCategoryNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy danh mục.")
}

func ErrRetryWardrobeAnalysisForbidden() *apperror.Error {
	return apperror.NewBadRequest("Chỉ có thể thử phân tích lại trang phục đang lỗi hoặc cần rà soát.")
}

// ErrRetryCategoryRequired returns an error when a needs-review item is retried
// without a valid category (mandatory to re-analyze at a fixed category).
func ErrRetryCategoryRequired() *apperror.Error {
	return apperror.NewBadRequest("Vui lòng chọn danh mục hợp lệ (có thể phối đồ) trước khi thử lại trang phục cần rà soát.")
}

func ErrRetryWardrobeAnalysisInProgress() *apperror.Error {
	return apperror.NewBadRequest("Trang phục này đang được xử lý hoặc không còn có thể thử phân tích lại.")
}

// ErrRetryInvalidImage rejects retry for items whose image is unusable (multiple
// items / full body / no garment at all); the user must upload a different
// image (FR-033). The generated metadata of such an item is not user-editable:
// PATCH /wardrobe-items/:id only accepts price and categoryId, so deleting and
// re-uploading is the only escape.
func ErrRetryInvalidImage() *apperror.Error {
	return apperror.NewBadRequest("Không thể thử lại vì ảnh không hợp lệ (không phải trang phục, nhiều món hoặc ảnh toàn thân). Vui lòng tải ảnh khác.")
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

func ErrOutfitNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy bộ phối đồ này.")
}

// ErrCatalogItemUnavailable returns a 400 when a system catalog item is no
// longer eligible to be saved in an outfit (removed, not InWardrobe, or without
// a fashion item). Feature 033 FR-029/FR-033.
func ErrCatalogItemUnavailable() *apperror.Error {
	return apperror.NewBadRequest("Món đồ trong bộ sưu tập hệ thống không còn khả dụng. Vui lòng chọn món khác.")
}

// ErrWardrobeLimitExceededForOutfit returns a 400 when cloning the requested
// catalog items would push the user's wardrobe over the plan limit. The whole
// save is rejected and the offending fashion item ids are listed (FR-032).
func ErrWardrobeLimitExceededForOutfit(itemIDs []uuid.UUID) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Không thể lưu bộ phối đồ vì các món đồ hệ thống cần thêm vào tủ vượt quá giới hạn của gói dịch vụ hiện tại (Mã: %s).", formatItemIDs(itemIDs)))
}

// ErrDuplicateOutfitItem returns a 400 when the same fashion item appears more
// than once in the outfit payload (FR-033).
func ErrDuplicateOutfitItem(fashionItemID uuid.UUID) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Bộ phối đồ không được chứa hai mục trùng nhau cho cùng một trang phục (Mã: %s).", fashionItemID))
}

func formatItemIDs(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	return strings.Join(parts, ", ")
}

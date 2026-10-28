package branderrors

import "smart-wardrobe-be/shared/application/constants/apperror"

// ErrBrandItemStatusLocked rejects any manual status change while an item is in
// the terminal `failed` state (feature 023, US4).
func ErrBrandItemStatusLocked() *apperror.Error {
	return apperror.NewForbidden("Sản phẩm đang ở trạng thái lỗi; chỉ có thể thử phân tích lại để đưa sản phẩm ra khỏi trạng thái này.")
}

// ErrBrandItemInvalidStatus rejects an unsupported status value on update.
func ErrBrandItemInvalidStatus() *apperror.Error {
	return apperror.NewBadRequest("Giá trị trạng thái sản phẩm không hợp lệ.")
}

// ErrBrandRetryForbidden rejects retry when the item is not in a retryable state.
func ErrBrandRetryForbidden() *apperror.Error {
	return apperror.NewBadRequest("Chỉ có thể thử phân tích lại sản phẩm đang lỗi hoặc sản phẩm nháp đang chờ duyệt.")
}

// ErrBrandRetryInProgress rejects retry when the item is already being processed.
func ErrBrandRetryInProgress() *apperror.Error {
	return apperror.NewBadRequest("Sản phẩm này đang được xử lý hoặc không thể thử phân tích lại lúc này.")
}

// ErrBrandRetryInvalidImage rejects retry for items whose image is unusable
// (multiple items / full body / no garment at all); staff must create a new
// item with another image.
func ErrBrandRetryInvalidImage() *apperror.Error {
	return apperror.NewBadRequest("Không thể thử lại vì ảnh không hợp lệ (không phải trang phục, nhiều món hoặc ảnh toàn thân). Vui lòng tạo sản phẩm mới với ảnh khác.")
}

// ErrBrandRetryCategoryRequired rejects retry of a draft awaiting review when no
// valid wearable category is supplied (feature 023, FR-041).
func ErrBrandRetryCategoryRequired() *apperror.Error {
	return apperror.NewBadRequest("Vui lòng chọn danh mục hợp lệ (có thể mặc được) trước khi thử lại sản phẩm đang chờ duyệt.")
}

// ErrBrandItemNotSample rejects converting an item that is not a digital sample.
func ErrBrandItemNotSample() *apperror.Error {
	return apperror.NewBadRequest("Chỉ có thể chuyển mẫu thử (sample) thành sản phẩm.")
}

// ErrBrandItemAlreadyProduct rejects converting an item that is already a product.
func ErrBrandItemAlreadyProduct() *apperror.Error {
	return apperror.NewConflict("Sản phẩm này đã là sản phẩm thương mại.")
}

// ErrBrandItemNotReadyForProduct rejects conversion before AI analysis has
// produced a garment category.
func ErrBrandItemNotReadyForProduct() *apperror.Error {
	return apperror.NewBadRequest("Mẫu thử phải được AI phân tích xong và có danh mục trước khi chuyển thành sản phẩm.")
}

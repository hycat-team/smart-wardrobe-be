package communityerrors

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrInvalidPostIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Định dạng mã bài đăng không hợp lệ.")
}

func ErrInvalidPostPublicIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Định dạng mã công khai bài đăng không hợp lệ.")
}

func ErrInvalidCommentIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Định dạng mã bình luận không hợp lệ.")
}

func ErrInvalidPostItemIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Định dạng mã sản phẩm trong bài đăng không hợp lệ.")
}

func ErrInvalidUserIDFormat() *apperror.Error {
	return apperror.NewBadRequest("Mã định danh người dùng không hợp lệ.")
}

func ErrInvalidSortCriterion() *apperror.Error {
	return apperror.NewBadRequest("Tiêu chí sắp xếp không hợp lệ.")
}

func ErrInvalidPostType() *apperror.Error {
	return apperror.NewBadRequest("Hình thức bài viết không hợp lệ.")
}

func ErrInvalidParentCommentTarget() *apperror.Error {
	return apperror.NewBadRequest("Chỉ được phản hồi trực tiếp vào bình luận cấp đầu.")
}

func ErrActiveTransferExists() *apperror.Error {
	return apperror.NewBadRequest("Trang phục này đang có giao dịch chờ xử lý, không thể đăng bán thêm.")
}

func ErrPostNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy bài viết này.")
}

func ErrPostItemNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy listing này.")
}

func ErrPostSaleItemsRequired() *apperror.Error {
	return apperror.NewBadRequest("Bài viết đăng bán phải chứa ít nhất một sản phẩm.")
}

func ErrPostContactInfoRequired() *apperror.Error {
	return apperror.NewBadRequest("Bài viết đăng bán phải đính kèm thông tin liên hệ.")
}

func ErrPostItemPriceRequired() *apperror.Error {
	return apperror.NewBadRequest("Mỗi món đồ đăng bán phải có giá hợp lệ.")
}

func ErrCommentNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy bình luận này.")
}

func ErrEditCommentForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không được phép chỉnh sửa bình luận này.")
}

func ErrDeleteCommentForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không được phép xóa bình luận này.")
}

func ErrCommentReplyTargetInvalid() *apperror.Error {
	return apperror.NewBadRequest("Không thể phản hồi vào bình luận này.")
}

func ErrRequestedProductNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy sản phẩm được yêu cầu.")
}

func ErrTransferForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không được phép thực hiện thao tác này.")
}

func ErrItemInAnotherTransfer() *apperror.Error {
	return apperror.NewBadRequest("Trang phục này đang nằm trong một giao dịch khác.")
}

func ErrTransferRequestInvalid() *apperror.Error {
	return apperror.NewBadRequest("Yêu cầu chuyển nhượng này đã được xử lý hoặc không còn hiệu lực.")
}

func ErrBuyerNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy người mua được chỉ định.")
}

func ErrNoPendingRequest() *apperror.Error {
	return apperror.NewBadRequest("Người mua này chưa gửi yêu cầu xin mua sản phẩm này.")
}

func ErrBuyerSelfRequest() *apperror.Error {
	return apperror.NewBadRequest("Bạn không thể tự gửi yêu cầu mua sản phẩm của chính mình.")
}

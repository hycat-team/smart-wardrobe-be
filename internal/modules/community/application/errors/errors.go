package communityerrors

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

var (
	ErrInvalidPostIDFormat        = apperror.NewBadRequest("Định dạng mã bài đăng không hợp lệ.")
	ErrInvalidPostPublicIDFormat  = apperror.NewBadRequest("Định dạng mã công khai bài đăng không hợp lệ.")
	ErrInvalidCommentIDFormat     = apperror.NewBadRequest("Định dạng mã bình luận không hợp lệ.")
	ErrInvalidPostItemIDFormat    = apperror.NewBadRequest("Định dạng mã sản phẩm trong bài đăng không hợp lệ.")
	ErrInvalidUserIDFormat        = apperror.NewBadRequest("Mã định danh người dùng không hợp lệ.")
	ErrInvalidSortCriterion       = apperror.NewBadRequest("Tiêu chí sắp xếp không hợp lệ.")
	ErrInvalidPostType            = apperror.NewBadRequest("Hình thức bài viết không hợp lệ.")
	ErrInvalidParentCommentTarget = apperror.NewBadRequest("Chỉ được phản hồi trực tiếp vào bình luận cấp đầu.")

	ErrActiveTransferExists    = apperror.NewBadRequest("Trang phục này đang có giao dịch chờ xử lý, không thể đăng bán thêm.")
	ErrPostNotFound            = apperror.NewNotFound("Không tìm thấy bài viết này.")
	ErrPostItemNotFound        = apperror.NewNotFound("Không tìm thấy listing này.")
	ErrPostSaleItemsRequired   = apperror.NewBadRequest("Bài viết đăng bán phải chứa ít nhất một sản phẩm.")
	ErrPostContactInfoRequired = apperror.NewBadRequest("Bài viết đăng bán phải đính kèm thông tin liên hệ.")
	ErrPostItemPriceRequired   = apperror.NewBadRequest("Mỗi món đồ đăng bán phải có giá hợp lệ.")

	ErrCommentNotFound           = apperror.NewNotFound("Không tìm thấy bình luận này.")
	ErrEditCommentForbidden      = apperror.NewForbidden("Bạn không được phép chỉnh sửa bình luận này.")
	ErrDeleteCommentForbidden    = apperror.NewForbidden("Bạn không được phép xóa bình luận này.")
	ErrCommentReplyTargetInvalid = apperror.NewBadRequest("Không thể phản hồi vào bình luận này.")

	ErrRequestedProductNotFound = apperror.NewNotFound("Không tìm thấy sản phẩm được yêu cầu.")
	ErrTransferForbidden        = apperror.NewForbidden("Bạn không được phép thực hiện thao tác này.")
	ErrItemInAnotherTransfer    = apperror.NewBadRequest("Trang phục này đang nằm trong một giao dịch khác.")
	ErrTransferRequestInvalid   = apperror.NewBadRequest("Yêu cầu chuyển nhượng này đã được xử lý hoặc không còn hiệu lực.")
	ErrBuyerNotFound            = apperror.NewNotFound("Không tìm thấy người mua được chỉ định.")
	ErrNoPendingRequest         = apperror.NewBadRequest("Người mua này chưa gửi yêu cầu xin mua sản phẩm này.")
	ErrBuyerSelfRequest         = apperror.NewBadRequest("Bạn không thể tự gửi yêu cầu mua sản phẩm của chính mình.")
)

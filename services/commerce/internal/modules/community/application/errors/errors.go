package communityerrors

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/utils/numberutils"
)

func ErrPostNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy bài đăng.")
}

func ErrNotPostOwner() *apperror.Error {
	return apperror.NewForbidden("Bạn không có quyền thao tác bài đăng này.")
}

func ErrOutfitInvalid() *apperror.Error {
	return apperror.NewBadRequest("Outfit không hợp lệ hoặc không thuộc sở hữu của bạn.")
}

func ErrInvalidPostType() *apperror.Error {
	return apperror.NewBadRequest("Loại bài đăng không hợp lệ.")
}

func ErrEmptyPostBody() *apperror.Error {
	return apperror.NewBadRequest("Vui lòng nhập nội dung hoặc đính kèm media.")
}

func ErrCannotChangePostType() *apperror.Error {
	return apperror.NewBadRequest("Không thể thay đổi loại bài đăng.")
}

func ErrMediaLimitExceeded(limit int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Số lượng media tối đa là %d.", limit))
}

func ErrInvalidMedia() *apperror.Error {
	return apperror.NewBadRequest("Media không hợp lệ.")
}

func ErrMediaExtensionNotAllowed(mediaType, ext string, allowed []string) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf(
		"Định dạng file %s không được phép cho loại media %s. Chỉ chấp nhận: %s.",
		ext, mediaType, strings.Join(allowed, ", ")))
}

func ErrMediaTooLarge(mediaType string, limitBytes int64) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf(
		"Dung lượng media %s không được vượt quá %s.",
		mediaType, numberutils.FormatBytes(limitBytes)))
}

func ErrVideoDurationTooLong(limitSeconds int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf(
		"Thời lượng video không được vượt quá %d giây.", limitSeconds))
}

func ErrInvalidResourceType() *apperror.Error {
	return apperror.NewBadRequest("Loại media không hợp lệ.")
}

func ErrInvalidMediaOrder() *apperror.Error {
	return apperror.NewBadRequest("Thứ tự media không hợp lệ hoặc bị trùng.")
}

func ErrTitleTooLong(limit int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Tiêu đề không được vượt quá %d ký tự.", limit))
}

func ErrContentTooLong(limit int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Nội dung không được vượt quá %d ký tự.", limit))
}

func ErrCommentTooLong(limit int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Bình luận không được vượt quá %d ký tự.", limit))
}

func ErrCommentNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy bình luận.")
}

func ErrNotCommentOwner() *apperror.Error {
	return apperror.NewForbidden("Bạn không có quyền thao tác bình luận này.")
}

func ErrParentCommentInvalid() *apperror.Error {
	return apperror.NewBadRequest("Bình luận được phản hồi không hợp lệ.")
}

func ErrSelfFollow() *apperror.Error {
	return apperror.NewBadRequest("Không thể theo dõi chính mình.")
}

func ErrUserNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy người dùng.")
}

func ErrInvalidFollowType() *apperror.Error {
	return apperror.NewBadRequest("Loại danh sách theo dõi không hợp lệ.")
}

func ErrCannotRestoreAuthorDeleted() *apperror.Error {
	return apperror.NewBadRequest("Không thể khôi phục nội dung do người dùng đã tự xóa.")
}

func ErrRateLimited() *apperror.Error {
	return apperror.NewTooManyRequest("Bạn thao tác quá nhanh. Vui lòng thử lại sau.")
}

package fashioerrors

import (
	"smart-wardrobe-be/shared/application/constants/apperror"
)

func ErrCategoryNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy danh mục.")
}

func ErrCategoryNameAlreadyExists() *apperror.Error {
	return apperror.NewConflict("Tên danh mục đã tồn tại.")
}

func ErrCategorySlugAlreadyExists() *apperror.Error {
	return apperror.NewConflict("Slug danh mục đã tồn tại.")
}

func ErrCategoryLegacySlugForbidden() *apperror.Error {
	return apperror.NewBadRequest("Slug 'vay' đã lỗi thời, không được sử dụng.")
}

func ErrCategoryOtherImmutable() *apperror.Error {
	return apperror.NewBadRequest("Danh mục 'other' không thể xóa.")
}

func ErrCategoryHasUserItems() *apperror.Error {
	return apperror.NewBadRequest("Không thể xóa danh mục đang có trang phục người dùng.")
}

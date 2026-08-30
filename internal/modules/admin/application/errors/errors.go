package adminerrors

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrUserNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy người dùng.")
}

func ErrBrandMemberNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thành viên nhãn hàng.")
}

func ErrCannotRemoveLastOwner() *apperror.Error {
	return apperror.NewBadRequest("Không thể gỡ thành viên cuối cùng là chủ sở hữu nhãn hàng.")
}

func ErrCannotDemoteLastOwner() *apperror.Error {
	return apperror.NewBadRequest("Không thể hạ cấp chủ sở hữu cuối cùng của nhãn hàng.")
}

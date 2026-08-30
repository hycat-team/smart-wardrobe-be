package adminerrors

import (
	"fmt"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrAuditLogNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy nhật ký quản trị.")
}

func ErrInvalidAuditFilter() *apperror.Error {
	return apperror.NewBadRequest("Bộ lọc nhật ký quản trị không hợp lệ.")
}

func ErrUserNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy người dùng.")
}

func ErrSubscriptionNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy đăng ký gói cước.")
}

func ErrPlanNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói cước.")
}

func ErrPlanSlugExists(slug string) *apperror.Error {
	return apperror.NewConflict(fmt.Sprintf("Gói cước với mã '%s' đã tồn tại.", slug))
}

func ErrInvalidSubscriptionStatus() *apperror.Error {
	return apperror.NewBadRequest("Trạng thái đăng ký không hợp lệ cho thao tác này.")
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

func ErrAIModelConfigNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cấu hình mô hình AI.")
}

func ErrInvalidAIGrant() *apperror.Error {
	return apperror.NewBadRequest("Grant AI không hợp lệ.")
}

package fashioerrors

import (
	"fmt"

	"smart-wardrobe-be/shared/application/constants/apperror"
)

// ErrPersonalizationNotAvailable refuses a profile write for a user whose
// subscription plan does not grant personalized styling (FR-003). Reading the
// profile stays allowed (FR-004).
func ErrPersonalizationNotAvailable() *apperror.Error {
	return apperror.NewForbidden("Cá nhân hoá phong cách là quyền lợi của gói trả phí. Vui lòng nâng cấp gói để sử dụng tính năng này.")
}

// ErrPersonalizationDisabled is the kill switch: the operator turned
// personalized styling off, so profile writes are refused even for entitled
// users. Reads stay available for operational checks.
func ErrPersonalizationDisabled() *apperror.Error {
	return apperror.NewForbidden("Tính năng cá nhân hoá phong cách đang tạm tắt. Vui lòng thử lại sau.")
}

// ErrStyleLimitExceeded refuses a list that carries more labels than the
// configured per-list cap (FR-005).
func ErrStyleLimitExceeded(maxStyles int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Mỗi danh sách phong cách chỉ được tối đa %d giá trị.", maxStyles))
}

// ErrStyleValueInvalid refuses a single label that is longer than the allowed
// length. Empty values are dropped during normalization and never reach this
// error (FR-008).
func ErrStyleValueInvalid(maxLength int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Mỗi phong cách chỉ được dài tối đa %d ký tự.", maxLength))
}

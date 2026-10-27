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

// ErrRecommendationSnapshotNotFound refuses feedback whose recommendation
// snapshot no longer exists: the 7-day TTL expired, the key was evicted, or the
// id was never issued by this server (contracts §2.2, §7, plan D8).
func ErrRecommendationSnapshotNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thông tin gợi ý. Gợi ý có thể đã hết hạn, hãy tạo gợi ý mới rồi thử lại.")
}

// ErrFeedbackForbidden refuses feedback for a recommendation snapshot that
// belongs to another user (contracts §2.2, §7).
func ErrFeedbackForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không có quyền phản hồi gợi ý này.")
}

// ErrFeedbackActionInvalid refuses an action outside the like|dislike|saved
// vocabulary. The HTTP binding already enforces it; the use case keeps the
// guard so a direct caller cannot pass an action with no learning weight.
func ErrFeedbackActionInvalid() *apperror.Error {
	return apperror.NewBadRequest("Hành động phản hồi không hợp lệ. Chỉ chấp nhận like, dislike hoặc saved.")
}

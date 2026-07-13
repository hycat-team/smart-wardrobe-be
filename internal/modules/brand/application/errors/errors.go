package branderrors

import (
	"fmt"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrBrandNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy brand.")
}

func ErrBrandSlugExists() *apperror.Error {
	return apperror.NewConflict("Slug brand đã tồn tại.")
}

func ErrBrandPortalForbidden() *apperror.Error {
	return apperror.NewForbidden("Bạn không có quyền quản trị brand này.")
}

func ErrBrandNotActive() *apperror.Error {
	return apperror.NewForbidden("Brand chưa active hoặc đã bị khóa.")
}

func ErrInvalidBrandStatus(status any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trạng thái brand không hợp lệ: %v.", status))
}

func ErrInvalidBrandMemberRole(role any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Vai trò brand member không hợp lệ: %v.", role))
}

func ErrPhoneRequired() *apperror.Error {
	return apperror.NewBadRequest("Số điện thoại offline customer không được để trống.")
}

func ErrCustomerIdentifierRequired() *apperror.Error {
	return apperror.NewBadRequest("Cần có userId, phone hoặc externalCustomerCode để xác định khách hàng.")
}

func ErrPurchaseAmountOrPointsRequired() *apperror.Error {
	return apperror.NewBadRequest("Cần có purchaseAmount hoặc pointsDelta.")
}

func ErrInvalidLoyaltyTransactionType() *apperror.Error {
	return apperror.NewBadRequest("Loại giao dịch loyalty không được hỗ trợ qua endpoint này.")
}

func ErrPointsDeltaZero() *apperror.Error {
	return apperror.NewBadRequest("pointsDelta không được bằng 0.")
}

func ErrActiveLoyaltyProgramRequired() *apperror.Error {
	return apperror.NewNotFound("Brand chưa có loyalty program active.")
}

func ErrInsufficientLoyaltyPoints() *apperror.Error {
	return apperror.NewBadRequest("Số dư điểm loyalty không đủ để thực hiện giao dịch.")
}

func ErrUserNotFoundOrInactive() *apperror.Error {
	return apperror.NewNotFound("User không tồn tại hoặc không active.")
}

func ErrBenefitNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy quyền lợi.")
}

func ErrBenefitNotActive() *apperror.Error {
	return apperror.NewForbidden("Quyền lợi không active hoặc đã bị ẩn.")
}

func ErrBenefitUnlockTypeNotSupported() *apperror.Error {
	return apperror.NewBadRequest("Loại mở khóa quyền lợi không được hỗ trợ.")
}

func ErrBenefitRequiredPointsNotMet() *apperror.Error {
	return apperror.NewBadRequest("Quyền lợi yêu cầu số điểm cao hơn số dư hiện tại.")
}

func ErrBenefitRequiredTierNotMet() *apperror.Error {
	return apperror.NewBadRequest("Tier của bạn không đạt yêu cầu để nhận quyền lợi này.")
}

func ErrBenefitRedemptionExists() *apperror.Error {
	return apperror.NewConflict("Bạn đã đổi quyền lợi này và lượt đổi vẫn đang còn hạn.")
}

func ErrBenefitInvalidStatus() *apperror.Error {
	return apperror.NewBadRequest("Trạng thái quyền lợi không hợp lệ.")
}

func ErrInvalidBenefitFeatureCode(featureCode any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Mã tính năng quyền lợi không hợp lệ: %v.", featureCode))
}

func ErrBenefitFeatureCodeRequired() *apperror.Error {
	return apperror.NewBadRequest("Quyền lợi feature_access bắt buộc phải có featureCode hợp lệ.")
}

func ErrInvalidVoteType(voteType any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Loại voteType không hợp lệ: %v.", voteType))
}

func ErrCustomerNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy khách hàng của brand.")
}

func ErrCustomerAlreadyLinked() *apperror.Error {
	return apperror.NewConflict("Khách hàng này đã được liên kết với một tài khoản người dùng.")
}

func ErrInvalidToken() *apperror.Error {
	return apperror.NewBadRequest("Mã claim không hợp lệ.")
}

func ErrTokenAlreadyUsed() *apperror.Error {
	return apperror.NewConflict("Mã claim đã được sử dụng.")
}

func ErrTokenExpired() *apperror.Error {
	return apperror.NewBadRequest("Mã claim đã hết hạn sử dụng.")
}

func ErrUserAlreadyHasCustomer() *apperror.Error {
	return apperror.NewConflict("Tài khoản của bạn đã được liên kết với một hồ sơ khách hàng tại brand này.")
}

func ErrTokenRevoked() *apperror.Error {
	return apperror.NewBadRequest("Mã claim đã bị thu hồi.")
}

func ErrClaimRateLimited() *apperror.Error {
	return apperror.NewTooManyRequest("Bạn đã thử claim quá nhiều lần. Vui lòng thử lại sau.")
}

func ErrClaimRateLimitUnavailable() *apperror.Error {
	return apperror.NewServiceUnavailable("Chưa thể kiểm tra giới hạn thử claim. Vui lòng thử lại sau.")
}

func ErrProductCodeExists() *apperror.Error {
	return apperror.NewConflict("Mã sản phẩm đã tồn tại trong thương hiệu này.")
}

func ErrTierNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy hạng thành viên này.")
}

func ErrTierNameExists(name string) *apperror.Error {
	return apperror.NewConflict(fmt.Sprintf("Hạng thành viên \"%s\" đã tồn tại trong thương hiệu này.", name))
}

func ErrTierRankExists(rank int) *apperror.Error {
	return apperror.NewConflict(fmt.Sprintf("Hạng thành viên thứ %d đã tồn tại trong thương hiệu này.", rank))
}

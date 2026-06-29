package branderrors

import (
	"fmt"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func ErrBrandNotFound() *apperror.Error {
	return apperror.NewNotFound("Khong tim thay brand.")
}

func ErrBrandSlugExists() *apperror.Error {
	return apperror.NewConflict("Slug brand da ton tai.")
}

func ErrBrandPortalForbidden() *apperror.Error {
	return apperror.NewForbidden("Ban khong co quyen quan tri brand nay.")
}

func ErrBrandNotActive() *apperror.Error {
	return apperror.NewForbidden("Brand chua active hoac da bi khoa.")
}

func ErrInvalidBrandStatus(status any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Trang thai brand khong hop le: %v.", status))
}

func ErrInvalidBrandMemberRole(role any) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Vai tro brand member khong hop le: %v.", role))
}

func ErrPhoneRequired() *apperror.Error {
	return apperror.NewBadRequest("So dien thoai offline customer khong duoc de trong.")
}

func ErrCustomerIdentifierRequired() *apperror.Error {
	return apperror.NewBadRequest("Can co userId, phone hoac externalCustomerCode de xac dinh khach hang.")
}

func ErrPurchaseAmountOrPointsRequired() *apperror.Error {
	return apperror.NewBadRequest("Can co purchaseAmount hoac pointsDelta.")
}

func ErrInvalidLoyaltyTransactionType() *apperror.Error {
	return apperror.NewBadRequest("Loai giao dich loyalty khong duoc ho tro qua endpoint nay.")
}

func ErrPointsDeltaZero() *apperror.Error {
	return apperror.NewBadRequest("pointsDelta khong duoc bang 0.")
}

func ErrActiveLoyaltyProgramRequired() *apperror.Error {
	return apperror.NewBadRequest("Brand chua co loyalty program active.")
}

func ErrInsufficientLoyaltyPoints() *apperror.Error {
	return apperror.NewBadRequest("So du diem loyalty khong du de thuc hien giao dich.")
}

func ErrUserNotFoundOrInactive() *apperror.Error {
	return apperror.NewBadRequest("User khong ton tai hoac khong active.")
}

func ErrBenefitNotFound() *apperror.Error {
	return apperror.NewNotFound("Khong tim thay quyen loi.")
}

func ErrBenefitNotActive() *apperror.Error {
	return apperror.NewForbidden("Quyen loi khong active hoac da bi an.")
}

func ErrBenefitUnlockTypeNotSupported() *apperror.Error {
	return apperror.NewBadRequest("Loai mo khoa quyen loi khong duoc ho tro.")
}

func ErrBenefitRequiredPointsNotMet() *apperror.Error {
	return apperror.NewBadRequest("Quyen loi yeu cau so diem cao hon so du hien tai.")
}

func ErrBenefitRequiredTierNotMet() *apperror.Error {
	return apperror.NewBadRequest("Tier cua ban khong dat yeu cau de nhan quyen loi nay.")
}

func ErrBenefitRedemptionExists() *apperror.Error {
	return apperror.NewConflict("Ban da doi quyen loi nay va luot doi van dang con han.")
}

func ErrBenefitInvalidStatus() *apperror.Error {
	return apperror.NewBadRequest("Trang thai quyen loi khong hop le.")
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
	return apperror.NewError(503, "Dịch vụ tạm thời gián đoạn", "Chưa thể kiểm tra giới hạn thử claim. Vui lòng thử lại sau.")
}

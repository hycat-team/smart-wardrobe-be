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

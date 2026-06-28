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

package dto

import (
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
)

type UpdateProfileReq struct {
	FirstName   string         `json:"firstName" binding:"required" label:"tên"`
	LastName    *string        `json:"lastName" binding:"omitempty" label:"họ"`
	DateOfBirth string         `json:"dateOfBirth" binding:"omitempty,datetime=2006-01-02" label:"ngày sinh"`
	Gender      *gender.Gender `json:"gender" binding:"omitempty,oneof=0 1 2 3" label:"giới tính"`
	Address     string         `json:"address" binding:"omitempty" label:"địa chỉ"`
}

type UpdateUserStatusReq struct {
	Status userstatus.UserStatus `json:"status" binding:"oneof=0 1" label:"trạng thái tài khoản"`
}

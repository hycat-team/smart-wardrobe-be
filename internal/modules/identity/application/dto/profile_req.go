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
	Address     *string        `json:"address" binding:"omitempty" label:"địa chỉ"`
}

type UpdateBodyProfileReq struct {
	HeightCM       float64                    `json:"heightCm" binding:"required,gte=0" label:"chiều cao"`
	WeightKG       float64                    `json:"weightKg" binding:"required,gte=0" label:"cân nặng"`
	BodyShape      string                     `json:"bodyShape" binding:"required" label:"dáng người"`
	Measurements   *UpdateBodyMeasurementsReq `json:"measurements,omitempty"`
	InferredByAI   *InferredBodyProfileReq    `json:"inferredByAi,omitempty"`
	VerifiedByUser bool                       `json:"verifiedByUser"`
}

type UpdateBodyMeasurementsReq struct {
	ChestCM float64 `json:"chestCm" binding:"omitempty,gte=0" label:"vòng ngực"`
	WaistCM float64 `json:"waistCm" binding:"omitempty,gte=0" label:"vòng eo"`
	HipCM   float64 `json:"hipCm" binding:"omitempty,gte=0" label:"vòng hông"`
}

type InferredBodyProfileReq struct {
	BodyShape       string   `json:"bodyShape" binding:"required" label:"dáng người AI suy luận"`
	ConfidenceScore *float64 `json:"confidenceScore,omitempty" binding:"omitempty,gte=0,lte=1" label:"độ tự tin AI"`
}

type UpdateUserStatusReq struct {
	Status userstatus.UserStatus `json:"status" binding:"oneof=0 1" label:"trạng thái tài khoản"`
}

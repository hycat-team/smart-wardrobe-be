package dto

import (
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
)

type UpdateProfileReq struct {
	FirstName   string         `json:"firstName" binding:"required"`
	LastName    *string        `json:"lastName" binding:"omitempty"`
	DateOfBirth string         `json:"dateOfBirth" binding:"omitempty,datetime=2006-01-02"`
	Gender      *gender.Gender `json:"gender" binding:"omitempty,oneof=0 1 2 3"`
	Address     string         `json:"address" binding:"omitempty"`
}

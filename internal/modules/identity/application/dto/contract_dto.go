package dto

import (
	"smart-wardrobe-be/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

type UserBriefContractDTO struct {
	ID        uuid.UUID             `json:"id"`
	Username  string                `json:"username"`
	Email     string                `json:"email,omitempty"`
	FirstName string                `json:"firstName"`
	LastName  string                `json:"lastName,omitempty"`
	Gender    gender.Gender         `json:"gender"`
	Status    userstatus.UserStatus `json:"status"`
	AvatarUrl *string               `json:"avatarUrl,omitempty"`
}

type UserBriefRes = UserBriefContractDTO

package dto

import (
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"time"

	"github.com/google/uuid"
)

type TokenRes struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type UserRes struct {
	ID        uuid.UUID     `json:"id"`
	Username  string        `json:"username"`
	Email     string        `json:"email"`
	RoleSlug  string        `json:"roleSlug"`
	FirstName string        `json:"firstName"`
	LastName  string        `json:"lastName,omitempty"`
	Address   string        `json:"address,omitempty"`
	Gender    gender.Gender `json:"gender"`
	CreatedAt time.Time     `json:"createdAt"`
}

package contract

import (
	"time"

	"github.com/google/uuid"
)

type UserRes struct {
	ID             uuid.UUID  `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	FirstName      *string    `json:"firstName,omitempty"`
	LastName       *string    `json:"lastName,omitempty"`
	DateOfBirth    *time.Time `json:"dateOfBirth,omitempty"`
	Address        *string    `json:"address,omitempty"`
	Gender         *int       `json:"gender,omitempty"`
	RoleSlug       string     `json:"roleSlug"`
	Status         int16      `json:"status"`
	AvatarUrl      *string    `json:"avatarUrl,omitempty"`
	AvatarPublicID *string    `json:"avatarPublicId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type UserStyleProfileRes struct {
	UserID          uuid.UUID `json:"userId"`
	TasteEmbedding  []float32 `json:"tasteEmbedding,omitempty"`
	PreferredColors []string  `json:"preferredColors,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

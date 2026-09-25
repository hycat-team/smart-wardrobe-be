package entities

import (
	"time"

	"github.com/google/uuid"
)

type PostLike struct {
	PostID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

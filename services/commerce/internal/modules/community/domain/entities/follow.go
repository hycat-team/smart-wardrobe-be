package entities

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	FollowerID uuid.UUID `gorm:"type:uuid;primaryKey"`
	FolloweeID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

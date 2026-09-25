package entities

import (
	"time"

	"github.com/google/uuid"
)

type UserFollowStats struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	FollowerCount  int       `gorm:"type:integer;not null;default:0"`
	FollowingCount int       `gorm:"type:integer;not null;default:0"`
	UpdatedAt      time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

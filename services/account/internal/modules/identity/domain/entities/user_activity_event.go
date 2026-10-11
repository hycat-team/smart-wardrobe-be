package entities

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/useractivitytype"

	"github.com/google/uuid"
)

// UserActivityEvent is one row of the append-only activity log feeding the
// admin dashboard DAU. It is deliberately NOT an audited aggregate: DAU needs
// distinct users per day, which a raw event log answers directly and a daily
// rollup would have to maintain.
type UserActivityEvent struct {
	ID           uuid.UUID                         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID                         `gorm:"type:uuid;not null"`
	ActivityType useractivitytype.UserActivityType `gorm:"type:varchar(20);not null"`
	OccurredAt   time.Time                         `gorm:"type:timestamp with time zone;not null;default:now()"`
	CreatedAt    time.Time                         `gorm:"type:timestamp with time zone;not null;default:now()"`
}

func (UserActivityEvent) TableName() string {
	return "user_activity_events"
}

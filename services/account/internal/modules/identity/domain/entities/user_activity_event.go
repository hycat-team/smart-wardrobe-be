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

	// 029 D-07: copied from the session this event belongs to, so an admin can
	// answer "which device/IP did this login come from" without a timestamp join.
	// Pointers because the columns are nullable: rows written before the migration
	// have no source, and "not recorded" must stay distinguishable from "empty".
	DeviceName *string `gorm:"type:varchar(255)"`
	DeviceIP   *string `gorm:"type:varchar(45)"`
	UserAgent  *string `gorm:"type:varchar(512)"`
}

func (UserActivityEvent) TableName() string {
	return "user_activity_events"
}

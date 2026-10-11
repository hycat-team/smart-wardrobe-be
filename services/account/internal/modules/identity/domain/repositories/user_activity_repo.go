package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
)

// IUserActivityRepository appends activity events and answers the DAU query.
// The table is the raw source for admin dashboard statistics; consumers beyond
// stats are not expected, so the interface stays deliberately small.
type IUserActivityRepository interface {
	// Record appends one activity event.
	Record(ctx context.Context, event *entities.UserActivityEvent) error
	// CountDistinctUsers counts the users with at least one activity event in
	// [from, to).
	CountDistinctUsers(ctx context.Context, from, to time.Time) (int64, error)
}

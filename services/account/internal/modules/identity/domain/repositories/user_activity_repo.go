package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/useractivitytype"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"

	"github.com/google/uuid"
)

// AdminActivityEventFilter narrows the admin activity list (029 R3). Every
// field is optional: nil/zero drops its condition (open ended).
type AdminActivityEventFilter struct {
	UserID *uuid.UUID
	// ActivityType only ever holds one of the two existing constants
	// (login/refresh); the use case refuses anything else before this point.
	ActivityType *useractivitytype.UserActivityType
	// From is inclusive, To is exclusive — the half-open VN range of R1. A zero
	// value leaves that side open.
	From time.Time
	To   time.Time
}

// IUserActivityRepository appends activity events, answers the DAU query, and
// serves the admin activity list.
type IUserActivityRepository interface {
	// Record appends one activity event.
	Record(ctx context.Context, event *entities.UserActivityEvent) error
	// CountDistinctUsers counts the users with at least one activity event in
	// [from, to).
	CountDistinctUsers(ctx context.Context, from, to time.Time) (int64, error)
	// ListAdmin returns one page of activity events, newest first
	// (occurred_at DESC, id DESC) and the total matching the SAME filter set.
	// Identity comes from a LEFT JOIN on users, so events of soft-deleted
	// accounts stay visible.
	ListAdmin(ctx context.Context, filter AdminActivityEventFilter, page, limit int) ([]AdminActivityEventRow, int64, error)
}

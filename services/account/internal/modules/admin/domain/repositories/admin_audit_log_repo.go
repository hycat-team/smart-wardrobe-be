package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/admin/domain/entities"

	"github.com/google/uuid"
)

type AuditLogFilter struct {
	ActorUserID *uuid.UUID
	Action      *string
	TargetType  *string
	TargetID    *uuid.UUID
	FromDate    *time.Time
	ToDate      *time.Time
}

type IAdminAuditLogRepository interface {
	Create(ctx context.Context, log *entities.AdminAuditLog) error
	List(ctx context.Context, filter AuditLogFilter, page, limit int) ([]*entities.AdminAuditLog, int64, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
	// ExistsRecentChange implements the check-then-insert half of the gate-2
	// de-duplication rule (024 FR-004e). shared.admin_audit_logs has no natural
	// unique constraint for "one row per configuration change", and a unique
	// index is not an option either: two legitimate edits can carry the same
	// field name and the same new value at different times, and a unique index
	// would silently swallow the second one.
	ExistsRecentChange(ctx context.Context, query RecentChangeQuery) (bool, error)
}

// RecentChangeQuery is the stable identity of one gate-2 configuration change:
// who changed it, which policy/operation/field, to which value, and when.
// A nil NewValue is a legitimate value — it means the field was set to NULL —
// so the query has to distinguish "cleared" from "absent", which is why it takes
// pointers rather than a single "isNewValueSet" flag.
type RecentChangeQuery struct {
	ActorUserID uuid.UUID
	Action      string
	TargetType  string
	PolicyCode  string
	Operation   string
	FieldName   string
	NewValue    *string
	Since       time.Time
}

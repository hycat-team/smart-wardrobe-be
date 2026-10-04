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
	// ListByCampaignCode returns one page of the trail of EXACTLY ONE campaign
	// (026 T065, FR-063).
	//
	// IT IS NOT `List` WITH A FILTER, and that is the reason it exists. `List` can only
	// select on `actor_user_id`, `action`, `target_type`, `target_id` and a date range,
	// and `target_id` is a `*uuid.UUID` while a campaign is keyed by a text code. The
	// code therefore lives in `payload.campaignCode` (FR-063), and reaching it needs the
	// JSONB `->>` operator — a predicate the typed filter has no field for.
	//
	// Adding a `CampaignCode` to `AuditLogFilter` instead would put a campaign-specific
	// question on the general audit filter, which the whole administration screen also
	// uses; one filter serving two audiences is how a screen ends up with a parameter
	// that silently does nothing.
	//
	// `page < 1` and `limit <= 0` are CLAMPED rather than refused, exactly as `List`
	// does — "give me more than the maximum" is answered with the maximum plus a total,
	// not with an error.
	ListByCampaignCode(ctx context.Context, campaignCode string, page, limit int) ([]*entities.AdminAuditLog, int64, error)
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

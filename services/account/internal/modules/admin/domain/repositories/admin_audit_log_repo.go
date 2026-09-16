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
}

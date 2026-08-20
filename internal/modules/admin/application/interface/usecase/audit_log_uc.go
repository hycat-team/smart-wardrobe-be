package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/admin/application/dto"
)

type IAuditLogUseCase interface {
	Record(ctx context.Context, input dto.RecordAuditLogInput) error
	List(ctx context.Context, filter dto.AuditLogFilter, page, limit int) ([]*dto.AdminAuditLogDTO, int64, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

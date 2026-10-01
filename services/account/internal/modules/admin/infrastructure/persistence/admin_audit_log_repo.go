package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/admin/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminAuditLogRepository struct {
	*shared_repos.GenericRepository[entities.AdminAuditLog, uuid.UUID]
}

func NewAdminAuditLogRepository(db *gorm.DB) repositories.IAdminAuditLogRepository {
	return &AdminAuditLogRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.AdminAuditLog, uuid.UUID](db, nil),
	}
}

func (r *AdminAuditLogRepository) Create(ctx context.Context, log *entities.AdminAuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return r.GetDB(ctx).Create(log).Error
}

func (r *AdminAuditLogRepository) List(ctx context.Context, filter repositories.AuditLogFilter, page, limit int) ([]*entities.AdminAuditLog, int64, error) {
	db := r.GetDB(ctx).Model(&entities.AdminAuditLog{})

	if filter.ActorUserID != nil {
		db = db.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.Action != nil && *filter.Action != "" {
		db = db.Where("action = ?", *filter.Action)
	}
	if filter.TargetType != nil && *filter.TargetType != "" {
		db = db.Where("target_type = ?", *filter.TargetType)
	}
	if filter.TargetID != nil {
		db = db.Where("target_id = ?", *filter.TargetID)
	}
	if filter.FromDate != nil && !filter.FromDate.IsZero() {
		db = db.Where("created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil && !filter.ToDate.IsZero() {
		db = db.Where("created_at <= ?", *filter.ToDate)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	var logs []*entities.AdminAuditLog
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *AdminAuditLogRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res := r.GetDB(ctx).Where("created_at < ?", cutoff).Delete(&entities.AdminAuditLog{})
	return res.RowsAffected, res.Error
}

// ExistsRecentChange looks for a previously recorded identical gate-2 change.
//
// The four payload fields are compared with jsonb containment so the check
// works on the stored document without adding columns. NewValue is handled
// separately from the containment test because a JSON null must match a
// previously cleared field rather than matching everything.
func (r *AdminAuditLogRepository) ExistsRecentChange(ctx context.Context, query repositories.RecentChangeQuery) (bool, error) {
	var count int64

	db := r.GetDB(ctx).
		Model(&entities.AdminAuditLog{}).
		Where("actor_user_id = ?", query.ActorUserID).
		Where("action = ?", query.Action).
		Where("target_type = ?", query.TargetType).
		Where("created_at >= ?", query.Since).
		Where("payload @> ?::jsonb", `{"policyCode":"`+query.PolicyCode+`","operation":"`+query.Operation+`","fieldName":"`+query.FieldName+`"}`)

	if query.NewValue == nil {
		db = db.Where("(payload->>'newValue') IS NULL")
	} else {
		db = db.Where("payload->>'newValue' = ?", *query.NewValue)
	}

	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/admin/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
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

	page, limit = normalizeAuditPagination(page, limit)
	offset := (page - 1) * limit

	var logs []*entities.AdminAuditLog
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByCampaignCode returns one page of the trail of EXACTLY ONE campaign (026 T065,
// FR-063).
//
// THE TWO TERMS ARE BOTH REQUIRED, and dropping either one is a bug that looks correct:
//
//   - `target_type = 'campaign'` keeps every other audit row out. Without it the page
//     would include the rows for whatever else happens to carry the same code in its
//     payload, and the operator would be shown somebody else's change under this
//     campaign;
//   - `payload->>'campaignCode' = ?` is the selector, because FR-063 stores the code there
//     and leaves `target_id` NULL. The `->>` operator returns TEXT rather than JSON, so
//     the comparison is a plain string equality — the same expression
//     `ExistsRecentChange` already uses for `payload->>'newValue'`, which is why the
//     column is declared `jsonb` rather than `text`.
//
// THE PARAMETER IS BOUND, NOT INTERPOLATED. `?` with a value is what keeps a code
// containing a quote from changing the query's meaning; the payload is data, not SQL.
//
// ORDERED BY `created_at DESC, id DESC`. The trailing key is not decoration: a campaign's
// create and its first edit routinely land inside one millisecond, and without a second
// key PostgreSQL may return those two rows in either order between two identical calls —
// so a screen would show the edit above the create it followed. `id` is unique and
// monotonic, which makes the order total.
func (r *AdminAuditLogRepository) ListByCampaignCode(
	ctx context.Context,
	campaignCode string,
	page, limit int,
) ([]*entities.AdminAuditLog, int64, error) {
	db := r.GetDB(ctx).
		Model(&entities.AdminAuditLog{}).
		Where("target_type = ?", string(adminaudit.Campaign)).
		Where("payload->>'campaignCode' = ?", campaignCode)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit = normalizeAuditPagination(page, limit)

	var logs []*entities.AdminAuditLog
	err := db.Order("created_at DESC, id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// normalizeAuditPagination clamps a caller-supplied page and limit.
//
// IT IS EXTRACTED rather than copied into both listing methods, because the two must
// agree: an operator paging through the campaign trail and an administrator paging
// through the general audit screen would otherwise be able to reach different pages with
// the same `page=0&limit=0`, and the clamp is the kind of rule that has to be one rule.
func normalizeAuditPagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	return page, limit
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

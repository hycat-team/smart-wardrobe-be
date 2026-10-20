package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserActivityRepository struct {
	shared_persist.GenericRepository[entities.UserActivityEvent, uuid.UUID]
	db *gorm.DB
}

func NewUserActivityRepository(dbConn *gorm.DB) repositories.IUserActivityRepository {
	relations := []string{}
	return &UserActivityRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.UserActivityEvent, uuid.UUID](dbConn, relations),
		db:                dbConn,
	}
}

func (r *UserActivityRepository) current(ctx context.Context) *gorm.DB {
	if tx := shareddb.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

func (r *UserActivityRepository) Record(ctx context.Context, event *entities.UserActivityEvent) error {
	return r.current(ctx).Create(event).Error
}

// ListAdmin returns one page of activity events with the account identity,
// newest first, plus the total matching the same filters (029 R3,
// data-model §1).
//
// The LEFT JOIN is deliberate: soft delete keeps the users row, and the event
// must not disappear when its account is deactivated; only a hard delete
// cascades the events away. The COUNT carries the same WHERE set (built once
// by applyAdminActivityFilter) but no join — identity is not part of the
// filter, so joining for the count would only add work.
//
// Table names stay unqualified, following the identity module search_path
// convention.
func (r *UserActivityRepository) ListAdmin(
	ctx context.Context,
	filter repositories.AdminActivityEventFilter,
	page, limit int,
) ([]repositories.AdminActivityEventRow, int64, error) {
	db := applyAdminActivityFilter(r.current(ctx).Table("user_activity_events AS e"), filter)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []repositories.AdminActivityEventRow
	query := db.
		Select("e.id, e.user_id, e.activity_type, e.occurred_at, e.created_at, " +
			"e.device_name, e.device_ip, e.user_agent, " +
			"u.email, u.first_name, u.last_name").
		Joins("LEFT JOIN users u ON u.id = e.user_id").
		Order("e.occurred_at DESC, e.id DESC")
	query = shared_persist.ApplyPagination(query, shared_dto.PaginationQuery{Page: page, Limit: limit})

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// applyAdminActivityFilter adds the optional conditions shared by the COUNT
// and the page query. `e` is the user_activity_events alias both queries use,
// so the two can never drift.
func applyAdminActivityFilter(db *gorm.DB, filter repositories.AdminActivityEventFilter) *gorm.DB {
	if filter.UserID != nil {
		db = db.Where("e.user_id = ?", *filter.UserID)
	}
	if filter.ActivityType != nil {
		db = db.Where("e.activity_type = ?", *filter.ActivityType)
	}
	if !filter.From.IsZero() {
		db = db.Where("e.occurred_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		db = db.Where("e.occurred_at < ?", filter.To)
	}
	return db
}

func (r *UserActivityRepository) CountDistinctUsers(ctx context.Context, from, to time.Time) (int64, error) {
	var count int64
	// Joined against users so activity from soft-deleted accounts does not
	// inflate DAU: a deleted account is not an active user. The unqualified
	// `users` follows the same search_path convention as the rest of the
	// identity repositories.
	err := r.current(ctx).
		Model(&entities.UserActivityEvent{}).
		Where("occurred_at >= ? AND occurred_at < ?", from, to).
		Where("EXISTS (SELECT 1 FROM users u WHERE u.id = user_activity_events.user_id AND u.is_deleted = ?)", false).
		Distinct("user_id").
		Count(&count).Error
	return count, err
}

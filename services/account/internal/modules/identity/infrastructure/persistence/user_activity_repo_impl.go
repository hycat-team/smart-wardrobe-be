package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
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

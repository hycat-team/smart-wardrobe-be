package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/infrastructure/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserSubscriptionEventRepository struct{ db *gorm.DB }

func NewUserSubscriptionEventRepository(conn *gorm.DB) repositories.IUserSubscriptionEventRepository {
	return &UserSubscriptionEventRepository{db: conn}
}

func (r *UserSubscriptionEventRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

// Create writes the history row and nothing else.
//
// 024 T071 / FR-037, FR-038: this used to be followed by projectAIPolicyGrant,
// which inserted a row into aiexecutor.ai_policy_grants with raw SQL. That is
// the constitution §I violation this whole feature exists to remove: the Account
// service was a second writer on a table owned by the AI service, and the
// compiler could not see it because it was not a function call.
//
// It was also broken in its own right — it wrote '{}'::jsonb into a column the
// reader requires to carry a two-part snapshot, so every row it produced was
// born broken, and the failure surfaced as a database error in the middle of a
// customer's payment.
//
// The AI service now derives the grant from the event, in its own transaction,
// against its own table.
func (r *UserSubscriptionEventRepository) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	return r.current(ctx).Create(event).Error
}

// CreateIfAbsent writes the row unless `event_key` is already taken, and reports
// which of the two happened (024 T148).
//
// The insert is what serialises the race, not the pre-check that usually precedes
// it: two transactions that both read "no such key" and then both insert will have
// the second block on the first's UNIQUE index and, once the first commits, come
// back with zero rows affected. That zero is the signal the caller needs.
//
// A plain `DO NOTHING` with no conflict target is deliberately not used: naming the
// column makes the intent explicit and stops this from silently absorbing an
// unrelated constraint violation, which is the failure mode a bare ON CONFLICT DO
// NOTHING would give us.
func (r *UserSubscriptionEventRepository) CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (bool, error) {
	result := r.current(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_key"}},
		DoNothing: true,
	}).Create(event)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *UserSubscriptionEventRepository) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	var event entities.UserSubscriptionEvent
	err := r.current(ctx).Where("event_key = ?", key).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}

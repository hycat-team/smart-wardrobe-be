package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	shared_db "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StyleSignalRepository persists fashion.style_signals.
type StyleSignalRepository struct {
	db *gorm.DB
}

func NewStyleSignalRepository(db *gorm.DB) repositories.IStyleSignalRepository {
	return &StyleSignalRepository{db: db}
}

func (r *StyleSignalRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := shared_db.GetTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

// InsertIfAbsent inserts the signal once. The primary key doubles as the
// idempotency key, so an outbox redelivery of the same signal_id is dropped by
// ON CONFLICT (id) DO NOTHING. The partial unique index
// ux_style_signals_feedback is the database-level guard for feedback rows.
func (r *StyleSignalRepository) InsertIfAbsent(ctx context.Context, signal *entities.StyleSignal) (bool, error) {
	if signal == nil {
		return false, nil
	}
	if signal.ID == uuid.Nil {
		signal.ID = uuid.New()
	}
	if signal.CreatedAt.IsZero() {
		signal.CreatedAt = time.Now().UTC()
	}

	result := r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(signal)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *StyleSignalRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.getDB(ctx).
		Model(&entities.StyleSignal{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

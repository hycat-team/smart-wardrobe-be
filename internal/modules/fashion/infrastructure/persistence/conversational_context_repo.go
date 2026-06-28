package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConversationalContextRepository struct {
	shared_persist.GenericRepository[entities.ConversationalContext, uuid.UUID]
}

func NewConversationalContextRepository(db *gorm.DB) repositories.IConversationalContextRepository {
	return &ConversationalContextRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.ConversationalContext, uuid.UUID](db, nil),
	}
}

func (r *ConversationalContextRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ConversationalContext, error) {
	var items []*entities.ConversationalContext
	err := r.GetDB(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

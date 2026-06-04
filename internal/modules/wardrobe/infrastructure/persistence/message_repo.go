package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository struct {
	shared_persist.GenericRepository[entities.Message, uuid.UUID]
}

func NewMessageRepository(db *gorm.DB) repositories.IMessageRepository {
	return &MessageRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Message, uuid.UUID](db, nil),
	}
}

func (r *MessageRepository) GetByContextID(ctx context.Context, contextID uuid.UUID) ([]*entities.Message, error) {
	var items []*entities.Message
	err := r.GetDB(ctx).
		Where("context_id = ?", contextID).
		Order("created_at ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *MessageRepository) GetRecentByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error) {
	var items []*entities.Message
	err := r.GetDB(ctx).
		Where("context_id = ?", contextID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
		items[left], items[right] = items[right], items[left]
	}

	return items, nil
}

func (r *MessageRepository) GetOldestByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error) {
	var items []*entities.Message
	err := r.GetDB(ctx).
		Where("context_id = ?", contextID).
		Order("created_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *MessageRepository) DeleteByIDs(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	return r.GetDB(ctx).Where("id IN ?", ids).Delete(&entities.Message{}).Error
}

func (r *MessageRepository) CountByContextID(ctx context.Context, contextID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.Message{}).Where("context_id = ?", contextID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

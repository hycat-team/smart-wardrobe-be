package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandConversationRepository struct {
	shared_persist.GenericRepository[entities.BrandConversation, uuid.UUID]
}

func NewBrandConversationRepository(db *gorm.DB) repositories.IBrandConversationRepository {
	relations := []string{"Brand"}
	return &BrandConversationRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandConversation, uuid.UUID](db, relations),
	}
}

func (r *BrandConversationRepository) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandConversation, error) {
	var conv entities.BrandConversation
	err := r.GetDB(ctx).Where("brand_id = ? AND user_id = ?", brandID, userID).First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *BrandConversationRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandConversation, error) {
	var list []*entities.BrandConversation
	err := r.GetDB(ctx).Where("brand_id = ?", brandID).Order("last_message_at desc").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *BrandConversationRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.BrandConversation, error) {
	var conv entities.BrandConversation
	err := r.GetDB(ctx).Set("gorm:query_option", "FOR UPDATE").Where("id = ?", id).First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

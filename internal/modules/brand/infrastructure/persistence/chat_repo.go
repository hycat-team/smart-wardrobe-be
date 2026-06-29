package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandConversationRepository struct {
	shared_persist.GenericRepository[entities.BrandConversation, uuid.UUID]
}

func NewBrandConversationRepository(db *gorm.DB) repositories.IBrandConversationRepository {
	relations := []string{"Brand", "User"}
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
	err := r.GetDB(ctx).Preload("User").Where("brand_id = ?", brandID).Order("last_message_at desc").Find(&list).Error
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

type BrandConversationMessageRepository struct {
	shared_persist.GenericRepository[entities.BrandConversationMessage, uuid.UUID]
}

func NewBrandConversationMessageRepository(db *gorm.DB) repositories.IBrandConversationMessageRepository {
	relations := []string{"Conversation", "SenderUser"}
	return &BrandConversationMessageRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandConversationMessage, uuid.UUID](db, relations),
	}
}

func (r *BrandConversationMessageRepository) GetByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entities.BrandConversationMessage, error) {
	var list []*entities.BrandConversationMessage
	err := r.GetDB(ctx).Where("conversation_id = ?", conversationID).Order("created_at asc").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *BrandConversationMessageRepository) CountUnread(ctx context.Context, conversationID uuid.UUID, senderRole string, since *time.Time) (int, error) {
	query := r.GetDB(ctx).
		Model(&entities.BrandConversationMessage{}).
		Where("conversation_id = ? AND sender_role = ?", conversationID, senderRole)
	if since != nil {
		query = query.Where("created_at > ?", *since)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandConversationMessageRepository struct {
	shared_persist.GenericRepository[entities.BrandConversationMessage, uuid.UUID]
}

func NewBrandConversationMessageRepository(db *gorm.DB) repositories.IBrandConversationMessageRepository {
	relations := []string{"Conversation"}
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

func (r *BrandConversationMessageRepository) GetByConversationIDPaginated(ctx context.Context, filter repositories.BrandConversationMessageFilter) (*repositories.BrandConversationMessageListResult, error) {
	db := r.GetDB(ctx).Model(&entities.BrandConversationMessage{}).Where("conversation_id = ?", filter.ConversationID)

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	var messages []*entities.BrandConversationMessage
	if err := db.Order("created_at DESC, id DESC").Find(&messages).Error; err != nil {
		return nil, err
	}

	return &repositories.BrandConversationMessageListResult{
		Messages:   messages,
		TotalCount: totalCount,
	}, nil
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

package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandConversationMessageFilter struct {
	ConversationID uuid.UUID
	Page           int
	Limit          int
}

type BrandConversationMessageListResult struct {
	Messages   []*entities.BrandConversationMessage
	TotalCount int64
}

type IBrandConversationRepository interface {
	shared_repos.IGenericRepository[entities.BrandConversation, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandConversation, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandConversation, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.BrandConversation, error)
}

type IBrandConversationMessageRepository interface {
	shared_repos.IGenericRepository[entities.BrandConversationMessage, uuid.UUID]
	GetByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entities.BrandConversationMessage, error)
	GetByConversationIDPaginated(ctx context.Context, filter BrandConversationMessageFilter) (*BrandConversationMessageListResult, error)
	CountUnread(ctx context.Context, conversationID uuid.UUID, senderRole string, since *time.Time) (int, error)
}

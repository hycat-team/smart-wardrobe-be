package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"

	"github.com/google/uuid"
)

type IBrandChatUseCase interface {
	GetUserConversation(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetConversationMessagesQueryReq) (*dto.BrandConversationDetailRes, error)
	SendUserMessage(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error)
	ListBrandConversations(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandConversationRes, error)
	ListConversationMessages(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, query dto.GetConversationMessagesQueryReq) (*dto.BrandConversationMessageListRes, error)
	SendStaffMessage(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error)
	MarkUserConversationRead(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandConversationRes, error)
	MarkStaffConversationRead(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error)
	CloseBrandConversation(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error)
	ReopenBrandConversation(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error)
}

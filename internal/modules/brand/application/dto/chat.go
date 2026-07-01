package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type SendBrandChatMessageReq struct {
	Message string `json:"message" binding:"required,min=1" label:"nội dung tin nhắn"`
}

type BrandConversationRes struct {
	ID               uuid.UUID  `json:"id"`
	BrandID          uuid.UUID  `json:"brandId"`
	UserID           uuid.UUID  `json:"userId"`
	CustomerId       uuid.UUID  `json:"customerId"`
	CustomerName     *string    `json:"customerName"`
	UserDisplayName  *string    `json:"userDisplayName"`
	Status           string     `json:"status"`
	LastMessageAt    *time.Time `json:"lastMessageAt"`
	UserLastReadAt   *time.Time `json:"userLastReadAt"`
	StaffLastReadAt  *time.Time `json:"staffLastReadAt"`
	UserUnreadCount  int        `json:"userUnreadCount"`
	StaffUnreadCount int        `json:"staffUnreadCount"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type BrandConversationMessageRes struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversationId"`
	SenderRole     string     `json:"senderRole"`
	SenderUserID   *uuid.UUID `json:"senderUserId"`
	Message        string     `json:"message"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type GetConversationMessagesQueryReq struct {
	shared_dto.PaginationQuery
}

type BrandConversationMessageListRes = shared_dto.PaginationResult[*BrandConversationMessageRes]

type BrandConversationDetailRes struct {
	BrandConversationRes
	Messages []*BrandConversationMessageRes `json:"messages"`
	Metadata shared_dto.PaginationMetadata  `json:"metadata"`
}

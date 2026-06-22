package dto

import (
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"time"

	"github.com/google/uuid"
)

type CreateChatSessionReq struct {
	Title *string `json:"title"`
}

type GetChatMessagesQueryReq struct {
	shared_dto.PaginationQuery
}

type ChatSessionRes struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	ContextSummary string    `json:"contextSummary"`
	IsArchived     bool      `json:"isArchived"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type SendChatMessageReq struct {
	Content string `json:"content" binding:"required" label:"nội dung tin nhắn"`
}

type ChatMessageRes struct {
	ID        uuid.UUID                   `json:"id"`
	Sender    messagesender.MessageSender `json:"sender"`
	Content   string                      `json:"content"`
	CreatedAt time.Time                   `json:"createdAt"`
}

type UpdateChatSessionReq struct {
	Title *string `json:"title" binding:"omitempty,max=255"`
}

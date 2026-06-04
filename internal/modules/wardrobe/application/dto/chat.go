package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateChatSessionReq struct {
	Title *string `json:"title"`
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
	Content string `json:"content" binding:"required"`
}

type ChatMessageRes struct {
	ID        uuid.UUID `json:"id"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

package mapper

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// MapChatSession maps a conversation entity into its response model.
func MapChatSession(item *entities.ConversationalContext) *dto.ChatSessionRes {
	if item == nil {
		return nil
	}
	return &dto.ChatSessionRes{
		ID:             item.ID,
		Title:          item.Title,
		ContextSummary: item.ContextSummary,
		IsArchived:     item.IsArchived,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

// MapChatMessage maps a chat message entity into its response model.
func MapChatMessage(item *entities.Message) *dto.ChatMessageRes {
	if item == nil {
		return nil
	}
	return &dto.ChatMessageRes{
		ID:        item.ID,
		Sender:    item.Sender,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}

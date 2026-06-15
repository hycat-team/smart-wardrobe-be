package chat

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

const outfitRedirectMessage = "Để nhận được gợi ý phối đồ chuẩn xác nhất từ thuật toán của Smart Wardrobe, bạn vui lòng sử dụng chức năng Phối đồ trên màn hình chính."

// buildChatSystemPrompt creates a compact fashion-aware system prompt for chat generation.
func buildChatSystemPrompt(summary string, wardrobeItems []*entities.WardrobeItem, recent []*entities.Message) string {
	var builder strings.Builder
	builder.WriteString("You are the AI fashion stylist of Closy. You must reply to the user in natural, friendly Vietnamese. Only recommend items from the user's available wardrobe items listed below. Do not suggest buying external products.\n")
	if strings.TrimSpace(summary) != "" {
		builder.WriteString("Summary of previous conversation:\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}

	builder.WriteString("Available wardrobe items:\n")
	limit := min(len(wardrobeItems), 20)
	for i := range limit {
		item := wardrobeItems[i]
		builder.WriteString("- ")
		if item.Category != nil {
			builder.WriteString(item.Category.Name)
			builder.WriteString(" ")
		}
		if item.Color != nil {
			builder.WriteString(*item.Color)
			builder.WriteString(" ")
		}
		if item.Style != nil {
			builder.WriteString(*item.Style)
			builder.WriteString(" ")
		}
		builder.WriteString("\n")
	}

	builder.WriteString("5 most recent messages:\n")
	for _, item := range recent {
		fmt.Fprintf(&builder, "%s: %s\n", item.Sender, item.Content)
	}

	return builder.String()
}

// isOutfitIntent detects whether the user is asking for an outfit recommendation.
func isOutfitIntent(content string) bool {
	lowered := strings.ToLower(content)
	keywords := []string{"phoi do", "outfit", "mac gi", "goi y do", "chon quan ao", "phoi cho toi"}
	for _, keyword := range keywords {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}

// mapChatSession maps a conversation entity into its response model.
func mapChatSession(item *entities.ConversationalContext) *dto.ChatSessionRes {
	return &dto.ChatSessionRes{
		ID:             item.ID,
		Title:          item.Title,
		ContextSummary: item.ContextSummary,
		IsArchived:     item.IsArchived,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

// mapChatMessage maps a chat message entity into its response model.
func mapChatMessage(item *entities.Message) *dto.ChatMessageRes {
	return &dto.ChatMessageRes{
		ID:        item.ID,
		Sender:    item.Sender,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}

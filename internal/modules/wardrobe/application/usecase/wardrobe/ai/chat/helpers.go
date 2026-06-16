package chat

import (
	"fmt"
	"regexp"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

const outfitRedirectMessage = "Bạn có thể tham khảo phong cách này nhé. Để nhận được gợi ý phối đồ chuẩn xác nhất từ tủ đồ cá nhân của bạn, vui lòng sử dụng chức năng Phối đồ."

var reWardrobeKeywords = regexp.MustCompile(`\b(tu do|ao|quan|vay|dam|giay|ao-khoac|ao khoac|do cua|mac|phoi|style|gu|mac gi)\b`)

// buildChatSystemPrompt creates a compact fashion-aware system prompt for chat generation.
func buildChatSystemPrompt(summary string, wardrobeItems []*entities.WardrobeItem, recent []*entities.Message) string {
	var builder strings.Builder
	builder.WriteString("You are the AI fashion stylist of Closy. You must reply to the user in natural, friendly Vietnamese. Do not suggest buying external products.\n")
	if strings.TrimSpace(summary) != "" {
		builder.WriteString("Summary of previous conversation:\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}

	if len(wardrobeItems) > 0 {
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
	}

	builder.WriteString("5 most recent messages:\n")
	for _, item := range recent {
		fmt.Fprintf(&builder, "%s: %s\n", item.Sender, item.Content)
	}

	return builder.String()
}

// isOutfitIntent detects whether the user is asking for an outfit recommendation.
func isOutfitIntent(content string) bool {
	normalized := strings.ToLower(shared.RemoveVietnameseSigns(content))
	keywords := []string{"phoi do", "outfit", "mac gi", "goi y do", "chon quan ao", "phoi cho toi"}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

// isWardrobeRelatedQuery detects whether the query contains keywords asking about wardrobe or styles.
func isWardrobeRelatedQuery(content string, recent []*entities.Message) bool {
	normalized := strings.ToLower(shared.RemoveVietnameseSigns(content))
	if reWardrobeKeywords.MatchString(normalized) {
		return true
	}
	// Also check last 2 messages for ongoing fashion context
	limit := min(len(recent), 2)
	for i := range limit {
		msg := recent[len(recent)-1-i]
		normalizedRecent := strings.ToLower(shared.RemoveVietnameseSigns(msg.Content))
		if reWardrobeKeywords.MatchString(normalizedRecent) {
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

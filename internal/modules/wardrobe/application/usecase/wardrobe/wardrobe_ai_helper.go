package wardrobe

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

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

func mapChatMessage(item *entities.Message) *dto.ChatMessageRes {
	return &dto.ChatMessageRes{
		ID:        item.ID,
		Sender:    item.Sender,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}

func buildRecommendationPrompt(groups []*dto.RecommendedItemGroup, input dto.RecommendOutfitReq) string {
	var builder strings.Builder
	builder.WriteString("Hãy viết một đoạn giải thích cho bộ phối đồ sau:\n")
	for _, group := range groups {
		builder.WriteString("- ")
		builder.WriteString(group.Role)
		builder.WriteString(": ")
		if group.Primary != nil && group.Primary.Category != nil {
			builder.WriteString(group.Primary.Category.Name)
		}
		if group.Primary != nil && group.Primary.Color != "" {
			builder.WriteString(" / ")
			builder.WriteString(group.Primary.Color)
		}
		builder.WriteString("\n")
	}
	if input.Occasion != nil {
		builder.WriteString("Occasion: ")
		builder.WriteString(*input.Occasion)
		builder.WriteString("\n")
	}
	if input.StyleTarget != nil {
		builder.WriteString("Style target: ")
		builder.WriteString(*input.StyleTarget)
		builder.WriteString("\n")
	}
	if input.Season != nil {
		builder.WriteString("Season: ")
		builder.WriteString(*input.Season)
		builder.WriteString("\n")
	}
	if input.Weather != nil {
		builder.WriteString("Weather: ")
		builder.WriteString(*input.Weather)
		builder.WriteString("\n")
	}
	if input.Details != nil {
		builder.WriteString("Chi tiết thêm: ")
		builder.WriteString(*input.Details)
		builder.WriteString("\n")
	}
	return builder.String()
}

func buildChatSystemPrompt(summary string, wardrobeItems []*entities.WardrobeItem, recent []*entities.Message) string {
	var builder strings.Builder
	builder.WriteString("Bạn là stylist AI của Smart Wardrobe. Chỉ được dựa trên trang phục có trong tủ đồ người dùng. Không gợi ý mua sản phẩm ngoài hệ thống.\n")
	if strings.TrimSpace(summary) != "" {
		builder.WriteString("Tom tat hoi thoai truoc do:\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}

	builder.WriteString("Trang phuc hien co:\n")
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

	builder.WriteString("5 tin nhan gan nhat:\n")
	for _, item := range recent {
		fmt.Fprintf(&builder, "%s: %s\n", item.Sender, item.Content)
	}

	return builder.String()
}

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

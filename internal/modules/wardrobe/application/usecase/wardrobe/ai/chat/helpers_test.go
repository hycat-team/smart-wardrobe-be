package chat

import (
	"testing"

	"smart-wardrobe-be/internal/shared/domain/entities"
)

func TestIsOutfitIntent(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"phối đồ đi chơi", true},
		{"phoi do di choi", true},
		{"hôm nay mặc gì hợp nhỉ", true},
		{"mac gi di lam", true},
		{"goi y do di tiec", true},
		{"gợi ý đồ đi tiệc", true},
		{"cho mình xin outfit đi học", true},
		{"bạn có khoẻ không", false},
		{"làm thế nào để bảo quản áo len", false},
	}

	for _, tt := range tests {
		actual := isOutfitIntent(tt.input)
		if actual != tt.expected {
			t.Errorf("isOutfitIntent(%q) = %v, expected %v", tt.input, actual, tt.expected)
		}
	}
}

func TestIsWardrobeRelatedQuery(t *testing.T) {
	tests := []struct {
		input    string
		recent   []*entities.Message
		expected bool
	}{
		{"mình có áo thun trắng trong tủ đồ không?", nil, true},
		{"ao thun do phoi voi gi", nil, true},
		{"chào bạn", nil, false},
		{"bạn khoẻ không", []*entities.Message{
			{Sender: "user", Content: "phối đồ đi chơi"},
		}, true}, // should be true due to context in recent messages
		{"bạn khoẻ không", []*entities.Message{
			{Sender: "user", Content: "chào bạn"},
		}, false},
	}

	for _, tt := range tests {
		actual := isWardrobeRelatedQuery(tt.input, tt.recent)
		if actual != tt.expected {
			t.Errorf("isWardrobeRelatedQuery(%q) = %v, expected %v", tt.input, actual, tt.expected)
		}
	}
}

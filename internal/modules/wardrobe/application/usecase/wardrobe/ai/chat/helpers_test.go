package chat

import (
	"strings"
	"testing"

	"smart-wardrobe-be/internal/shared/domain/entities"
)

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

func TestFilterThinkTags(t *testing.T) {
	tests := []struct {
		name           string
		chunks         []string
		expected       string
		expectedChunks []string
	}{
		{
			name:           "No thoughts and no marker (short fallback)",
			chunks:         []string{"Hello ", "world!"},
			expected:       "Hello world!",
			expectedChunks: []string{"Hello world!"},
		},
		{
			name:           "No thoughts and no marker (long bypass)",
			chunks:         []string{"Chào bạn! Hôm ", "nay mình rất ", "vui được giúp bạn."},
			expected:       "Chào bạn! Hôm nay mình rất vui được giúp bạn.",
			expectedChunks: []string{"Chào bạn! Hôm ", "nay mình rất ", "vui được giúp bạn."},
		},
		{
			name:           "Marker present",
			chunks:         []string{"* thought 1\n* thought 2\n===RESPONSE===\nHello ", "world!"},
			expected:       "Hello world!",
			expectedChunks: []string{"Hello ", "world!"},
		},
		{
			name:           "Split marker",
			chunks:         []string{"* thought\n===", "RESPONSE===\nHello ", "world!"},
			expected:       "Hello world!",
			expectedChunks: []string{"Hello ", "world!"},
		},
		{
			name:           "Fallback bullet thoughts with concatenation",
			chunks:         []string{"*   User says: \"Chào\"\n    *   Therefore, respond friendly.Hello ", "world!"},
			expected:       "Hello world!",
			expectedChunks: []string{"Hello ", "world!"},
		},
		{
			name:           "Fallback bullet thoughts with newlines",
			chunks:         []string{"* thought 1\n* thought 2\n", "Hello \n", "world!"},
			expected:       "Hello \nworld!",
			expectedChunks: []string{"Hello \n", "world!"},
		},
		{
			name:           "Fallback bullet thoughts with Vietnamese accents",
			chunks:         []string{"*   Therefore, recommend wardrobe.Đơn giản", " mà đẹp!"},
			expected:       "Đơn giản mà đẹp!",
			expectedChunks: []string{"Đơn giản", " mà đẹp!"},
		},
		{
			name:           "Paragraph thoughts starting with AI",
			chunks:         []string{"AI fashion stylist of Closy.\nUser is asking.Để mình ", "giúp bạn chọn đồ nhé!"},
			expected:       "Để mình giúp bạn chọn đồ nhé!",
			expectedChunks: []string{"Để mình ", "giúp bạn chọn đồ nhé!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inChan := make(chan string, len(tt.chunks))
			for _, c := range tt.chunks {
				inChan <- c
			}
			close(inChan)

			var accumulatedClean strings.Builder
			outChan := FilterThinkTags(inChan, func(chunk string) {
				accumulatedClean.WriteString(chunk)
			})

			var receivedChunks []string
			var streamResult strings.Builder
			for cleanChunk := range outChan {
				streamResult.WriteString(cleanChunk)
				receivedChunks = append(receivedChunks, cleanChunk)
			}

			if streamResult.String() != tt.expected {
				t.Errorf("Stream output = %q, expected %q", streamResult.String(), tt.expected)
			}
			if accumulatedClean.String() != tt.expected {
				t.Errorf("Accumulated clean output = %q, expected %q", accumulatedClean.String(), tt.expected)
			}
			if len(receivedChunks) != len(tt.expectedChunks) {
				t.Errorf("Received %d chunks (%v), expected %d chunks (%v)", len(receivedChunks), receivedChunks, len(tt.expectedChunks), tt.expectedChunks)
			} else {
				for i := range receivedChunks {
					if receivedChunks[i] != tt.expectedChunks[i] {
						t.Errorf("Chunk [%d] = %q, expected %q", i, receivedChunks[i], tt.expectedChunks[i])
					}
				}
			}
		})
	}
}

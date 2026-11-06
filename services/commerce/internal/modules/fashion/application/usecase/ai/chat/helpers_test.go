package chat

import (
	"fmt"
	"strings"
	"testing"

	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
)

func TestParseAndValidateSummaryDropsReasoningAndNormalizes(t *testing.T) {
	raw := "thinking first\n```json\n{\"preferences\":[\"a\\u0301o pastel\",\"a\\u0301o pastel\"],\"dislikes\":[],\"constraints\":[],\"important_context\":[]}\n```"
	got, err := parseAndValidateSummary(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != "{\"preferences\":[\"áo pastel\"],\"dislikes\":[],\"constraints\":[],\"important_context\":[]}" {
		t.Fatalf("unexpected summary %s", got)
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

func chatPromptTestItems(count int) []*fashion_dto.FashionItemContractDTO {
	items := make([]*fashion_dto.FashionItemContractDTO, 0, count)
	for i := range count {
		color := fmt.Sprintf("Màu %02d", i)
		items = append(items, &fashion_dto.FashionItemContractDTO{
			Category: &fashion_dto.CategoryBriefRes{Name: fmt.Sprintf("Áo %02d", i)},
			Color:    &color,
		})
	}
	return items
}

// The chat system prompt must not depend on Go map iteration order: the same
// wardrobe has to yield the same prompt on every turn, otherwise answers drift
// and provider prompt caching never hits.
func TestBuildChatSystemPromptIsStableAcrossCalls(t *testing.T) {
	items := chatPromptTestItems(12)

	first := buildChatSystemPromptWithLimits("", nil, nil, 0, 0, items, nil)
	for range 50 {
		got := buildChatSystemPromptWithLimits("", nil, nil, 0, 0, items, nil)
		if got != first {
			t.Fatalf("prompt changed between calls:\nfirst=%q\ngot=%q", first, got)
		}
	}
}

// Only the first maxPromptWardrobeItems entries may reach the prompt, in the
// order the caller supplied them.
func TestBuildChatSystemPromptCapsWardrobeItemsInOrder(t *testing.T) {
	items := chatPromptTestItems(maxPromptWardrobeItems + 5)

	prompt := buildChatSystemPromptWithLimits("", nil, nil, 0, 0, items, nil)

	for i := range maxPromptWardrobeItems {
		want := fmt.Sprintf("Áo %02d", i)
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt is missing expected item %q", want)
		}
	}
	for i := maxPromptWardrobeItems; i < len(items); i++ {
		unexpected := fmt.Sprintf("Áo %02d", i)
		if strings.Contains(prompt, unexpected) {
			t.Fatalf("prompt must not include an item beyond the cap, got %q", unexpected)
		}
	}
}

// Items without category, color or style carry nothing for the model and stay
// out of the prompt, exactly as before the ordering change.
func TestBuildChatSystemPromptSkipsEmptyItems(t *testing.T) {
	prompt := buildChatSystemPromptWithLimits("", nil, nil, 0, 0, []*fashion_dto.FashionItemContractDTO{
		nil,
		{},
		{Category: &fashion_dto.CategoryBriefRes{Name: "Quần jean"}},
	}, nil)

	if !strings.Contains(prompt, "Category: Quần jean") {
		t.Fatalf("prompt is missing the populated item: %q", prompt)
	}
	if strings.Count(prompt, "- Category:") != 1 {
		t.Fatalf("prompt must list exactly one item, got: %q", prompt)
	}
}

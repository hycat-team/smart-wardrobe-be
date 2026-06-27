package ai

import (
	"math"
	"testing"
)

func TestLocalTokenEstimator(t *testing.T) {
	estimator := NewLocalTokenEstimator(4.0, 1.25)

	tests := []struct {
		name     string
		input    []string
		expected int64
	}{
		{
			name:     "Empty input",
			input:    []string{},
			expected: 0,
		},
		{
			name:     "Single empty string",
			input:    []string{""},
			expected: 0,
		},
		{
			name:     "English text - 4 characters",
			input:    []string{"abcd"}, // 4 runes -> 4/4 * 1.25 = 1.25 -> ceil = 2
			expected: 2,
		},
		{
			name:     "English text - 8 characters",
			input:    []string{"abcdefgh"}, // 8 runes -> 8/4 * 1.25 = 2.5 -> ceil = 3
			expected: 3,
		},
		{
			name:     "Tiếng Việt có dấu",
			input:    []string{"Xin chào thế giới"}, // 17 runes -> 17/4 * 1.25 = 5.3125 -> ceil = 6
			expected: 6,
		},
		{
			name:     "Emoji text",
			input:    []string{"👋 Hello 🚀"}, // 9 runes -> 9/4 * 1.25 = 2.8125 -> ceil = 3
			expected: 3,
		},
		{
			name:     "Multiple text blocks",
			input:    []string{"System prompt", "User request"}, // 13 + 12 = 25 runes -> 25/4 * 1.25 = 7.8125 -> ceil = 8
			expected: 8,
		},
		{
			name:     "Integer overflow protection",
			input:    []string{string(make([]rune, 1000000))}, // Large array of runes to check scaling
			expected: 312500, // 1000000/4 * 1.25 = 312500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := estimator.EstimateFromText(tt.input...)
			if actual != tt.expected {
				t.Errorf("EstimateFromText() for %s = %d, want %d", tt.name, actual, tt.expected)
			}
		})
	}
}

func TestLocalTokenEstimator_Overflow(t *testing.T) {
	estimator := NewLocalTokenEstimator(4.0, 1.25)
	
	// Create mock large string slices that trigger internal protection
	actual := estimator.EstimateFromText("some text", string(make([]rune, math.MaxInt16)))
	if actual <= 0 {
		t.Errorf("expected positive token count for large content, got %d", actual)
	}
}

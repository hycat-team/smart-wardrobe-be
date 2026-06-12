package ai

import (
	"strings"
	"testing"
)

func TestLocalNLPParser_Parse(t *testing.T) {
	parser := NewLocalNLPParser()

	tests := []struct {
		name                 string
		input                string
		expectedOccasion     string
		expectedStyles       []string
		expectedColorTone    string
		expectedPositive     []string
		expectedNegative     []string
		semanticQueryContain []string
	}{
		{
			name:              "Basic casual request with accents",
			input:             "Tôi muốn một bộ đồ đi chơi đơn giản phong cách tối giản màu sáng",
			expectedOccasion:  "casual",
			expectedStyles:    []string{"minimalist"},
			expectedColorTone: "light",
			expectedPositive:  nil,
			expectedNegative:  nil,
			semanticQueryContain: []string{
				"occasion: casual",
				"style: minimalist",
				"color tone: light",
			},
		},
		{
			name:              "Office formal work request with cold weather",
			input:             "Phối đồ đi làm văn phòng thanh lịch ấm áp cho ngày lạnh",
			expectedOccasion:  "work",
			expectedStyles:    []string{"elegant"},
			expectedColorTone: "",
			expectedPositive:  []string{"cold"},
			expectedNegative:  nil,
			semanticQueryContain: []string{
				"occasion: work",
				"style: elegant",
				"constraints: cold",
			},
		},
		{
			name:              "Negation of winter weather",
			input:             "Đi tiệc sang trọng nhưng không muốn lạnh",
			expectedOccasion:  "party",
			expectedStyles:    []string{"elegant"},
			expectedColorTone: "",
			expectedPositive:  nil,
			expectedNegative:  []string{"avoid-cold"},
			semanticQueryContain: []string{
				"occasion: party",
				"style: elegant",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.input)

			if result.Occasion != tt.expectedOccasion {
				t.Errorf("expected occasion %q, got %q", tt.expectedOccasion, result.Occasion)
			}

			if len(result.StyleTarget) != len(tt.expectedStyles) {
				t.Errorf("expected styles count %d, got %d", len(tt.expectedStyles), len(result.StyleTarget))
			} else {
				for i, s := range result.StyleTarget {
					if s != tt.expectedStyles[i] {
						t.Errorf("expected style at %d to be %q, got %q", i, tt.expectedStyles[i], s)
					}
				}
			}

			if result.ColorTone != tt.expectedColorTone {
				t.Errorf("expected color tone %q, got %q", tt.expectedColorTone, result.ColorTone)
			}

			if len(result.PositiveConstraints) != len(tt.expectedPositive) {
				t.Errorf("expected positive constraints count %d, got %d", len(tt.expectedPositive), len(result.PositiveConstraints))
			}

			if len(result.NegativeConstraints) != len(tt.expectedNegative) {
				t.Errorf("expected negative constraints count %d, got %d", len(tt.expectedNegative), len(result.NegativeConstraints))
			}

			for _, sub := range tt.semanticQueryContain {
				if !strings.Contains(result.SemanticQuery, sub) {
					t.Errorf("expected semantic query to contain %q, but got %q", sub, result.SemanticQuery)
				}
			}
		})
	}
}

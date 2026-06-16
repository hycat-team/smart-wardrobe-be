package recommendation

import (
	"reflect"
	"strings"
	"testing"
)

// TestParserNormalizeText verifies Telex orphan stripping and standard unicode normalization.
func TestParserNormalizeText(t *testing.T) {
	parser := NewLocalNLPParser()

	tests := []struct {
		input    string
		expected string
	}{
		{"đi chơi", "di choi"},
		{"di choiw", "di choi"},
		{"thanh lichj", "thanh lich"},
		{"don gianr", "don gian"},
		{"bong das", "bong da"},
		{"muaf dong", "mua dong"},
		{"tennis", "tennis"},
		{"summer", "summer"},
		{"interview", "interview"},
		{"golf", "golf"},
		{"new dress", "new dress"},
		{"di choiw nhung khong mac vay", "di choi nhung khong mac vay"},
	}

	for _, tt := range tests {
		actual := parser.normalizeText(tt.input)
		cleanedActual := strings.ReplaceAll(actual, " , ", ",")
		cleanedActual = strings.ReplaceAll(cleanedActual, " ; ", ";")
		cleanedActual = strings.ReplaceAll(cleanedActual, " - ", "-")
		cleanedActual = strings.ReplaceAll(cleanedActual, " . ", ".")
		cleanedActual = strings.TrimSpace(cleanedActual)

		if cleanedActual != tt.expected {
			t.Errorf("normalizeText(%q) = %q (cleaned: %q), expected %q", tt.input, actual, cleanedActual, tt.expected)
		}
	}
}

// TestParserSplitSentences verifies sentence splitting by punctuation and transitional words.
func TestParserSplitSentences(t *testing.T) {
	parser := NewLocalNLPParser()

	tests := []struct {
		input    string
		expected []string
	}{
		{"di choi. di lam", []string{"di choi", "di lam"}},
		{"di choi nhung khong mac vay", []string{"di choi", "nhung khong mac vay"}},
		{"quan jean ma lich su", []string{"quan jean", "ma lich su"}},
		{"dam dep chu khong loi mot", []string{"dam dep", "chu khong loi mot"}},
		{"mac dep, sang trong; di tiec", []string{"mac dep", "sang trong", "di tiec"}},
	}

	for _, tt := range tests {
		normalized := parser.normalizeText(tt.input)
		actual := parser.splitSentences(normalized)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("splitSentences(%q) = %v, expected %v", tt.input, actual, tt.expected)
		}
	}
}

// TestParserParseIntent verifies the parsing of occasions, styles, colors, and constraints.
func TestParserParseIntent(t *testing.T) {
	parser := NewLocalNLPParser()

	// 1. Test run-on punctuation gõ dính: "di lam,dl" should detect work occasion
	intentPunctuation := parser.Parse("di lam,dl")
	expectedOccasionsPunctuation := []string{"work"}
	if !reflect.DeepEqual(intentPunctuation.Occasion, expectedOccasionsPunctuation) {
		t.Errorf("expected occasions for run-on punctuation %v, got %v", expectedOccasionsPunctuation, intentPunctuation.Occasion)
	}

	// 2. Test negation scope for color tone: "di choi nhung k thich mac do toi"
	intentNegationColor := parser.Parse("di choi nhung k thich mac do toi")
	expectedOccasionsNegationColor := []string{"casual"}
	if !reflect.DeepEqual(intentNegationColor.Occasion, expectedOccasionsNegationColor) {
		t.Errorf("expected occasions %v, got %v", expectedOccasionsNegationColor, intentNegationColor.Occasion)
	}
	if len(intentNegationColor.ColorTone) != 0 {
		t.Errorf("expected negated color tone 'dark' to be filtered out, but got %v", intentNegationColor.ColorTone)
	}

	// 3. Test slang negation word: "phoi do di ca phe che mau toi nha"
	intentSlang := parser.Parse("phoi do di ca phe che mau toi nha")
	if len(intentSlang.ColorTone) != 0 {
		t.Errorf("expected negated color tone 'dark' via slang 'che' to be filtered out, but got %v", intentSlang.ColorTone)
	}

	// 4. Test weather constraints negation scope: "di choi nhung khong thich lanh"
	intentWeather := parser.Parse("di choi nhung khong thich lanh")
	expectedNegatives := []string{"avoid-cold"}
	if !reflect.DeepEqual(intentWeather.NegativeConstraints, expectedNegatives) {
		t.Errorf("expected negative constraints %v, got %v", expectedNegatives, intentWeather.NegativeConstraints)
	}
}

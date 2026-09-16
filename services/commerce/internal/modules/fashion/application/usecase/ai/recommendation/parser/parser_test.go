package parser

import (
	"reflect"
	"strings"
	"testing"
)

// TestParserNormalizeText verifies Telex orphan stripping and standard unicode normalization.
func TestParserNormalizeText(t *testing.T) {
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
		actual := NormalizeText(tt.input)
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
		normalized := NormalizeText(tt.input)
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
	expectedNegatives := []string{"avoid-weather:cold"}
	if !reflect.DeepEqual(intentWeather.NegativeConstraints, expectedNegatives) {
		t.Errorf("expected negative constraints %v, got %v", expectedNegatives, intentWeather.NegativeConstraints)
	}

	// 5. "dung" should be treated as use/apply, while "đừng" is negation.
	intentUseStyle := parser.Parse("dung phong cach toi gian")
	if !reflect.DeepEqual(intentUseStyle.StyleTarget, []string{"minimalist"}) {
		t.Errorf("expected positive minimalist style, got %v", intentUseStyle.StyleTarget)
	}

	intentAvoidStyle := parser.Parse("đừng dùng phong cách tối giản")
	if !reflect.DeepEqual(intentAvoidStyle.ExcludedStyles, []string{"minimalist"}) {
		t.Errorf("expected excluded minimalist style, got %v", intentAvoidStyle.ExcludedStyles)
	}
}

func TestParserKeywordMatchesIncludePositions(t *testing.T) {
	parser := NewLocalNLPParser()
	normalized := NormalizeText("di lam mac toi gian")

	matches := parser.detectMatches(normalized, "style", parser.styles)
	if len(matches) != 1 {
		t.Fatalf("expected one style match, got %v", matches)
	}
	if matches[0].Value != "minimalist" || matches[0].Start < 0 || matches[0].End <= matches[0].Start {
		t.Fatalf("unexpected match metadata: %+v", matches[0])
	}
}

func TestParserNegationWindowDoesNotNegateDistantPositiveStyle(t *testing.T) {
	parser := NewLocalNLPParser()

	intent := parser.Parse("khong thich mau toi nhung toi gian")
	if !reflect.DeepEqual(intent.ExcludedColorTones, []string{"dark"}) {
		t.Fatalf("expected excluded dark color tone, got %v", intent.ExcludedColorTones)
	}
	if !reflect.DeepEqual(intent.StyleTarget, []string{"minimalist"}) {
		t.Fatalf("expected positive minimalist style, got %v", intent.StyleTarget)
	}
}

func TestParserPopulatesLexicalTerms(t *testing.T) {
	parser := NewLocalNLPParser()

	intent := parser.Parse("troi lanh muon mac cardigan len cashmere")
	for _, term := range []string{"cold", "cardigan", "len", "cashmere"} {
		if !containsString(intent.LexicalTerms, term) {
			t.Fatalf("expected lexical term %q in %v", term, intent.LexicalTerms)
		}
	}
}

func TestRemoveConflictingLexicalTerms(t *testing.T) {
	parser := NewLocalNLPParser()
	intent := parser.Parse("di tiec mac mau toi")
	intent.Occasion = []string{"work"}
	intent.ColorTone = []string{"light"}

	filtered := parser.RemoveConflictingLexicalTerms(intent)
	for _, term := range []string{"party", "dark"} {
		if containsString(filtered, term) {
			t.Fatalf("expected conflicting term %q to be removed from %v", term, filtered)
		}
	}
}

func TestParserVietnameseSlangAndDiacritics(t *testing.T) {
	parser := NewLocalNLPParser()

	// Test case 1: diacritics and slang "đi quẩy" -> party occasion
	intent1 := parser.Parse("Hôm nay đi quẩy cực vui")
	if !reflect.DeepEqual(intent1.Occasion, []string{"party"}) {
		t.Errorf("expected occasion 'party' for 'đi quẩy', got %v", intent1.Occasion)
	}

	// Test case 2: slang "rét căm căm" -> cold weather constraint
	intent2 := parser.Parse("Trời rét căm căm ghê")
	if !reflect.DeepEqual(intent2.PositiveConstraints, []string{"cold"}) {
		t.Errorf("expected weather constraint 'cold' for 'rét căm căm', got %v", intent2.PositiveConstraints)
	}

	// Test case 3: slang "đi cày" -> work occasion
	intent3 := parser.Parse("mặc gì đi cày hôm nay")
	if !reflect.DeepEqual(intent3.Occasion, []string{"work"}) {
		t.Errorf("expected occasion 'work' for 'đi cày', got %v", intent3.Occasion)
	}

	// Test case 4: slang "nắng gắt" -> hot weather constraint
	intent4 := parser.Parse("ngoài trời nắng gắt quá")
	if !reflect.DeepEqual(intent4.PositiveConstraints, []string{"hot"}) {
		t.Errorf("expected weather constraint 'hot' for 'nắng gắt', got %v", intent4.PositiveConstraints)
	}
}

func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

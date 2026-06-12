package ai

import (
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type LocalNLPParser struct {
	occasions    map[string][]string
	styles       map[string][]string
	colorTones   map[string][]string
	weathers     map[string][]string
	negations    []string
}

func NewLocalNLPParser() *LocalNLPParser {
	return &LocalNLPParser{
		occasions: map[string][]string{
			"casual": {"di choi", "dao pho", "cafe", "tu do", "dao", "hop lop", "casual", "hangout", "street"},
			"work":   {"di lam", "cong so", "van phong", "hop", "thuyet trinh", "work", "office", "interview", "formal"},
			"date":   {"hen ho", "di choi voi nguoi yeu", "date", "romantic", "gap mat", "to tinh"},
			"party":  {"di tiec", "party", "dam cuoi", "sinh nhat", "bar", "club", "event", "su kien", "ky niem"},
			"sport":  {"the thao", "chay bo", "gym", "sport", "workout", "golf", "tennis", "da bong"},
		},
		styles: map[string][]string{
			"minimalist": {"toi gian", "minimalist", "don gian", "basic", "nhe nhang"},
			"vintage":    {"co dien", "vintage", "retro", "classic"},
			"streetwear": {"duong pho", "streetwear", "nang dong", "ca tinh", "hiphop", "swag"},
			"preppy":     {"hoc duong", "preppy", "sinh vien"},
			"elegant":    {"thanh lich", "elegant", "quy phai", "sang trong", "luxury"},
		},
		colorTones: map[string][]string{
			"light":  {"mau sang", "tong sang", "light", "trang", "kem", "pastel"},
			"dark":   {"mau toi", "tong toi", "dark", "den", "xam", "tram"},
			"earthy": {"tong dat", "earthy", "nau", "be", "olive", "reu"},
		},
		weathers: map[string][]string{
			"cold":  {"lanh", "ret", "dong", "cold", "winter"},
			"cool":  {"mat", "se lanh", "thu", "cool", "autumn"},
			"hot":   {"nong", "he", "nang", "hot", "summer"},
			"rainy": {"mua", "rainy", "uot"},
		},
		negations: []string{"khong", "dung", "tranh", "tiêu cuu", "ne", "avoid", "no", "not", "without"},
	}
}

// Parse parses the input query string to decompose intent and constraints.
// The parser maps free text to occasions, styles, colors, and seasonality preferences,
// supporting negation handling and producing an expanded semantic query.
func (p *LocalNLPParser) Parse(freeText string) dto.ParsedIntent {
	var intent dto.ParsedIntent
	if strings.TrimSpace(freeText) == "" {
		return intent
	}

	// Step 1: Normalize the raw input text (lowercase, strip diacritics/accents)
	normalized := p.normalizeText(freeText)
	words := strings.Fields(normalized)

	// Step 2: Detect predefined style, occasion, and color tone keywords
	intent.Occasion = p.detectOccasion(normalized)
	intent.StyleTarget = p.detectStyles(normalized)
	intent.ColorTone = p.detectColorTone(normalized)

	var exactKeywords []string
	var positiveConstraints []string
	var negativeConstraints []string

	// Step 3: Parse sentences for negation logic (e.g. "not cold", "avoid rainy")
	sentences := strings.Split(normalized, ".")
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if trimmed == "" {
			continue
		}

		// Check for negation keywords (e.g., "khong", "tranh", "not") in the segment
		hasNegation := false
		for _, neg := range p.negations {
			if strings.Contains(trimmed, neg) {
				hasNegation = true
				break
			}
		}

		// Determine if weather/season constraints are matched
		matchedAny := false
		for category, keywords := range p.weathers {
			for _, kw := range keywords {
				if strings.Contains(trimmed, kw) {
					matchedAny = true
					if hasNegation {
						// e.g., "tranh troi lanh" -> negative constraint: avoid-cold
						negativeConstraints = append(negativeConstraints, "avoid-"+category)
					} else {
						// e.g., "ngay ret" -> positive constraint: cold
						positiveConstraints = append(positiveConstraints, category)
					}
				}
			}
		}

		// Gather remaining words longer than 2 characters as exact keywords for Lexical GIN Index search
		if !matchedAny {
			for _, word := range words {
				if len(word) > 2 {
					exactKeywords = append(exactKeywords, word)
				}
			}
		}
	}

	// Step 4: Remove duplicates and set properties
	intent.ExactKeywords = p.deduplicateStrings(exactKeywords)
	intent.PositiveConstraints = p.deduplicateStrings(positiveConstraints)
	intent.NegativeConstraints = p.deduplicateStrings(negativeConstraints)

	// Step 5: Build query expansion template for vector search
	intent.SemanticQuery = p.buildSemanticQuery(intent, freeText)

	return intent
}

func (p *LocalNLPParser) normalizeText(text string) string {
	lowered := strings.ToLower(text)
	r := strings.NewReplacer(
		"à", "a", "á", "a", "ạ", "a", "ả", "a", "ã", "a",
		"â", "a", "ầ", "a", "ấ", "a", "ậ", "a", "ẩ", "a", "ẫ", "a",
		"ă", "a", "ằ", "a", "ắ", "a", "ặ", "a", "ẳ", "a", "ẵ", "a",
		"è", "e", "é", "e", "ẹ", "e", "ẻ", "e", "ẽ", "e",
		"ê", "e", "ề", "e", "ế", "e", "ệ", "e", "ể", "e", "ễ", "e",
		"ì", "i", "í", "i", "ị", "i", "ỉ", "i", "ĩ", "i",
		"ò", "o", "ó", "o", "ọ", "o", "ỏ", "o", "õ", "o",
		"ô", "o", "ồ", "o", "ố", "o", "ộ", "o", "ổ", "o", "ỗ", "o",
		"ơ", "o", "ờ", "o", "ớ", "o", "ợ", "o", "ở", "o", "ỡ", "o",
		"ù", "u", "ú", "u", "ụ", "u", "ủ", "u", "ũ", "u",
		"ư", "u", "ừ", "u", "ứ", "u", "ự", "u", "ử", "u", "ữ", "u",
		"ỳ", "y", "ý", "y", "ỵ", "y", "ỷ", "y", "ỹ", "y",
		"đ", "d",
	)
	return r.Replace(lowered)
}

func (p *LocalNLPParser) detectOccasion(normalized string) string {
	for name, keywords := range p.occasions {
		for _, kw := range keywords {
			if strings.Contains(normalized, kw) {
				return name
			}
		}
	}
	return ""
}

func (p *LocalNLPParser) detectStyles(normalized string) []string {
	var matched []string
	for name, keywords := range p.styles {
		for _, kw := range keywords {
			if strings.Contains(normalized, kw) {
				matched = append(matched, name)
				break
			}
		}
	}
	return matched
}

func (p *LocalNLPParser) detectColorTone(normalized string) string {
	for name, keywords := range p.colorTones {
		for _, kw := range keywords {
			if strings.Contains(normalized, kw) {
				return name
			}
		}
	}
	return ""
}

func (p *LocalNLPParser) deduplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func (p *LocalNLPParser) buildSemanticQuery(intent dto.ParsedIntent, originalText string) string {
	var parts []string
	if intent.Occasion != "" {
		parts = append(parts, "occasion: "+intent.Occasion)
	}
	if len(intent.StyleTarget) > 0 {
		parts = append(parts, "style: "+strings.Join(intent.StyleTarget, ", "))
	}
	if intent.ColorTone != "" {
		parts = append(parts, "color tone: "+intent.ColorTone)
	}
	if len(intent.PositiveConstraints) > 0 {
		parts = append(parts, "constraints: "+strings.Join(intent.PositiveConstraints, ", "))
	}
	parts = append(parts, originalText)

	return strings.Join(parts, " | ")
}

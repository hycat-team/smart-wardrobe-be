package recommendation

import (
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type LocalNLPParser struct {
	occasions  map[string][]string
	styles     map[string][]string
	colorTones map[string][]string
	weathers   map[string][]string
	negations  []string
}

// NewLocalNLPParser builds the local parser used to enrich recommendation intent from free text.
func NewLocalNLPParser() *LocalNLPParser {
	return &LocalNLPParser{
		occasions:  map[string][]string{"casual": {"di choi", "dao pho", "cafe", "tu do", "dao", "hop lop", "casual", "hangout", "street"}, "work": {"di lam", "cong so", "van phong", "hop", "thuyet trinh", "work", "office", "interview", "formal"}, "date": {"hen ho", "di choi voi nguoi yeu", "date", "romantic", "gap mat", "to tinh"}, "party": {"di tiec", "party", "dam cuoi", "sinh nhat", "bar", "club", "event", "su kien", "ky niem"}, "sport": {"the thao", "chay bo", "gym", "sport", "workout", "golf", "tennis", "da bong"}},
		styles:     map[string][]string{"minimalist": {"toi gian", "minimalist", "don gian", "basic", "nhe nhang"}, "vintage": {"co dien", "vintage", "retro", "classic"}, "streetwear": {"duong pho", "streetwear", "nang dong", "ca tinh", "hiphop", "swag"}, "preppy": {"hoc duong", "preppy", "sinh vien"}, "elegant": {"thanh lich", "elegant", "quy phai", "sang trong", "luxury"}},
		colorTones: map[string][]string{"light": {"mau sang", "tong sang", "light", "trang", "kem", "pastel"}, "dark": {"mau toi", "tong toi", "dark", "den", "xam", "tram"}, "earthy": {"tong dat", "earthy", "nau", "be", "olive", "reu"}},
		weathers:   map[string][]string{"cold": {"lanh", "ret", "dong", "cold", "winter"}, "cool": {"mat", "se lanh", "thu", "cool", "autumn"}, "hot": {"nong", "he", "nang", "hot", "summer"}, "rainy": {"mua", "rainy", "uot"}},
		negations:  []string{"khong", "dung", "tranh", "tiÃªu cuu", "ne", "avoid", "no", "not", "without"},
	}
}

// Parse parses the input query string to decompose intent and constraints.
func (p *LocalNLPParser) Parse(freeText string) dto.ParsedIntent {
	var intent dto.ParsedIntent
	if strings.TrimSpace(freeText) == "" {
		return intent
	}
	normalized := p.normalizeText(freeText)
	words := strings.Fields(normalized)
	intent.Occasion = p.detectOccasion(normalized)
	intent.StyleTarget = p.detectStyles(normalized)
	intent.ColorTone = p.detectColorTone(normalized)
	var exactKeywords, positiveConstraints, negativeConstraints []string
	for sentence := range strings.SplitSeq(normalized, ".") {
		trimmed := strings.TrimSpace(sentence)
		if trimmed == "" {
			continue
		}
		hasNegation := false
		for _, neg := range p.negations {
			if strings.Contains(trimmed, neg) {
				hasNegation = true
				break
			}
		}
		matchedAny := false
		for category, keywords := range p.weathers {
			for _, kw := range keywords {
				if strings.Contains(trimmed, kw) {
					matchedAny = true
					if hasNegation {
						negativeConstraints = append(negativeConstraints, "avoid-"+category)
					} else {
						positiveConstraints = append(positiveConstraints, category)
					}
				}
			}
		}
		if !matchedAny {
			for _, word := range words {
				if len(word) > 2 {
					exactKeywords = append(exactKeywords, word)
				}
			}
		}
	}
	intent.ExactKeywords = p.deduplicateStrings(exactKeywords)
	intent.PositiveConstraints = p.deduplicateStrings(positiveConstraints)
	intent.NegativeConstraints = p.deduplicateStrings(negativeConstraints)
	intent.SemanticQuery = p.buildSemanticQuery(intent, freeText)
	return intent
}

func (p *LocalNLPParser) normalizeText(text string) string {
	lowered := strings.ToLower(text)
	r := strings.NewReplacer("Ã ", "a", "Ã¡", "a", "áº¡", "a", "áº£", "a", "Ã£", "a", "Ã¢", "a", "áº§", "a", "áº¥", "a", "áº­", "a", "áº©", "a", "áº«", "a", "Äƒ", "a", "áº±", "a", "áº¯", "a", "áº·", "a", "áº³", "a", "áºµ", "a", "Ã¨", "e", "Ã©", "e", "áº¹", "e", "áº»", "e", "áº½", "e", "Ãª", "e", "á»", "e", "áº¿", "e", "á»‡", "e", "á»ƒ", "e", "á»…", "e", "Ã¬", "i", "Ã­", "i", "á»‹", "i", "á»‰", "i", "Ä©", "i", "Ã²", "o", "Ã³", "o", "á»", "o", "á»", "o", "Ãµ", "o", "Ã´", "o", "á»“", "o", "á»‘", "o", "á»™", "o", "á»•", "o", "á»—", "o", "Æ¡", "o", "á»", "o", "á»›", "o", "á»£", "o", "á»Ÿ", "o", "á»¡", "o", "Ã¹", "u", "Ãº", "u", "á»¥", "u", "á»§", "u", "Å©", "u", "Æ°", "u", "á»«", "u", "á»©", "u", "á»±", "u", "á»­", "u", "á»¯", "u", "á»³", "y", "Ã½", "y", "á»µ", "y", "á»·", "y", "á»¹", "y", "Ä‘", "d")
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
	keys := map[string]bool{}
	var list []string
	for _, entry := range slice {
		if !keys[entry] {
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

package recommendation

import (
	"regexp"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

var (
	reOrphanTelex   = regexp.MustCompile(`\b([a-z]+)([sfrxjw])\b`)
	reSplitSentence = regexp.MustCompile(`[,;\-\.]`)

	protectedWords = map[string]bool{
		"tennis": true, "summer": true, "winter": true, "interview": true,
		"golf": true, "jeans": true, "shorts": true, "pants": true,
		"shoes": true, "socks": true, "dress": true, "classics": true,
		"classic": true, "streetwear": true, "bar": true, "wear": true,
		"new": true, "show": true, "view": true, "raw": true, "crew": true,
		"glow": true, "flow": true, "slow": true, "mix": true, "max": true,
		"tax": true, "box": true, "flex": true, "wax": true, "fix": true,
		"sex": true, "plus": true, "class": true, "cross": true, "glass": true,
		"boss": true, "loss": true, "miss": true, "mess": true, "press": true,
		"stress": true, "process": true, "business": true, "focus": true,
		"minus": true, "bus": true, "gas": true, "yes": true, "this": true,
	}
)

type LocalNLPParser struct {
	occasions    map[string][]string
	styles       map[string][]string
	colorTones   map[string][]string
	weathers     map[string][]string
	negations    []string
	keywordRegex map[string]*regexp.Regexp
}

// NewLocalNLPParser builds the local parser used to enrich recommendation intent from free text.
func NewLocalNLPParser() *LocalNLPParser {
	parser := &LocalNLPParser{
		occasions: map[string][]string{
			"casual": {"di choi", "dchoi", "dc", "dao pho", "cafe", "tu do", "dao", "hop lop", "casual", "hangout", "street"},
			"work":   {"di lam", "dlam", "dl", "cong so", "van phong", "hop", "thuyet trinh", "work", "office", "interview", "formal"},
			"date":   {"hen ho", "di choi voi nguoi yeu", "date", "romantic", "gap mat", "to tinh"},
			"party":  {"di tiec", "dtiec", "party", "dam cuoi", "sinh nhat", "sn", "bar", "club", "event", "su kien", "ky niem"},
			"sport":  {"the thao", "tt", "chay bo", "gym", "sport", "workout", "golf", "tennis", "da bong"},
		},
		styles: map[string][]string{
			"minimalist": {"toi gian", "tgian", "minimalist", "don gian", "dgian", "basic", "nhe nhang"},
			"vintage":    {"co dien", "vintage", "retro", "classic"},
			"streetwear": {"duong pho", "streetwear", "nang dong", "ca tinh", "hiphop", "swag"},
			"preppy":     {"hoc duong", "preppy", "sinh vien", "sv"},
			"elegant":    {"thanh lich", "tlich", "elegant", "quy phai", "sang trong", "luxury"},
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
		negations: []string{"khong", "dung", "tranh", "tieu cuc", "ne", "avoid", "no", "not", "without", "k", "ko", "kh", "ghet", "che", "thoi"},
	}

	parser.keywordRegex = make(map[string]*regexp.Regexp)

	addKeyword := func(kw string) {
		if _, exists := parser.keywordRegex[kw]; !exists {
			parser.keywordRegex[kw] = regexp.MustCompile(`\b` + regexp.QuoteMeta(kw) + `\b`)
		}
	}

	for _, kws := range parser.occasions {
		for _, kw := range kws {
			addKeyword(kw)
		}
	}
	for _, kws := range parser.styles {
		for _, kw := range kws {
			addKeyword(kw)
		}
	}
	for _, kws := range parser.colorTones {
		for _, kw := range kws {
			addKeyword(kw)
		}
	}
	for _, kws := range parser.weathers {
		for _, kw := range kws {
			addKeyword(kw)
		}
	}
	for _, kw := range parser.negations {
		addKeyword(kw)
	}

	return parser
}

// Parse parses the input query string to decompose intent and constraints.
func (p *LocalNLPParser) Parse(freeText string) dto.ParsedIntent {
	var intent dto.ParsedIntent
	if strings.TrimSpace(freeText) == "" {
		return intent
	}
	normalized := p.normalizeText(freeText)
	words := strings.Fields(normalized)

	var exactKeywords, positiveConstraints, negativeConstraints []string
	var occasions, styles, colorTones []string

	sentences := p.splitSentences(normalized)
	for _, trimmed := range sentences {
		hasNegation := false
		for _, neg := range p.negations {
			if p.containsPhrase(trimmed, neg) {
				hasNegation = true
				break
			}
		}

		segOccasions := p.detectOccasion(trimmed)
		segStyles := p.detectStyles(trimmed)
		segColorTones := p.detectColorTone(trimmed)

		if hasNegation {
			// Negated elements are excluded from positive detection results
		} else {
			occasions = append(occasions, segOccasions...)
			styles = append(styles, segStyles...)
			colorTones = append(colorTones, segColorTones...)
		}

		matchedAny := false
		for category, keywords := range p.weathers {
			for _, kw := range keywords {
				if p.containsPhrase(trimmed, kw) {
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

	intent.Occasion = p.deduplicateStrings(occasions)
	intent.StyleTarget = p.deduplicateStrings(styles)
	intent.ColorTone = p.deduplicateStrings(colorTones)
	intent.ExactKeywords = p.deduplicateStrings(exactKeywords)
	intent.PositiveConstraints = p.deduplicateStrings(positiveConstraints)
	intent.NegativeConstraints = p.deduplicateStrings(negativeConstraints)
	intent.SemanticQuery = p.buildSemanticQuery(intent, freeText)
	return intent
}

// normalizeText normalizes input text to lowercased unaccented ASCII characters and cleans trailing Telex orphans.
func (p *LocalNLPParser) normalizeText(text string) string {
	lowered := strings.ToLower(text)

	padded := lowered
	padded = strings.ReplaceAll(padded, ",", " , ")
	padded = strings.ReplaceAll(padded, ";", " ; ")
	padded = strings.ReplaceAll(padded, "-", " - ")
	padded = strings.ReplaceAll(padded, ".", " . ")

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
	replaced := r.Replace(padded)

	return reOrphanTelex.ReplaceAllStringFunc(replaced, func(match string) string {
		if protectedWords[match] {
			return match
		}
		prefix := match[:len(match)-1]
		hasVowel := false
		for i := 0; i < len(prefix); i++ {
			c := prefix[i]
			if c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u' || c == 'y' {
				hasVowel = true
				break
			}
		}
		if hasVowel {
			return prefix
		}
		return match
	})
}

// splitSentences splits a text segment into smaller parts based on punctuation and transitional words.
func (p *LocalNLPParser) splitSentences(normalized string) []string {
	processed := normalized
	processed = strings.ReplaceAll(processed, " nhung ", " , nhung ")
	processed = strings.ReplaceAll(processed, " ma ", " , ma ")
	processed = strings.ReplaceAll(processed, " chu ", " , chu ")

	rawSegments := reSplitSentence.Split(processed, -1)
	var sentences []string
	for _, segment := range rawSegments {
		trimmed := strings.TrimSpace(segment)
		if trimmed != "" {
			sentences = append(sentences, trimmed)
		}
	}
	return sentences
}

// containsPhrase checks if the text contains the specified phrase with word boundaries.
func (p *LocalNLPParser) containsPhrase(text, phrase string) bool {
	if !strings.Contains(text, phrase) {
		return false
	}
	if re, exists := p.keywordRegex[phrase]; exists {
		return re.MatchString(text)
	}
	return false
}

// detectOccasion detects all occasions present in the normalized text.
func (p *LocalNLPParser) detectOccasion(normalized string) []string {
	var matched []string
	for name, keywords := range p.occasions {
		for _, kw := range keywords {
			if p.containsPhrase(normalized, kw) {
				matched = append(matched, name)
				break
			}
		}
	}
	return matched
}

// detectStyles detects all styles present in the normalized text.
func (p *LocalNLPParser) detectStyles(normalized string) []string {
	var matched []string
	for name, keywords := range p.styles {
		for _, kw := range keywords {
			if p.containsPhrase(normalized, kw) {
				matched = append(matched, name)
				break
			}
		}
	}
	return matched
}

// detectColorTone detects all color tones present in the normalized text.
func (p *LocalNLPParser) detectColorTone(normalized string) []string {
	var matched []string
	for name, keywords := range p.colorTones {
		for _, kw := range keywords {
			if p.containsPhrase(normalized, kw) {
				matched = append(matched, name)
				break
			}
		}
	}
	return matched
}

// deduplicateStrings removes duplicate strings from a slice.
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

// buildSemanticQuery constructs the final semantic query representation.
func (p *LocalNLPParser) buildSemanticQuery(intent dto.ParsedIntent, originalText string) string {
	var parts []string
	if len(intent.Occasion) > 0 {
		parts = append(parts, "occasion: "+strings.Join(intent.Occasion, ", "))
	}
	if len(intent.StyleTarget) > 0 {
		parts = append(parts, "style: "+strings.Join(intent.StyleTarget, ", "))
	}
	if len(intent.ColorTone) > 0 {
		parts = append(parts, "color tone: "+strings.Join(intent.ColorTone, ", "))
	}
	if len(intent.PositiveConstraints) > 0 {
		parts = append(parts, "constraints: "+strings.Join(intent.PositiveConstraints, ", "))
	}
	parts = append(parts, originalText)
	return strings.Join(parts, " | ")
}

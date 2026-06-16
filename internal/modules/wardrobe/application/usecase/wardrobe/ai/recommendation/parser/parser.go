// Package parser implements natural language query processing to extract wardrobe intent, occasion, and constraints.
package parser

import (
	"regexp"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/pkg/utils/sliceutils"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

// LocalNLPParser handles query processing and intent extraction using localized dictionaries.
type LocalNLPParser struct {
	occasions    map[string][]string
	styles       map[string][]string
	colorTones   map[string][]string
	weathers     map[string][]string
	negations    []string
	keywordRegex map[string]*regexp.Regexp
}

// NewLocalNLPParser builds a new LocalNLPParser with pre-compiled regex for all fashion keywords.
func NewLocalNLPParser() *LocalNLPParser {
	parser := &LocalNLPParser{
		occasions: map[string][]string{
			"casual": {"di choi", "dchoi", "dc", "dao pho", "cafe", "tu do", "dao", "hop lop", "casual", "hangout", "street", "di hoc", "hoc bai", "hoc"},
			"work":   {"di lam", "dlam", "dl", "cong so", "van phong", "hop", "thuyet trinh", "work", "office", "interview", "formal", "di cay", "cay do"},
			"date":   {"hen ho", "di choi voi nguoi yeu", "date", "romantic", "gap mat", "to tinh"},
			"party":  {"di tiec", "dtiec", "party", "dam cuoi", "sinh nhat", "sn", "bar", "club", "event", "su kien", "ky niem", "quay", "di quay", "an cuoi", "cuoi"},
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
			"cold":  {"lanh", "ret", "dong", "cold", "winter", "buot", "ret cam cam"},
			"cool":  {"mat", "se lanh", "thu", "cool", "autumn", "mat me"},
			"hot":   {"nong", "he", "nang", "hot", "summer", "nang gat", "nong chay mo"},
			"rainy": {"mua", "rainy", "uot", "mua phun", "am uot"},
		},
		negations: []string{"khong", "dung_neg", "tranh", "tieu cuc", "ne", "avoid", "no", "not", "without", "k", "ko", "kh", "ghet", "che", "thoi"},
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

// Parse extracts structured intent fields and constraints from free-text user queries.
func (p *LocalNLPParser) Parse(freeText string) dto.ParsedIntent {
	var intent dto.ParsedIntent
	if strings.TrimSpace(freeText) == "" {
		return intent
	}
	normalized := NormalizeText(freeText)

	var lexicalTerms, positiveConstraints, negativeConstraints []string
	var occasions, styles, colorTones []string
	var excludedStyles, excludedColorTones, excludedWeather []string

	sentences := p.splitSentences(normalized)
	for _, trimmed := range sentences {
		occasionMatches := p.detectMatches(trimmed, "occasion", p.occasions)
		styleMatches := p.detectMatches(trimmed, "style", p.styles)
		colorToneMatches := p.detectMatches(trimmed, "color-tone", p.colorTones)
		weatherMatches := p.detectMatches(trimmed, "weather", p.weathers)

		var dictionaryMatches []types.KeywordMatch
		dictionaryMatches = append(dictionaryMatches, occasionMatches...)
		dictionaryMatches = append(dictionaryMatches, styleMatches...)
		dictionaryMatches = append(dictionaryMatches, colorToneMatches...)
		dictionaryMatches = append(dictionaryMatches, weatherMatches...)

		for _, match := range occasionMatches {
			if !p.isNegatedMatch(trimmed, match) {
				occasions = append(occasions, match.Value)
				lexicalTerms = append(lexicalTerms, match.Value)
			}
		}
		for _, match := range styleMatches {
			if p.isNegatedMatch(trimmed, match) {
				excludedStyles = append(excludedStyles, match.Value)
				negativeConstraints = append(negativeConstraints, "avoid-style:"+match.Value)
			} else {
				styles = append(styles, match.Value)
				lexicalTerms = append(lexicalTerms, match.Value)
			}
		}
		for _, match := range colorToneMatches {
			if p.isNegatedMatch(trimmed, match) {
				excludedColorTones = append(excludedColorTones, match.Value)
				negativeConstraints = append(negativeConstraints, "avoid-color-tone:"+match.Value)
			} else {
				colorTones = append(colorTones, match.Value)
				lexicalTerms = append(lexicalTerms, match.Value)
			}
		}

		for _, match := range weatherMatches {
			if p.isNegatedMatch(trimmed, match) {
				excludedWeather = append(excludedWeather, match.Value)
				negativeConstraints = append(negativeConstraints, "avoid-weather:"+match.Value)
			} else {
				positiveConstraints = append(positiveConstraints, match.Value)
				lexicalTerms = append(lexicalTerms, match.Value)
			}
		}

		for _, match := range p.detectRawLexicalTerms(trimmed) {
			if overlapsAnyMatch(match, dictionaryMatches) {
				continue
			}
			if p.isNegatedMatch(trimmed, match) {
				negativeConstraints = append(negativeConstraints, "avoid-term:"+match.Value)
			} else {
				lexicalTerms = append(lexicalTerms, match.Value)
			}
		}
	}

	intent.Occasion = sliceutils.UniqueAndSortStrings(occasions)
	intent.StyleTarget = sliceutils.UniqueAndSortStrings(styles)
	intent.ColorTone = sliceutils.UniqueAndSortStrings(colorTones)
	intent.LexicalTerms = sliceutils.UniqueAndSortStrings(lexicalTerms)
	intent.PositiveConstraints = sliceutils.UniqueAndSortStrings(positiveConstraints)
	intent.NegativeConstraints = sliceutils.UniqueAndSortStrings(negativeConstraints)
	intent.ExcludedStyles = sliceutils.UniqueAndSortStrings(excludedStyles)
	intent.ExcludedColorTones = sliceutils.UniqueAndSortStrings(excludedColorTones)
	intent.ExcludedWeather = sliceutils.UniqueAndSortStrings(excludedWeather)
	return intent
}

// RemoveConflictingLexicalTerms removes raw query words that conflict with explicit search filters.
func (p *LocalNLPParser) RemoveConflictingLexicalTerms(intent dto.ParsedIntent) []string {
	explicitOccasions := stringutils.ToNormalizedSet(intent.Occasion)
	explicitStyles := stringutils.ToNormalizedSet(intent.StyleTarget)
	explicitColorTones := stringutils.ToNormalizedSet(intent.ColorTone)

	filtered := make([]string, 0, len(intent.LexicalTerms))
	for _, term := range intent.LexicalTerms {
		normalized := strings.ToLower(strings.TrimSpace(term))
		if normalized == "" {
			continue
		}
		if _, isOccasion := p.occasions[normalized]; isOccasion && len(explicitOccasions) > 0 && !explicitOccasions[normalized] {
			continue
		}
		if _, isStyle := p.styles[normalized]; isStyle && len(explicitStyles) > 0 && !explicitStyles[normalized] {
			continue
		}
		if _, isColorTone := p.colorTones[normalized]; isColorTone && len(explicitColorTones) > 0 && !explicitColorTones[normalized] {
			continue
		}
		filtered = append(filtered, normalized)
	}
	return sliceutils.UniqueAndSortStrings(filtered)
}

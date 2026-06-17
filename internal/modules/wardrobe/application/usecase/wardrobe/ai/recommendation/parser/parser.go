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

// LocalNLPParser xử lý phân tích cú pháp truy vấn ngôn ngữ tự nhiên và trích xuất ý định (intent) sử dụng các từ điển nội bộ.
type LocalNLPParser struct {
	occasions    map[string][]string
	styles       map[string][]string
	colorTones   map[string][]string
	weathers     map[string][]string
	negations    []string
	keywordRegex map[string]*regexp.Regexp
}

// NewLocalNLPParser khởi tạo một thực thể mới của [LocalNLPParser] với các cụm từ khóa thời trang được biên dịch Regex trước để tối ưu hiệu năng tìm kiếm.
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

// Parse trích xuất các trường ý định cấu trúc và ràng buộc từ truy vấn văn bản tự do của người dùng.
//
// Hành vi:
// 1. Chuẩn hóa chuỗi văn bản bằng [NormalizeText] để xử lý dấu tiếng Việt và lỗi gõ Telex.
// 2. Chia câu thành các mệnh đề nhỏ hơn thông qua [splitSentences].
// 3. Với mỗi mệnh đề, quét tìm các từ khóa thuộc dịp ([occasions]), phong cách ([styles]), tông màu ([colorTones]), và thời tiết ([weathers]).
// 4. Kiểm tra phủ định thông qua [isNegatedMatch] để phân loại xem từ khóa đó là thuộc tính tích cực (ví dụ: cần đồ ấm) hay tiêu cực (ví dụ: tránh đồ màu tối).
// 5. Quét các từ đơn thô còn lại không trùng với từ điển bằng [detectRawLexicalTerms] để đưa vào tập từ khóa lexical thô.
// 6. Trả về cấu trúc [ParsedIntent] chứa các lát chuỗi đã được lọc trùng và sắp xếp.
//
// Đầu vào mẫu:
//   freeText: "mình muốn tìm đồ đi chơi phố năng động nhưng không lấy màu tối"
//
// Đầu ra mẫu:
//   dto.ParsedIntent{
//       Occasion: []string{"casual"},
//       StyleTarget: []string{"streetwear"},
//       ExcludedColorTones: []string{"dark"},
//       NegativeConstraints: []string{"avoid-color-tone:dark"},
//       LexicalTerms: []string{"casual", "streetwear"},
//       ...
//   }
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

// RemoveConflictingLexicalTerms loại bỏ các từ khóa lexical thô bị xung đột với các bộ lọc tìm kiếm được người dùng chọn rõ ràng.
//
// Hành vi:
// Khi người dùng chọn một bộ lọc rõ ràng trên giao diện (ví dụ: chọn dịp "Làm việc" - work), hàm này sẽ kiểm tra danh sách từ khóa thô.
// Nếu trong danh sách từ khóa thô chứa các từ chỉ dịp khác (như "di choi" - casual) mà không khớp với dịp được chọn, từ đó sẽ bị lọc bỏ để tránh nhiễu kết quả.
//
// Đầu vào mẫu:
//   intent: dto.ParsedIntent{
//       Occasion: []string{"work"},
//       LexicalTerms: []string{"casual", "office"}
//   }
//
// Đầu ra mẫu:
//   []string{"office"} (từ "casual" bị loại bỏ vì xung đột với dịp "work" đã chọn)
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

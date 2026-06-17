package parser

import (
	"slices"
	"sort"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
)

// splitSentences phân tách một chuỗi đã chuẩn hóa thành danh sách các câu/mệnh đề nhỏ hơn dựa trên dấu câu và các từ nối phản chỉ (như "nhưng", "mà", "chứ").
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

// containsPhrase kiểm tra xem một cụm từ có xuất hiện trong văn bản hay không bằng cách sử dụng biểu thức chính quy (Regex) đã được biên dịch trước.
func (p *LocalNLPParser) containsPhrase(text, phrase string) bool {
	if !strings.Contains(text, phrase) {
		return false
	}
	if re, exists := p.keywordRegex[phrase]; exists {
		return re.MatchString(text)
	}
	return false
}

// detectMatches tìm kiếm và phát hiện các từ khóa thuộc về một danh mục cụ thể (ví dụ: occasion, style) trong văn bản,
// sau đó trả về danh sách các khớp từ khóa [KeywordMatch] được sắp xếp theo vị trí bắt đầu của chúng.
func (p *LocalNLPParser) detectMatches(text, category string, values map[string][]string) []types.KeywordMatch {
	valueNames := make([]string, 0, len(values))
	for name := range values {
		valueNames = append(valueNames, name)
	}
	sort.Strings(valueNames)

	var matches []types.KeywordMatch
	for _, value := range valueNames {
		keywords := values[value]
		for _, keyword := range keywords {
			if re, exists := p.keywordRegex[keyword]; exists {
				if loc := re.FindStringIndex(text); loc != nil {
					matches = append(matches, types.KeywordMatch{
						Category: category,
						Value:    value,
						Keyword:  keyword,
						Start:    loc[0],
						End:      loc[1],
						Source:   "local-dictionary",
					})
					break
				}
			}
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Start == matches[j].Start {
			return matches[i].Value < matches[j].Value
		}
		return matches[i].Start < matches[j].Start
	})
	return matches
}

// overlapsAnyMatch kiểm tra xem một khớp từ khóa mới có bị trùng lặp/chồng chéo vị trí (vị trí ký tự bắt đầu và kết thúc) với bất kỳ khớp từ khóa nào đã tồn tại trước đó hay không.
func overlapsAnyMatch(match types.KeywordMatch, existing []types.KeywordMatch) bool {
	for _, other := range existing {
		if match.Start < other.End && match.End > other.Start {
			return true
		}
	}
	return false
}

// detectRawLexicalTerms tách văn bản thành các từ đơn và lọc ra các từ hợp lệ (không phải từ dừng hay phủ định) làm từ khóa lexical thô (raw lexical terms).
func (p *LocalNLPParser) detectRawLexicalTerms(text string) []types.KeywordMatch {
	words := strings.Fields(text)
	matches := make([]types.KeywordMatch, 0, len(words))
	searchOffset := 0
	for _, word := range words {
		cleaned := strings.Trim(word, " \t\r\n,;.-")
		if !p.isLexicalTerm(cleaned) {
			searchOffset += len(word)
			continue
		}
		relativeStart := strings.Index(text[searchOffset:], word)
		if relativeStart < 0 {
			continue
		}
		start := searchOffset + relativeStart
		end := start + len(word)
		matches = append(matches, types.KeywordMatch{
			Category: "raw",
			Value:    cleaned,
			Keyword:  cleaned,
			Start:    start,
			End:      end,
			Source:   types.RetrievalTermSourceRaw,
		})
		searchOffset = end
	}
	return matches
}

// isNegatedMatch kiểm tra xem một từ khóa khớp có bị phủ định bởi các từ phủ định đứng trước nó (ví dụ: "không", "tránh", "đừng") trong một khoảng từ (window) giới hạn hay không.
func (p *LocalNLPParser) isNegatedMatch(text string, match types.KeywordMatch) bool {
	const negationWindowWords = 4
	prefix := strings.TrimSpace(text[:match.Start])
	if prefix == "" {
		return false
	}
	words := strings.Fields(prefix)

	start := max(len(words)-negationWindowWords, 0)

	window := strings.Join(words[start:], " ")

	for _, neg := range p.negations {
		if p.containsPhrase(window, neg) {
			return true
		}
	}
	return false
}

// isLexicalTerm xác định xem một từ đơn có hợp lệ để làm từ khóa tìm kiếm lexical hay không (độ dài > 2 ký tự, không phải từ dừng, và không phải từ phủ định).
func (p *LocalNLPParser) isLexicalTerm(word string) bool {
	word = strings.TrimSpace(word)
	if len(word) <= 2 {
		return false
	}
	if LexicalStopwords[word] {
		return false
	}
	return !slices.Contains(p.negations, word)
}

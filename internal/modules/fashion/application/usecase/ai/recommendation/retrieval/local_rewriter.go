// Package retrieval implements candidate retrieval, taxonomy term expansion, and lexical/semantic query rewriting.
package retrieval

import (
	"context"
	"sort"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

// LocalRecommendationQueryRewriter xây dựng các truy vấn tìm kiếm bằng quy tắc phân tích tĩnh (rule-based) cục bộ từ ý định của người dùng.
type LocalRecommendationQueryRewriter struct{}

// NewLocalRecommendationQueryRewriter khởi tạo thực thể [LocalRecommendationQueryRewriter].
func NewLocalRecommendationQueryRewriter() *LocalRecommendationQueryRewriter {
	return &LocalRecommendationQueryRewriter{}
}

// Rewrite xây dựng một truy vấn tìm kiếm nâng cao [RecommendationRetrievalQuery] từ ý định phân tích ngôn ngữ tự nhiên cục bộ.
//
// Hành vi:
// 1. Biên dịch danh sách từ khóa tìm kiếm qua [BuildSourceAwareRetrievalTerms].
// 2. Biên dịch danh sách từ khóa loại trừ qua [BuildSourceAwareExcludedTerms].
// 3. Chuẩn hóa và lọc trùng cả 2 danh sách từ khóa bằng [NormalizeRetrievalTerms].
// 4. Xây dựng các bộ lọc cứng (hard filters) như mùa vụ qua [BuildSeasonalityHardFilters].
// 5. Trả về cấu trúc [RecommendationRetrievalQuery] hoàn chỉnh làm tham số đầu vào cho tìm kiếm Elasticsearch.
//
// Đầu vào mẫu:
//
//	intent: dto.ParsedIntent{SemanticQuery: "style: casual", Occasion: []string{"casual"}}
//
// Đầu ra mẫu:
//
//	(types.RecommendationRetrievalQuery{RewriterSource: "local", ...}, nil)
func (LocalRecommendationQueryRewriter) Rewrite(_ context.Context, intent dto.ParsedIntent) (types.RecommendationRetrievalQuery, error) {
	terms := BuildSourceAwareRetrievalTerms(intent)
	excludedTerms := BuildSourceAwareExcludedTerms(intent)

	return types.RecommendationRetrievalQuery{
		SemanticQuery:  intent.SemanticQuery,
		LexicalTerms:   NormalizeRetrievalTerms(terms),
		ExcludedTerms:  NormalizeRetrievalTerms(excludedTerms),
		RewriterSource: "local",
		HardFilters: repositories.RecommendationHardFilters{
			Seasonality: BuildSeasonalityHardFilters(intent.PositiveConstraints),
		},
	}, nil
}

// BuildSourceAwareRetrievalTerms ánh xạ các thuộc tính ý định rõ ràng (như dịp, phong cách, tông màu) và từ khóa tự do thành danh sách các từ khóa tìm kiếm có nguồn gốc rõ ràng (source-aware retrieval terms).
func BuildSourceAwareRetrievalTerms(intent dto.ParsedIntent) []types.RetrievalTerm {
	terms := make([]types.RetrievalTerm, 0, len(intent.LexicalTerms)+len(intent.Occasion)+len(intent.StyleTarget)+len(intent.ColorTone)+len(intent.PositiveConstraints))
	for _, value := range intent.Occasion {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.StyleTarget {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.ColorTone {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.PositiveConstraints {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.LexicalTerms {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceRaw})
	}
	terms = append(terms, ExpandRecommendationLexicalRetrievalTerms(intent)...)
	return terms
}

// BuildSourceAwareExcludedTerms tổng hợp và mở rộng các từ khóa hoặc thuộc tính bị cấm/loại trừ để ngăn chặn việc gợi ý các trang phục mà người dùng muốn tránh.
func BuildSourceAwareExcludedTerms(intent dto.ParsedIntent) []types.RetrievalTerm {
	terms := make([]types.RetrievalTerm, 0, len(intent.ExcludedStyles)+len(intent.ExcludedColorTones)+len(intent.ExcludedWeather))

	// Excluded Styles
	for _, value := range intent.ExcludedStyles {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupStyle, intent.ExcludedStyles)...)

	// Excluded Color Tones
	for _, value := range intent.ExcludedColorTones {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupColorTone, intent.ExcludedColorTones)...)

	// Excluded Weather
	for _, value := range intent.ExcludedWeather {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupWeather, intent.ExcludedWeather)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupSeason, intent.ExcludedWeather)...)

	// Negative Constraints (Avoided Terms)
	for _, value := range ExtractAvoidTerms(intent.NegativeConstraints) {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}

	// Expand exclusions via taxonomyGroupExcluded
	var allExclusions []string
	allExclusions = append(allExclusions, intent.ExcludedStyles...)
	allExclusions = append(allExclusions, intent.ExcludedColorTones...)
	allExclusions = append(allExclusions, intent.ExcludedWeather...)
	allExclusions = append(allExclusions, ExtractAvoidTerms(intent.NegativeConstraints)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupExcluded, allExclusions)...)

	return terms
}

// NormalizeRetrievalTerms chuẩn hóa chữ thường, lọc bỏ các từ dừng (stopwords) và loại bỏ trùng lặp dựa trên thứ tự ưu tiên của nguồn (Dictionary > Raw > Taxonomy).
func NormalizeRetrievalTerms(terms []types.RetrievalTerm) []types.RetrievalTerm {
	byValue := map[string]types.RetrievalTerm{}
	for _, term := range terms {
		value := strings.ToLower(strings.TrimSpace(term.Value))
		if value == "" || parser.LexicalStopwords[value] {
			continue
		}
		term.Value = value
		existing, exists := byValue[value]
		if !exists || retrievalTermSourcePriority(term.Source) < retrievalTermSourcePriority(existing.Source) {
			byValue[value] = term
		}
	}

	normalized := make([]types.RetrievalTerm, 0, len(byValue))
	for _, term := range byValue {
		normalized = append(normalized, term)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Value < normalized[j].Value
	})
	return normalized
}

// ExpandRecommendationLexicalTerms mở rộng ý định đã phân tích thành một tập các từ khóa dạng lát cắt chuỗi đã chuẩn hóa.
func ExpandRecommendationLexicalTerms(intent dto.ParsedIntent) []string {
	terms := ExpandRecommendationLexicalRetrievalTerms(intent)
	values := make([]string, 0, len(terms))
	for _, term := range terms {
		values = append(values, term.Value)
	}
	return NormalizeTermSet(values)
}

// ExpandRecommendationLexicalRetrievalTerms thực hiện mở rộng phân loại (taxonomy expansion) cho tất cả các khía cạnh ý định (như dịp, phong cách, thời tiết, mùa, tông màu) và trả về danh sách [RetrievalTerm].
func ExpandRecommendationLexicalRetrievalTerms(intent dto.ParsedIntent) []types.RetrievalTerm {
	terms := make([]types.RetrievalTerm, 0)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupOccasion, intent.Occasion)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupStyle, intent.StyleTarget)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupWeather, intent.PositiveConstraints)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupSeason, intent.PositiveConstraints)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupColorTone, intent.ColorTone)...)
	return terms
}

// ExtractAvoidTerms phân tích cú pháp các chuỗi ràng buộc phủ định (ví dụ: "avoid-style:casual") để trích xuất ra từ khóa thô cần tránh (ví dụ: "casual").
func ExtractAvoidTerms(negativeConstraints []string) []string {
	var terms []string
	for _, constraint := range negativeConstraints {
		constraint = strings.TrimSpace(constraint)
		if after, ok := strings.CutPrefix(constraint, "avoid-term:"); ok {
			terms = append(terms, after)
			continue
		}
		if after, ok := strings.CutPrefix(constraint, "avoid-style:"); ok {
			terms = append(terms, after)
			continue
		}
		if after, ok := strings.CutPrefix(constraint, "avoid-color-tone:"); ok {
			terms = append(terms, after)
			continue
		}
		if after, ok := strings.CutPrefix(constraint, "avoid-weather:"); ok {
			terms = append(terms, after)
		}
	}
	return terms
}

// retrievalTermSourcePriority định nghĩa mức độ ưu tiên của nguồn từ khóa (Dictionary là cao nhất = 0, Taxonomy là thấp nhất = 2).
func retrievalTermSourcePriority(source string) int {
	switch source {
	case types.RetrievalTermSourceDictionary:
		return 0
	case types.RetrievalTermSourceRaw:
		return 1
	case types.RetrievalTermSourceTaxonomy:
		return 2
	default:
		return 3
	}
}

// Package retrieval implements candidate retrieval, taxonomy term expansion, and lexical/semantic query rewriting.
package retrieval

import (
	"sort"
	"strings"
)

// OuterwearCategoryTerms trả về danh sách các slug của danh mục áo khoác/đồ mặc ngoài có trong hệ thống phân loại (taxonomy).
func OuterwearCategoryTerms() []string {
	return ExpandTaxonomyTermValues(taxonomyGroupCategory, []string{"ao-khoac"})
}

// RecommendationAllowedCategorySlugs trả về danh sách tất cả các slug danh mục hợp lệ được định nghĩa trong hệ thống phân loại.
func RecommendationAllowedCategorySlugs() []string {
	slugs := make([]string, 0, len(recommendationTaxonomy[taxonomyGroupCategory]))
	for slug := range recommendationTaxonomy[taxonomyGroupCategory] {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

// RainyWeatherTerms trả về danh sách các từ khóa phân loại tương ứng với thời tiết mưa.
func RainyWeatherTerms() []string {
	return ExpandTaxonomyTermValues(taxonomyGroupWeather, []string{"rainy"})
}

// ColdLikeWeatherTerms trả về các từ khóa tương ứng với thời tiết lạnh/mùa đông và các chất liệu giữ ấm (ví dụ: len).
func ColdLikeWeatherTerms() []string {
	return NormalizeTermSet(append(
		append(ExpandTaxonomyTermValues(taxonomyGroupWeather, []string{"cold", "cool"}), ExpandTaxonomyTermValues(taxonomyGroupSeason, []string{"winter", "autumn"})...),
		"len",
	))
}

// HotLikeWeatherTerms trả về các từ khóa tương ứng với thời tiết nóng/mùa hè.
func HotLikeWeatherTerms() []string {
	return NormalizeTermSet(append(ExpandTaxonomyTermValues(taxonomyGroupWeather, []string{"hot"}), ExpandTaxonomyTermValues(taxonomyGroupSeason, []string{"summer"})...))
}

// BuildSeasonalityHardFilters tổng hợp một lát cắt các từ khóa mùa vụ từ thông tin ràng buộc đầu vào khớp với các định nghĩa mùa chuẩn của hệ thống.
func BuildSeasonalityHardFilters(values []string) []string {
	var seasons []string
	for _, value := range values {
		season := strings.ToLower(strings.TrimSpace(value))
		if recommendationSeasonalityTerms[season] {
			seasons = append(seasons, season)
		}
	}
	return NormalizeTermSet(seasons)
}

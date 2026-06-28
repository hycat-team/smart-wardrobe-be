package ranking

import (
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
)

var fallbackAllSeasonAliases = []string{
	"all",
	"all season",
	"all-season",
	"bon mua",
	"quanh nam",
}

var fallbackSeasonAliases = map[string][]string{
	"spring": {"spring", "xuan", "mua xuan"},
	"summer": {"summer", "he", "mua he"},
	"autumn": {"autumn", "fall", "thu", "mua thu"},
	"winter": {"winter", "dong", "mua dong", "lanh"},
}

// EnsureMinimumCandidatePool bổ sung thêm các ứng viên dự phòng (fallback) nếu bể ứng viên tìm kiếm lai (hybrid search) có kích thước nhỏ hơn mức yêu cầu tối thiểu.
//
// Hành vi:
//
//  1. Kiểm tra nếu số lượng ứng viên hiện tại đã đủ ([minimumPool]) hoặc bằng tổng số món đồ hoạt động trong tủ đồ, hàm sẽ trả về ngay.
//
//  2. Sử dụng map [taken] để theo dõi các món đồ đã có trong bể ứng viên để tránh trùng lặp.
//
//  3. Thực hiện thêm các ứng viên qua 3 mức độ nới lỏng tăng dần thông qua [appendFallbackCandidates]:
//     - Strict: Phải thỏa mãn bộ lọc cứng và chứa từ khóa tìm kiếm (lexical terms).
//     - Relaxed: Chỉ cần thỏa mãn bộ lọc cứng (không yêu cầu chứa từ khóa).
//     - General: Nới lỏng hoàn toàn các điều kiện.
//
//  4. Trả về bể ứng viên mới đã được mở rộng và số lượng ứng viên được thêm ở mỗi cấp độ.
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForRanking{...} (có 2 ứng viên)
//	activeItems: []*entities.WardrobeItem{...} (tổng 10 món đồ)
//	minimumPool: 5
//
// Đầu ra mẫu:
//
//	([]types.CandidateForRanking, types.FallbackCandidateCounts) chứa 5 ứng viên sau khi đã bổ sung 3 ứng viên dự phòng.
func EnsureMinimumCandidatePool(
	candidates []types.CandidateForRanking,
	activeItems []*entities.WardrobeItem,
	retrievalQuery types.RecommendationRetrievalQuery,
	minimumPool int,
) ([]types.CandidateForRanking, types.FallbackCandidateCounts) {
	rankingCandidates := make([]types.CandidateForRanking, 0, max(len(candidates), minimumPool))
	rankingCandidates = append(rankingCandidates, candidates...)

	if len(candidates) >= minimumPool || len(candidates) >= len(activeItems) {
		return rankingCandidates, types.FallbackCandidateCounts{}
	}

	taken := map[uuid.UUID]bool{}
	for _, candidate := range candidates {
		if candidate.Item != nil {
			taken[candidate.Item.ID] = true
		}
	}

	counts := types.FallbackCandidateCounts{}
	counts.Strict = appendFallbackCandidates(&rankingCandidates, activeItems, taken, retrievalQuery, candidateSourceStrictFallback, minimumPool)
	counts.Relaxed = appendFallbackCandidates(&rankingCandidates, activeItems, taken, retrievalQuery, candidateSourceRelaxedFallback, minimumPool)
	counts.General = appendFallbackCandidates(&rankingCandidates, activeItems, taken, retrievalQuery, candidateSourceGeneralFallback, minimumPool)

	return rankingCandidates, counts
}

// appendFallbackCandidates duyệt qua danh sách các món đồ hoạt động của người dùng để chọn lọc và đưa các món đồ đủ điều kiện vào bể ứng viên cho đến khi đạt đủ số lượng tối thiểu.
func appendFallbackCandidates(
	rankingCandidates *[]types.CandidateForRanking,
	activeItems []*entities.WardrobeItem,
	taken map[uuid.UUID]bool,
	retrievalQuery types.RecommendationRetrievalQuery,
	source string,
	minimumPool int,
) int {
	added := 0
	for _, item := range activeItems {
		if len(*rankingCandidates) >= minimumPool {
			break
		}
		if item == nil || taken[item.ID] {
			continue
		}
		if !FallbackCandidateEligible(item, retrievalQuery, source) {
			continue
		}

		*rankingCandidates = append(*rankingCandidates, types.CandidateForRanking{
			Item:            item,
			Source:          source,
			RetrievalScore:  0,
			RetrievalRank:   len(*rankingCandidates) + 1,
			RetrievalSource: source,
		})
		taken[item.ID] = true
		added++
	}
	return added
}

// FallbackCandidateEligible kiểm tra tính hợp lệ của một món đồ ứng viên dự phòng bằng cách kiểm tra các bộ lọc loại trừ nghiêm ngặt (hard filters, excluded terms và lexical terms tùy theo cấp độ).
func FallbackCandidateEligible(
	item *entities.WardrobeItem,
	retrievalQuery types.RecommendationRetrievalQuery,
	source string,
) bool {
	if !MatchesFallbackHardFilters(item, retrievalQuery.HardFilters) {
		return false
	}
	if MatchesFallbackExcludedTerms(item, retrievalQuery.ExcludedTerms) {
		return false
	}
	if source == candidateSourceStrictFallback && len(retrievalQuery.LexicalTerms) > 0 {
		return MatchesFallbackLexicalTerms(item, retrievalQuery.LexicalTerms)
	}
	return true
}

// MatchesFallbackHardFilters xác định xem món đồ có vượt qua các bộ lọc cứng về mùa (seasonality) và danh mục (category slugs) hay không.
func MatchesFallbackHardFilters(item *entities.WardrobeItem, filters repositories.RecommendationHardFilters) bool {
	if item == nil {
		return false
	}
	fashion := fashionForItem(item)
	if len(filters.Seasonality) > 0 {
		seasonality := ""
		if fashion != nil {
			seasonality = stringutils.GetString(fashion.Seasonality)
		}
		if !MatchesFallbackSeasonalityFilter(seasonality, filters.Seasonality) {
			return false
		}
	}
	if len(filters.CategorySlugs) > 0 {
		categorySlug := ""
		if category := item.FashionCategory(); category != nil {
			categorySlug = parser.NormalizeText(category.Slug)
		}
		matched := false
		for _, slug := range filters.CategorySlugs {
			if slug != "" && categorySlug == parser.NormalizeText(slug) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// MatchesFallbackSeasonalityFilter trả về true nếu mùa của món đồ có phần giao thoa/trùng khớp với mùa được yêu cầu trong truy vấn.
func MatchesFallbackSeasonalityFilter(seasonality string, requestedSeasons []string) bool {
	normalizedSeasonality := parser.NormalizeText(seasonality)
	if normalizedSeasonality == "" {
		return true
	}
	for _, alias := range fallbackAllSeasonAliases {
		if strings.Contains(normalizedSeasonality, parser.NormalizeText(alias)) {
			return true
		}
	}
	for _, season := range requestedSeasons {
		aliases := fallbackSeasonAliases[strings.ToLower(strings.TrimSpace(season))]
		if len(aliases) == 0 {
			aliases = []string{season}
		}
		for _, alias := range aliases {
			alias = parser.NormalizeText(alias)
			if alias != "" && strings.Contains(normalizedSeasonality, alias) {
				return true
			}
		}
	}
	return false
}

// MatchesFallbackExcludedTerms trả về true nếu mô tả hoặc thuộc tính của món đồ chứa các từ khóa nằm trong danh sách loại trừ nghiêm ngặt.
func MatchesFallbackExcludedTerms(item *entities.WardrobeItem, excludedTerms []types.RetrievalTerm) bool {
	if len(excludedTerms) == 0 {
		return false
	}
	document := FallbackSearchDocument(item)
	for _, term := range excludedTerms {
		normalized := parser.NormalizeText(term.Value)
		if normalized != "" && strings.Contains(document, normalized) {
			return true
		}
	}
	return false
}

// MatchesFallbackLexicalTerms trả về true nếu mô tả hoặc thuộc tính của món đồ có chứa ít nhất một trong các từ khóa tìm kiếm.
func MatchesFallbackLexicalTerms(item *entities.WardrobeItem, lexicalTerms []types.RetrievalTerm) bool {
	if len(lexicalTerms) == 0 {
		return true
	}
	document := FallbackSearchDocument(item)
	for _, term := range lexicalTerms {
		normalized := parser.NormalizeText(term.Value)
		if normalized != "" && strings.Contains(document, normalized) {
			return true
		}
	}
	return false
}

// FallbackSearchDocument nối tất cả các trường thông tin mô tả và thuộc tính của món đồ thành một chuỗi văn bản duy nhất để phục vụ cho việc quét từ khóa.
func FallbackSearchDocument(item *entities.WardrobeItem) string {
	if item == nil {
		return ""
	}
	fashion := fashionForItem(item)
	if fashion == nil {
		return ""
	}
	parts := []string{
		stringutils.GetString(fashion.Color),
		stringutils.GetString(fashion.Style),
		stringutils.GetString(fashion.Material),
		stringutils.GetString(fashion.Pattern),
		stringutils.GetString(fashion.Fit),
		stringutils.GetString(fashion.Seasonality),
		stringutils.GetString(fashion.Description),
	}
	if fashion.Category != nil {
		parts = append(parts, fashion.Category.Slug, fashion.Category.Name)
	}
	return parser.NormalizeText(strings.Join(parts, " "))
}

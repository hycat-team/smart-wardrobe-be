package ranking

import (
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/stringutils"

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
//	activeItems: []*wardrobe_entities.WardrobeItem{...} (tổng 10 món đồ)
//	minimumPool: 5
//
// Đầu ra mẫu:
//
//	([]types.CandidateForRanking, types.FallbackCandidateCounts) chứa 5 ứng viên sau khi đã bổ sung 3 ứng viên dự phòng.
func EnsureMinimumCandidatePool(
	candidates []types.CandidateForRanking,
	activeItems []*wardrobeDto.WardrobeItemContractDTO,
	retrievalQuery types.RecommendationRetrievalQuery,
	minimumPool int,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
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
	counts.Strict = appendFallbackCandidates(&rankingCandidates, activeItems, taken, retrievalQuery, types.CandidateSourceStrictFallback, minimumPool, fashionItemMap)
	counts.Relaxed = appendFallbackCandidates(&rankingCandidates, activeItems, taken, retrievalQuery, types.CandidateSourceRelaxedFallback, minimumPool, fashionItemMap)
	counts.General = appendFallbackCandidates(&rankingCandidates, activeItems, taken, retrievalQuery, types.CandidateSourceGeneralFallback, minimumPool, fashionItemMap)

	// Category Guarantee: Ensure candidates have (top AND bottom) OR fullbody
	if !HasValidOutfitStructure(rankingCandidates, fashionItemMap) {
		for _, item := range activeItems {
			if item == nil || taken[item.ID] {
				continue
			}
			if !FallbackCandidateEligible(item, retrievalQuery, types.CandidateSourceGeneralFallback, fashionItemMap) {
				continue
			}
			rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
				Item:            item,
				Source:          types.CandidateSourceGeneralFallback,
				RetrievalScore:  0,
				RetrievalRank:   len(rankingCandidates) + 1,
				RetrievalSource: string(types.CandidateSourceGeneralFallback),
			})
			taken[item.ID] = true
			if HasValidOutfitStructure(rankingCandidates, fashionItemMap) {
				break
			}
		}
	}

	return rankingCandidates, counts
}

// HasValidOutfitStructure checks whether the candidate pool contains (top AND bottom) OR fullbody.
func HasValidOutfitStructure(candidates []types.CandidateForRanking, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	hasFullbody := false
	hasTop := false
	hasBottom := false

	for _, c := range candidates {
		if c.Item == nil {
			continue
		}
		fi := fashionForItemFromMap(c.Item, fashionItemMap)
		if fi != nil && fi.Category != nil {
			role := mapper.CategorySlugToWearingRole(fi.Category.Slug)
			switch role {
			case "fullbody":
				hasFullbody = true
			case "top":
				hasTop = true
			case "bottom":
				hasBottom = true
			}
		}
	}

	return hasFullbody || (hasTop && hasBottom)
}

// HasValidOutfitStructurePrompts checks whether prompt candidates contain (top AND bottom) OR fullbody.
func HasValidOutfitStructurePrompts(candidates []types.CandidateForPrompt, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	hasFullbody := false
	hasTop := false
	hasBottom := false

	for _, c := range candidates {
		if c.Item == nil {
			continue
		}
		fi := fashionForItemFromMap(c.Item, fashionItemMap)
		if fi != nil && fi.Category != nil {
			role := mapper.CategorySlugToWearingRole(fi.Category.Slug)
			switch role {
			case "fullbody":
				hasFullbody = true
			case "top":
				hasTop = true
			case "bottom":
				hasBottom = true
			}
		}
	}

	return hasFullbody || (hasTop && hasBottom)
}

// appendFallbackCandidates duyệt qua danh sách các món đồ hoạt động của người dùng để chọn lọc và đưa các món đồ đủ điều kiện vào bể ứng viên cho đến khi đạt đủ số lượng tối thiểu.
func appendFallbackCandidates(
	rankingCandidates *[]types.CandidateForRanking,
	activeItems []*wardrobeDto.WardrobeItemContractDTO,
	taken map[uuid.UUID]bool,
	retrievalQuery types.RecommendationRetrievalQuery,
	source types.CandidateSource,
	minimumPool int,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) int {
	added := 0
	for _, item := range activeItems {
		if len(*rankingCandidates) >= minimumPool {
			break
		}
		if item == nil || taken[item.ID] {
			continue
		}
		if !FallbackCandidateEligible(item, retrievalQuery, source, fashionItemMap) {
			continue
		}

		*rankingCandidates = append(*rankingCandidates, types.CandidateForRanking{
			Item:            item,
			Source:          source,
			RetrievalScore:  0,
			RetrievalRank:   len(*rankingCandidates) + 1,
			RetrievalSource: string(source),
		})
		taken[item.ID] = true
		added++
	}
	return added
}

// FallbackCandidateEligible kiểm tra tính hợp lệ của một món đồ ứng viên dự phòng bằng cách kiểm tra các bộ lọc loại trừ nghiêm ngặt (hard filters, excluded terms và lexical terms tùy theo cấp độ).
func FallbackCandidateEligible(
	item *wardrobeDto.WardrobeItemContractDTO,
	retrievalQuery types.RecommendationRetrievalQuery,
	source types.CandidateSource,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) bool {
	if !MatchesFallbackHardFilters(item, retrievalQuery.HardFilters, fashionItemMap) {
		return false
	}
	if MatchesFallbackExcludedTerms(item, retrievalQuery.ExcludedTerms, fashionItemMap) {
		return false
	}
	if source == types.CandidateSourceStrictFallback && len(retrievalQuery.LexicalTerms) > 0 {
		return MatchesFallbackLexicalTerms(item, retrievalQuery.LexicalTerms, fashionItemMap)
	}
	return true
}

// MatchesFallbackHardFilters xác định xem món đồ có vượt qua các bộ lọc cứng về mùa (seasonality) và danh mục (category slugs) hay không.
func MatchesFallbackHardFilters(item *wardrobeDto.WardrobeItemContractDTO, filters repositories.RecommendationHardFilters, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	if item == nil {
		return false
	}
	fashion := fashionForItemFromMap(item, fashionItemMap)
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
		if fashion != nil && fashion.Category != nil {
			categorySlug = fashion.Category.Slug
		}
		matched := false
		for _, slug := range filters.CategorySlugs {
			if slug != "" && categorySlug == slug {
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
func MatchesFallbackExcludedTerms(item *wardrobeDto.WardrobeItemContractDTO, excludedTerms []types.RetrievalTerm, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	if len(excludedTerms) == 0 {
		return false
	}
	document := FallbackSearchDocument(item, fashionItemMap)
	for _, term := range excludedTerms {
		normalized := parser.NormalizeText(term.Value)
		if normalized != "" && strings.Contains(document, normalized) {
			return true
		}
	}
	return false
}

func MatchesFallbackLexicalTerms(item *wardrobeDto.WardrobeItemContractDTO, lexicalTerms []types.RetrievalTerm, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	if len(lexicalTerms) == 0 {
		return true
	}
	document := FallbackSearchDocument(item, fashionItemMap)
	for _, term := range lexicalTerms {
		normalized := parser.NormalizeText(term.Value)
		if normalized != "" && strings.Contains(document, normalized) {
			return true
		}
	}
	return false
}

// FallbackSearchDocument nối tất cả các trường thông tin mô tả và thuộc tính của món đồ thành một chuỗi văn bản duy nhất để phục vụ cho việc quét từ khóa.
func FallbackSearchDocument(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) string {
	if item == nil {
		return ""
	}
	fashion := fashionForItemFromMap(item, fashionItemMap)
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
	return parser.NormalizeText(strings.Join(parts, " "))
}

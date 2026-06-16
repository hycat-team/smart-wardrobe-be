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

// EnsureMinimumCandidatePool expands candidates using fallbacks if the hybrid search pool is smaller than required.
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

// FallbackCandidateEligible verifies eligibility of a fallback item against hard exclusions.
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

// MatchesFallbackHardFilters checks if the candidate matches the category/seasonality hard filters.
func MatchesFallbackHardFilters(item *entities.WardrobeItem, filters repositories.RecommendationHardFilters) bool {
	if item == nil {
		return false
	}
	if len(filters.Seasonality) > 0 {
		if !MatchesFallbackSeasonalityFilter(stringutils.GetString(item.Seasonality), filters.Seasonality) {
			return false
		}
	}
	if len(filters.CategorySlugs) > 0 {
		categorySlug := ""
		if item.Category != nil {
			categorySlug = parser.NormalizeText(item.Category.Slug)
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

// MatchesFallbackSeasonalityFilter returns true if a candidate's seasonality overlaps with requested query seasonality.
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

// MatchesFallbackExcludedTerms returns true if the candidate's metadata matches forbidden exclusion terms.
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

// MatchesFallbackLexicalTerms returns true if the candidate matches lexical keywords.
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

// FallbackSearchDocument concatenates a candidate's metadata fields for string matching.
func FallbackSearchDocument(item *entities.WardrobeItem) string {
	if item == nil {
		return ""
	}
	parts := []string{
		stringutils.GetString(item.Color),
		stringutils.GetString(item.Style),
		stringutils.GetString(item.Material),
		stringutils.GetString(item.Pattern),
		stringutils.GetString(item.Fit),
		stringutils.GetString(item.Seasonality),
		stringutils.GetString(item.Description),
	}
	if item.Category != nil {
		parts = append(parts, item.Category.Slug, item.Category.Name)
	}
	return parser.NormalizeText(strings.Join(parts, " "))
}

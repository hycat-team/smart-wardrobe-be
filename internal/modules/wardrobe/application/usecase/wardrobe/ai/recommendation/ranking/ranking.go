package ranking

import (
	"fmt"
	"math"
	"sort"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
)

// RankCandidates scores, ranks, and diversified the candidates pool.
func RankCandidates(
	candidates []types.CandidateForRanking,
	intent dto.ParsedIntent,
	retrievalQuery types.RecommendationRetrievalQuery,
	recentlyWornPenaltyDays int,
	longUnwornBonusDays int,
	minimumPool int,
) ([]types.CandidateForPrompt, types.RerankStats) {
	scored := make([]types.RankedCandidate, len(candidates))
	for i, candidate := range candidates {
		score, tags := ScoreCandidateItem(
			candidate.Item,
			intent,
			retrievalQuery,
			recentlyWornPenaltyDays,
			longUnwornBonusDays,
		)
		score += RetrievalScoreBonus(candidate)
		tags = append(tags,
			"candidate-source:"+candidate.Source,
			fmt.Sprintf("retrieval-rank:%d", candidate.RetrievalRank),
			fmt.Sprintf("retrieval-score:%.3f", candidate.RetrievalScore),
		)
		score -= FallbackSourcePenalty(candidate.Source)
		scored[i] = types.RankedCandidate{
			Item:          candidate.Item,
			Score:         score,
			Tags:          tags,
			Source:        candidate.Source,
			RetrievalRank: candidate.RetrievalRank,
		}
	}
	stats := RankedCandidateScoreStats(scored)

	sort.Slice(scored, func(i, j int) bool {
		return RankedCandidateLess(scored[i], scored[j])
	})

	limit := min(len(scored), minimumPool)
	if limit < 15 && len(scored) >= 15 {
		limit = 15
	}

	diversified := DiversifyRankedCandidates(scored, limit)
	stats.DiversifiedCount = len(diversified)
	final := make([]types.CandidateForPrompt, 0, len(diversified))
	for _, candidate := range diversified {
		final = append(final, types.CandidateForPrompt{
			Item: candidate.Item,
			Tags: candidate.Tags,
		})
	}

	return final, stats
}

// RetrievalScoreBonus awards priority points to hybrid candidates with higher baseline engine scores.
func RetrievalScoreBonus(candidate types.CandidateForRanking) float64 {
	if FallbackSourcePenalty(candidate.Source) > 0 {
		return 0
	}
	scoreComponent := math.Max(0, math.Min(candidate.RetrievalScore, 1.0)) * 0.2
	rankComponent := 0.0
	if candidate.RetrievalRank > 0 {
		rankComponent = math.Min(0.1, 0.1/float64(candidate.RetrievalRank))
	}
	return math.Min(0.25, scoreComponent+rankComponent)
}

// FallbackSourcePenalty applies ranking penalties based on how relaxed a candidate fallback source was.
func FallbackSourcePenalty(source string) float64 {
	switch source {
	case candidateSourceFallback, candidateSourceStrictFallback:
		return 0.1
	case candidateSourceRelaxedFallback:
		return 0.2
	case candidateSourceGeneralFallback:
		return 0.35
	default:
		return 0
	}
}

// RetrievalScoreStats gathers statistics on pre-reranking candidate scores.
func RetrievalScoreStats(candidates []types.CandidateForRanking) types.RerankStats {
	if len(candidates) == 0 {
		return types.RerankStats{}
	}
	minScore := candidates[0].RetrievalScore
	maxScore := candidates[0].RetrievalScore
	total := 0.0
	for _, candidate := range candidates {
		score := candidate.RetrievalScore
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
		total += score
	}
	return types.RerankStats{
		MinScore: minScore,
		MaxScore: maxScore,
		AvgScore: total / float64(len(candidates)),
	}
}

// RankedCandidateScoreStats gathers statistics on final candidate scores.
func RankedCandidateScoreStats(candidates []types.RankedCandidate) types.RerankStats {
	if len(candidates) == 0 {
		return types.RerankStats{}
	}
	minScore := candidates[0].Score
	maxScore := candidates[0].Score
	total := 0.0
	for _, candidate := range candidates {
		score := candidate.Score
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
		total += score
	}
	return types.RerankStats{
		MinScore: minScore,
		MaxScore: maxScore,
		AvgScore: total / float64(len(candidates)),
	}
}

// DiversifyRankedCandidates caps category-specific recommendations to promote variety in results.
func DiversifyRankedCandidates(scored []types.RankedCandidate, limit int) []types.RankedCandidate {
	if limit <= 0 || len(scored) <= limit {
		return scored
	}

	const categorySoftCap = 3
	categoryCounts := map[string]int{}
	selected := make([]types.RankedCandidate, 0, limit)
	deferred := make([]types.RankedCandidate, 0, len(scored))

	for _, candidate := range scored {
		category := "uncategorized"
		if candidate.Item != nil && candidate.Item.Category != nil && candidate.Item.Category.Slug != "" {
			category = candidate.Item.Category.Slug
		}
		if categoryCounts[category] < categorySoftCap && len(selected) < limit {
			selected = append(selected, candidate)
			categoryCounts[category]++
			continue
		}
		deferred = append(deferred, candidate)
	}

	for _, candidate := range deferred {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, candidate)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		return RankedCandidateLess(selected[i], selected[j])
	})
	return selected
}

// RankedCandidateLess compares two candidates by score, then rank, then UUID string comparison.
func RankedCandidateLess(left, right types.RankedCandidate) bool {
	if left.Score != right.Score {
		return left.Score > right.Score
	}
	leftRank := NormalizedRetrievalRank(left.RetrievalRank)
	rightRank := NormalizedRetrievalRank(right.RetrievalRank)
	if leftRank != rightRank {
		return leftRank < rightRank
	}
	return RankedCandidateID(left) < RankedCandidateID(right)
}

// NormalizedRetrievalRank converts ranking positions to handle unspecified values.
func NormalizedRetrievalRank(rank int) int {
	if rank <= 0 {
		return int(^uint(0) >> 1)
	}
	return rank
}

// RankedCandidateID returns candidate item ID as string.
func RankedCandidateID(candidate types.RankedCandidate) string {
	if candidate.Item == nil {
		return ""
	}
	return candidate.Item.ID.String()
}

package ranking

import (
	"fmt"
	"math"
	"sort"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
)

// RankCandidates tính toán điểm số, xếp hạng và đa dạng hóa bể ứng viên nhằm chọn ra các món đồ phù hợp nhất để gửi cho LLM/AI.
//
// Hành vi:
//  1. Duyệt qua từng ứng viên và gọi [ScoreCandidateItem] để chấm điểm dựa trên độ tương quan ý định, mức độ thường mặc và bonus không mặc lâu ngày.
//  2. Cộng thêm điểm thưởng từ công cụ tìm kiếm ngữ nghĩa qua [RetrievalScoreBonus].
//  3. Trừ đi điểm phạt nếu ứng viên đó thuộc các nguồn dự phòng nới lỏng qua [FallbackSourcePenalty].
//  4. Thu thập các thống kê điểm số qua [RankedCandidateScoreStats].
//  5. Sắp xếp danh sách ứng viên theo thứ tự điểm giảm dần, thứ tự tìm kiếm, và UUID.
//  6. Thực hiện đa dạng hóa danh sách qua [DiversifyRankedCandidates] để giới hạn số lượng đồ của cùng một danh mục (tránh trường hợp gợi ý quá nhiều áo hoặc quần cùng lúc).
//  7. Trả về lát cắt [CandidateForPrompt] chứa thông tin ứng viên và danh sách nhãn (tags) đi kèm, cùng với thống kê xếp hạng [RerankStats].
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForRanking{...}
//	intent: dto.ParsedIntent{...}
//
// Đầu ra mẫu:
//
//	([]types.CandidateForPrompt, types.RerankStats)
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

// RetrievalScoreBonus cộng điểm ưu tiên cho các ứng viên từ tìm kiếm lai (hybrid search) có điểm số cao từ Elasticsearch hoặc vị trí xếp hạng tìm kiếm tốt.
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

// FallbackSourcePenalty áp dụng điểm phạt xếp hạng đối với các ứng viên từ nguồn dự phòng, tùy thuộc vào mức độ nới lỏng của nguồn đó (General bị phạt nặng nhất).
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

// RetrievalScoreStats thu thập các thống kê (nhỏ nhất, lớn nhất, trung bình) trên điểm tìm kiếm gốc của các ứng viên trước khi xếp hạng lại.
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

// RankedCandidateScoreStats thu thập các thống kê trên điểm số xếp hạng lại (reranking) cuối cùng của các ứng viên.
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

// DiversifyRankedCandidates giới hạn số lượng món đồ tối đa của mỗi danh mục (mặc định là 3) để thúc đẩy tính đa dạng trong kết quả phối đồ, tránh tình trạng hiển thị quá nhiều sản phẩm cùng loại.
//
// Hành vi:
// 1. Duyệt qua danh sách đã sắp xếp điểm số, đếm số lượng món đồ cho từng Category Slug.
// 2. Nếu một danh mục chưa đạt soft cap (3 đồ), đưa món đồ vào danh sách được chọn ([selected]).
// 3. Nếu vượt quá soft cap hoặc danh sách chọn đã đủ số lượng giới hạn ([limit]), đưa món đồ vào danh sách hoãn lại ([deferred]).
// 4. Nếu danh sách chọn vẫn chưa đủ số lượng giới hạn sau vòng quét đầu tiên, tiếp tục lấy các món đồ từ danh sách hoãn lại đưa vào.
// 5. Sắp xếp lại danh sách đã chọn và trả về.
//
// Đầu vào mẫu:
//
//	scored: []types.RankedCandidate (chứa 6 cái áo, 2 cái quần)
//	limit: 5
//
// Đầu ra mẫu:
//
//	[]types.RankedCandidate (chứa 3 cái áo, 2 cái quần - tối đa hóa sự đa dạng)
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

// RankedCandidateLess so sánh hai ứng viên để phục vụ sắp xếp: ưu tiên điểm số lớn hơn, sau đó đến thứ tự xếp hạng tìm kiếm nhỏ hơn, cuối cùng so sánh chuỗi UUID.
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

// NormalizedRetrievalRank chuẩn hóa vị trí xếp hạng tìm kiếm, gán giá trị lớn nhất của kiểu int cho các vị trí không xác định (<= 0) để đẩy chúng về cuối danh sách.
func NormalizedRetrievalRank(rank int) int {
	if rank <= 0 {
		return int(^uint(0) >> 1)
	}
	return rank
}

// RankedCandidateID trả về chuỗi UUID ID của món đồ ứng viên.
func RankedCandidateID(candidate types.RankedCandidate) string {
	if candidate.Item == nil {
		return ""
	}
	return candidate.Item.ID.String()
}

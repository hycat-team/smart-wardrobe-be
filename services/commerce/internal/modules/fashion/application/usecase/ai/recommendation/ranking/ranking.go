package ranking

import (
	"fmt"
	"math"
	"sort"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"

	"github.com/google/uuid"
)

// RankCandidates tính toán điểm số, xếp hạng và đa dạng hóa bể ứng viên nhằm chọn ra các món đồ phù hợp nhất để gửi cho LLM/AI.
//
// Hành vi:
//  1. Duyệt qua từng ứng viên và gọi [ScoreCandidateItem] để chấm điểm dựa trên độ tương quan ý định, mức độ thường mặc và bonus không mặc lâu ngày.
//  2. Cộng thêm điểm thưởng từ công cụ tìm kiếm ngữ nghĩa qua [RetrievalScoreBonus].
//  3. Trừ đi điểm phạt nếu ứng viên đó thuộc các nguồn dự phòng nới lỏng qua [FallbackSourcePenalty].
//  4. Áp dụng ưu tiên vòng xoay qua [ApplyRecencyCooldown]: trong mỗi vai trò, nếu món đứng đầu
//     đang trong vùng làm mát mà có món ngoài vùng làm mát chất lượng tương đương (trong
//     `cooldownTolerance`), món ngoài vùng được đưa lên trước.
//  5. Thu thập các thống kê điểm số qua [RankedCandidateScoreStats].
//  6. Sắp xếp danh sách ứng viên theo thứ tự điểm giảm dần, thứ tự tìm kiếm, và UUID.
//  7. Thực hiện đa dạng hóa danh sách qua [DiversifyRankedCandidates] để giới hạn số lượng đồ của cùng một danh mục (tránh trường hợp gợi ý quá nhiều áo hoặc quần cùng lúc).
//  8. Trả về lát cắt [CandidateForPrompt] chứa thông tin ứng viên và danh sách nhãn (tags) đi kèm, cùng với thống kê xếp hạng [RerankStats].
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
	recency RecencyWindow,
	cooldownTolerance float64,
	minimumPool int,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) ([]types.CandidateForPrompt, types.RerankStats) {
	window := recency.Normalized()
	now := time.Now()
	scored := make([]types.RankedCandidate, len(candidates))
	for i, candidate := range candidates {
		score, tags := ScoreCandidateItem(
			candidate.Item,
			intent,
			retrievalQuery,
			window,
			fashionItemMap,
		)
		score += RetrievalScoreBonus(candidate)
		tags = append(tags,
			"candidate-source:"+string(candidate.Source),
			fmt.Sprintf("retrieval-rank:%d", candidate.RetrievalRank),
			fmt.Sprintf("retrieval-score:%.3f", candidate.RetrievalScore),
		)
		score -= FallbackSourcePenalty(candidate.Source)
		if candidate.ItemContext == outfititemcontext.BrandItem {
			tags = append(tags, "source:brand-item")
		}
		tags = prioritizeRecencyCooldownTag(tags)
		scored[i] = types.RankedCandidate{
			Item:          candidate.Item,
			Score:         score,
			OrderScore:    score,
			Tags:          tags,
			Source:        candidate.Source,
			RetrievalRank: candidate.RetrievalRank,
			ItemContext:   candidate.ItemContext,
			BrandItem:     candidate.BrandItem,
			Cooldown:      IsInRecencyCooldown(candidate.Item, window, now),
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		return RankedCandidateLess(scored[i], scored[j])
	})
	scored, cooldownSubstitutions := ApplyRecencyCooldown(scored, cooldownTolerance, fashionItemMap)

	limit := min(len(scored), minimumPool)
	if limit < 15 && len(scored) >= 15 {
		limit = 15
	}

	stats := RankedCandidateScoreStats(scored)
	stats.CooldownSubstitutions = cooldownSubstitutions
	diversified := DiversifyRankedCandidates(scored, limit, fashionItemMap)
	// Safeguard: diversification có thể cắt mất một role do giới hạn `limit`. Bổ sung
	// lại role còn thiếu từ pool đã chấm điểm để không mất cấu trúc bộ hợp lệ.
	diversified = ensureRankedOutfitStructure(diversified, scored, fashionItemMap)
	stats.DiversifiedCount = len(diversified)
	final := make([]types.CandidateForPrompt, 0, len(diversified))
	for _, candidate := range diversified {
		final = append(final, types.CandidateForPrompt{
			Item:        candidate.Item,
			Tags:        candidate.Tags,
			ItemContext: candidate.ItemContext,
			BrandItem:   candidate.BrandItem,
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
func FallbackSourcePenalty(source types.CandidateSource) float64 {
	switch source {
	case types.CandidateSourceFallback, types.CandidateSourceStrictFallback:
		return 0.1
	case types.CandidateSourceRelaxedFallback:
		return 0.2
	case types.CandidateSourceGeneralFallback:
		return 0.35
	default:
		return 0
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
func DiversifyRankedCandidates(scored []types.RankedCandidate, limit int, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) []types.RankedCandidate {
	if limit <= 0 || len(scored) <= limit {
		return scored
	}

	const categorySoftCap = 3
	categoryCounts := map[string]int{}
	selected := make([]types.RankedCandidate, 0, limit)
	deferred := make([]types.RankedCandidate, 0, len(scored))

	for _, candidate := range scored {
		category := "uncategorized"
		if candidate.Item != nil && candidate.Item.FashionItemID != nil {
			if fi := fashionItemMap[*candidate.Item.FashionItemID]; fi != nil && fi.Category != nil {
				category = fi.Category.Name
			}
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

// RankedCandidateLess so sánh hai ứng viên để phục vụ sắp xếp: ưu tiên khóa sắp xếp lớn hơn
// (OrderScore, mặc định bằng Score), sau đó đến thứ tự xếp hạng tìm kiếm nhỏ hơn, cuối cùng so sánh chuỗi UUID.
func RankedCandidateLess(left, right types.RankedCandidate) bool {
	leftOrder := RankedCandidateOrderScore(left)
	rightOrder := RankedCandidateOrderScore(right)
	if leftOrder != rightOrder {
		return leftOrder > rightOrder
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

// rankedWearingRole suy ra wearing role của một RankedCandidate từ metadata fashion.
func rankedWearingRole(candidate types.RankedCandidate, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
	if candidate.Item == nil || candidate.Item.FashionItemID == nil {
		return ""
	}
	fi := fashionItemMap[*candidate.Item.FashionItemID]
	if fi == nil || fi.Category == nil {
		return ""
	}
	return mapper.CategorySlugToWearingRole(fi.Category.Slug)
}

// hasValidRankedStructure kiểm tra pool đã xếp hạng có (top AND bottom) OR fullbody.
func hasValidRankedStructure(candidates []types.RankedCandidate, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	hasFullbody, hasTop, hasBottom := false, false, false
	for _, candidate := range candidates {
		switch rankedWearingRole(candidate, fashionItemMap) {
		case "fullbody":
			hasFullbody = true
		case "top":
			hasTop = true
		case "bottom":
			hasBottom = true
		}
	}
	return hasFullbody || (hasTop && hasBottom)
}

// missingRankedRoles trả về tập role còn thiếu để pool xếp hạng hợp lệ (nil nếu đã hợp lệ).
func missingRankedRoles(candidates []types.RankedCandidate, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) map[string]bool {
	if hasValidRankedStructure(candidates, fashionItemMap) {
		return nil
	}
	hasTop, hasBottom := false, false
	for _, candidate := range candidates {
		switch rankedWearingRole(candidate, fashionItemMap) {
		case "top":
			hasTop = true
		case "bottom":
			hasBottom = true
		}
	}
	need := map[string]bool{"fullbody": true}
	if !hasTop {
		need["top"] = true
	}
	if !hasBottom {
		need["bottom"] = true
	}
	return need
}

// optionalLayeringRoles là các vai trò lớp phụ (không bắt buộc cho cấu trúc bộ)
// vẫn phải được giữ lại trong pool sau khi đa dạng hóa, vì giới hạn theo danh mục
// (categorySoftCap) và giới hạn tổng số ứng viên dễ loại chúng khỏi danh sách.
var optionalLayeringRoles = []string{"headwear", "accessory", "outerwear"}

// missingOptionalRoles trả về tập vai trò lớp phụ còn thiếu trong pool đã chọn.
// Trả về nil nếu không còn vai trò lớp phụ nào cần bù.
func missingOptionalRoles(candidates []types.RankedCandidate, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) map[string]bool {
	present := make(map[string]bool, len(optionalLayeringRoles))
	for _, candidate := range candidates {
		if role := rankedWearingRole(candidate, fashionItemMap); role != "" {
			present[role] = true
		}
	}
	need := map[string]bool{}
	for _, role := range optionalLayeringRoles {
		if !present[role] {
			need[role] = true
		}
	}
	if len(need) == 0 {
		return nil
	}
	return need
}

// backfillMissingRoles bổ sung ứng viên còn thiếu cho các role trong need từ pool
// đã chấm điểm. Hàm không thay đổi các role đã hiện diện trong selected.
func backfillMissingRoles(
	selected []types.RankedCandidate,
	all []types.RankedCandidate,
	inSelected map[uuid.UUID]bool,
	need map[string]bool,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) []types.RankedCandidate {
	if len(need) == 0 {
		return selected
	}
	for _, candidate := range all {
		if len(need) == 0 {
			break
		}
		if candidate.Item == nil || inSelected[candidate.Item.ID] {
			continue
		}
		role := rankedWearingRole(candidate, fashionItemMap)
		if !need[role] {
			continue
		}
		selected = append(selected, candidate)
		inSelected[candidate.Item.ID] = true
		delete(need, role)
	}
	return selected
}

// ensureRankedOutfitStructure bảo đảm danh sách đã diversify vẫn giữ cấu trúc bộ hợp lệ
// bằng cách thêm lại các ứng viên còn thiếu role từ pool đã chấm điểm.
//
// Hành vi:
//  1. Bù các vai trò lõi còn thiếu (top/bottom/fullbody) theo [missingRankedRoles].
//  2. Bù các vai trò lớp phụ còn thiếu (headwear/accessory/outerwear) theo [missingOptionalRoles].
//     Bước này chạy kể cả khi cấu trúc lõi đã hợp lệ, vì đa dạng hóa có thể đã loại
//     các vai trò lớp phụ khỏi danh sách chỉ vì chạm trần theo danh mục.
func ensureRankedOutfitStructure(
	selected []types.RankedCandidate,
	all []types.RankedCandidate,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) []types.RankedCandidate {
	inSelected := map[uuid.UUID]bool{}
	for _, candidate := range selected {
		if candidate.Item != nil {
			inSelected[candidate.Item.ID] = true
		}
	}
	selected = backfillMissingRoles(selected, all, inSelected, missingRankedRoles(selected, fashionItemMap), fashionItemMap)
	selected = backfillMissingRoles(selected, all, inSelected, missingOptionalRoles(selected, fashionItemMap), fashionItemMap)

	sort.SliceStable(selected, func(i, j int) bool {
		return RankedCandidateLess(selected[i], selected[j])
	})
	return selected
}

package ranking

import (
	"sort"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"

	"github.com/google/uuid"
)

// defaultCooldownScoreTolerance mirrors rag.cooldown_score_tolerance: the score
// margin inside which a candidate outside the cooldown window outranks one
// inside it (033 FR-059).
const defaultCooldownScoreTolerance = 0.15

// recencyCooldownPromotionEpsilon lifts the promoted candidate just above the
// cooldown candidate it replaces, so the preference survives later stable sorts.
const recencyCooldownPromotionEpsilon = 1e-6

// ApplyRecencyCooldown đẩy món ngoài vùng làm mát lên trước món trong vùng làm mát khi chất lượng
// tương đương (033 FR-059), theo từng vai trò.
//
// Hành vi:
//  1. Xác định món đứng đầu của mỗi vai trò trong danh sách đã sắp xếp.
//  2. Nếu món đứng đầu đang trong vùng làm mát, tìm món ngoài vùng làm mát tốt nhất cùng vai trò.
//  3. Nếu chênh lệch điểm không vượt `tolerance`, gán khóa sắp xếp của món ngoài vùng cao hơn món
//     trong vùng một lượng rất nhỏ; điểm `Score` gốc giữ nguyên để thống kê vẫn trung thực.
//  4. Nếu mọi ứng viên của vai trò đều trong vùng làm mát (hoặc không có món tương đương), giữ nguyên
//     thứ tự — bộ đồ vẫn hoàn chỉnh (FR-060).
//
// Đầu vào mẫu:
//
//	scored: []types.RankedCandidate (áo A trong vùng làm mát 1.50, áo B ngoài vùng 1.40)
//	tolerance: 0.15
//
// Đầu ra mẫu:
//
//	([]types.RankedCandidate (B đứng trước A), 1)
func ApplyRecencyCooldown(
	scored []types.RankedCandidate,
	tolerance float64,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) ([]types.RankedCandidate, int) {
	if len(scored) == 0 {
		return scored, 0
	}
	if tolerance <= 0 {
		tolerance = defaultCooldownScoreTolerance
	}

	roleTop := make(map[string]int, len(scored))
	for i := range scored {
		role := cooldownRoleKey(scored[i], fashionItemMap)
		if _, seen := roleTop[role]; !seen {
			roleTop[role] = i
		}
	}

	substitutions := 0
	for role, topIndex := range roleTop {
		top := scored[topIndex]
		if !top.Cooldown {
			continue
		}
		bestOutside := -1
		for i := range scored {
			if i == topIndex || scored[i].Cooldown {
				continue
			}
			if cooldownRoleKey(scored[i], fashionItemMap) != role {
				continue
			}
			if bestOutside == -1 || RankedCandidateLess(scored[i], scored[bestOutside]) {
				bestOutside = i
			}
		}
		if bestOutside == -1 {
			continue
		}
		if top.Score-scored[bestOutside].Score > tolerance {
			continue
		}
		scored[bestOutside].OrderScore = RankedCandidateOrderScore(top) + recencyCooldownPromotionEpsilon
		substitutions++
	}
	if substitutions == 0 {
		return scored, 0
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return RankedCandidateLess(scored[i], scored[j])
	})
	return scored, substitutions
}

// prioritizeRecencyCooldownTag đưa nhãn [RecencyCooldownTag] lên đầu danh sách nhãn để nó không bị
// cắt bởi giới hạn `recommendation_tags_limit` khi prompt chỉ giữ một số nhãn đầu tiên.
func prioritizeRecencyCooldownTag(tags []string) []string {
	for i, tag := range tags {
		if tag != RecencyCooldownTag {
			continue
		}
		if i == 0 {
			return tags
		}
		reordered := make([]string, 0, len(tags))
		reordered = append(reordered, RecencyCooldownTag)
		reordered = append(reordered, tags[:i]...)
		reordered = append(reordered, tags[i+1:]...)
		return reordered
	}
	return tags
}

// cooldownRoleKey trả về khóa nhóm vai trò của một ứng viên. Món không xác định được vai trò
// được tách thành nhóm riêng để không bị hoán đổi với một món không liên quan.
func cooldownRoleKey(candidate types.RankedCandidate, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
	role := rankedWearingRole(candidate, fashionItemMap)
	if role == "" || role == "other" {
		return "item:" + RankedCandidateID(candidate)
	}
	return role
}

// RankedCandidateOrderScore trả về khóa sắp xếp của một ứng viên: OrderScore khi đã được gán,
// ngược lại là Score. Nhờ đó các lời gọi cũ chỉ đặt Score vẫn giữ nguyên hành vi.
func RankedCandidateOrderScore(candidate types.RankedCandidate) float64 {
	if candidate.OrderScore > 0 {
		return candidate.OrderScore
	}
	return candidate.Score
}

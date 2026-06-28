// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// CategorySlugToWearingRole maps database category slugs to wearing position roles.
func CategorySlugToWearingRole(slug string) string {
	switch strings.ToLower(slug) {
	case "ao":
		return "top"
	case "quan", "chan-vay":
		return "bottom"
	case "dam":
		return "fullbody"
	case "ao-khoac":
		return "outerwear"
	case "giay":
		return "footwear"
	case "mu":
		return "headwear"
	case "phu-kien":
		return "accessory"
	default:
		return "other"
	}
}

func parseAliasIndex(alias string) (int, bool) {
	alias = strings.TrimSpace(strings.ToUpper(alias))
	if len(alias) > 1 && alias[0] == 'A' {
		var index int
		if _, err := fmt.Sscanf(alias[1:], "%d", &index); err == nil {
			return index - 1, true
		}
	}
	return -1, false
}

// MapLLMResponseToGroups ánh xạ các gợi ý về ID và vai trò trang phục trong phản hồi của AI/LLM về lại các món đồ ứng viên thực tế trong tủ đồ của người dùng.
//
// Hành vi:
// 1. Tạo một bảng ánh xạ [candidateMap] từ UUID sang thực thể [WardrobeItem] để tra cứu nhanh.
// 2. Duyệt qua từng nhóm món đồ trong đề xuất của AI.
// 3. Phân giải món đồ chính qua [resolvePrimaryCandidate] dựa trên ID hoặc vai trò (role/category slug) làm dự phòng. Nếu không tìm thấy, bỏ qua nhóm này.
// 4. Phân giải tối đa 2 món đồ thay thế qua [resolveAlternativeCandidates].
// 5. Trả về danh sách các nhóm món đồ gợi ý hợp lệ [RecommendedItemGroup].
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForPrompt{...}
//	llmRes: types.LlmOutfitResponse{Items: []LlmOutfitItem{ {Role: "ao", PrimaryID: "uuid-ao-1", AlternativeIDs: []string{"uuid-ao-2"}} }}
//
// Đầu ra mẫu:
//
//	[]*dto.RecommendedItemGroup{ {Role: "ao", Primary: ... (ao-1), Alternatives: [... (ao-2)]} }
func MapLLMResponseToGroups(
	candidates []types.CandidateForPrompt,
	llmRes types.LlmOutfitResponse,
) []*dto.RecommendedItemGroup {
	candidateMap := map[uuid.UUID]*entities.WardrobeItem{}
	for _, candidate := range candidates {
		candidateMap[candidate.Item.ID] = candidate.Item
	}

	valid := make([]*dto.RecommendedItemGroup, 0)
	var hasFullbody bool

	for _, group := range llmRes.Items {
		role := strings.ToLower(group.Role)
		primary := resolvePrimaryCandidate(candidateMap, candidates, role, group.PrimaryID)
		if primary == nil {
			continue
		}

		actualRole := role
		if category := primary.FashionCategory(); category != nil {
			actualRole = CategorySlugToWearingRole(category.Slug)
		}

		if actualRole == "fullbody" {
			hasFullbody = true
		}

		valid = append(valid, &dto.RecommendedItemGroup{
			Role:         actualRole,
			Primary:      mapper.MapToWardrobeItemRes(primary),
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, actualRole, primary.ID, group.AlternativeIDs),
		})
	}

	if hasFullbody {
		filtered := make([]*dto.RecommendedItemGroup, 0, len(valid))
		for _, item := range valid {
			if item.Role != "top" && item.Role != "bottom" {
				filtered = append(filtered, item)
			}
		}
		valid = filtered
	}

	return valid
}

// resolvePrimaryCandidate tìm kiếm món đồ chính được đề xuất bởi AI. Nếu ID không khớp hoặc không hợp lệ, hàm sẽ lấy món đồ đầu tiên trong danh sách ứng viên có danh mục (category slug) khớp với vai trò (role) yêu cầu.
func resolvePrimaryCandidate(
	candidateMap map[uuid.UUID]*entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID string,
) *entities.WardrobeItem {
	if idx, ok := parseAliasIndex(primaryID); ok {
		if idx >= 0 && idx < len(candidates) {
			return candidates[idx].Item
		}
	}

	if id, err := uuid.Parse(primaryID); err == nil {
		if item := candidateMap[id]; item != nil {
			return item
		}
	}

	for _, candidate := range candidates {
		if category := candidate.Item.FashionCategory(); category != nil && CategorySlugToWearingRole(category.Slug) == role {
			return candidate.Item
		}
	}

	return nil
}

// resolveAlternativeCandidates tìm kiếm tối đa 2 món đồ thay thế cho vai trò được đề xuất.
//
// Hành vi:
// 1. Duyệt qua danh sách ID thay thế do AI trả về.
// 2. Parse UUID và lấy món đồ tương ứng từ [candidateMap]. Yêu cầu món đồ phải tồn tại, khác món đồ chính, và có danh mục trùng khớp với vai trò.
// 3. Nếu danh sách thay thế chưa đủ 2 món đồ, duyệt qua toàn bộ ứng viên thực tế để tìm thêm các món đồ cùng danh mục (khác đồ chính và chưa có trong danh sách thay thế) để lấp đầy.
// 4. Trả về danh sách món đồ thay thế dạng DTO [WardrobeItemRes].
//
// Đầu vào mẫu:
//
//	alternativeIDs: []string{"uuid-ao-2"}
//	role: "ao"
//	primaryID: "uuid-ao-1"
//
// Đầu ra mẫu:
//
//	[]*dto.WardrobeItemRes{...} (chứa tối đa 2 áo thay thế)
func resolveAlternativeCandidates(
	candidateMap map[uuid.UUID]*entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID uuid.UUID,
	alternativeIDs []string,
) []*dto.WardrobeItemRes {
	alternatives := make([]*dto.WardrobeItemRes, 0, 2)
	for _, altID := range alternativeIDs {
		var altItem *entities.WardrobeItem
		if idx, ok := parseAliasIndex(altID); ok {
			if idx >= 0 && idx < len(candidates) {
				altItem = candidates[idx].Item
			}
		}

		if altItem == nil {
			if altUUID, err := uuid.Parse(altID); err == nil {
				altItem = candidateMap[altUUID]
			}
		}

		if altItem == nil {
			continue
		}
		category := altItem.FashionCategory()
		if altItem.ID == primaryID || category == nil || CategorySlugToWearingRole(category.Slug) != role {
			continue
		}

		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(altItem))
		if len(alternatives) == 2 {
			return alternatives
		}
	}

	for _, candidate := range candidates {
		item := candidate.Item
		category := item.FashionCategory()
		if item.ID == primaryID || category == nil || CategorySlugToWearingRole(category.Slug) != role {
			continue
		}

		exists := false
		for _, alternative := range alternatives {
			if alternative.ID == item.ID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(item))
		if len(alternatives) == 2 {
			break
		}
	}

	return alternatives
}

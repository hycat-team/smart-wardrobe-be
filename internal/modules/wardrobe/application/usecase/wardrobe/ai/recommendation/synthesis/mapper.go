// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"strings"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

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
	for _, group := range llmRes.Items {
		role := strings.ToLower(group.Role)
		primary := resolvePrimaryCandidate(candidateMap, candidates, role, group.PrimaryID)
		if primary == nil {
			continue
		}

		valid = append(valid, &dto.RecommendedItemGroup{
			Role:         role,
			Primary:      mapper.MapToWardrobeItemRes(primary),
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, role, primary.ID, group.AlternativeIDs),
		})
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
	if id, err := uuid.Parse(primaryID); err == nil {
		if item := candidateMap[id]; item != nil {
			return item
		}
	}

	for _, candidate := range candidates {
		if candidate.Item.Category != nil && candidate.Item.Category.Slug == role {
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
		altUUID, err := uuid.Parse(altID)
		if err != nil {
			continue
		}

		altItem := candidateMap[altUUID]
		if altItem == nil || altItem.ID == primaryID || altItem.Category == nil || altItem.Category.Slug != role {
			continue
		}

		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(altItem))
		if len(alternatives) == 2 {
			return alternatives
		}
	}

	for _, candidate := range candidates {
		item := candidate.Item
		if item.ID == primaryID || item.Category == nil || item.Category.Slug != role {
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

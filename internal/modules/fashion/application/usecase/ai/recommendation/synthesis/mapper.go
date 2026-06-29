// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/outfititemcontext"
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
	promptCandidateMap := map[uuid.UUID]types.CandidateForPrompt{}
	for _, candidate := range candidates {
		candidateMap[candidate.Item.ID] = candidate.Item
		promptCandidateMap[candidate.Item.ID] = candidate
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

		primaryDTO := mapper.MapToWardrobeItemRes(primary)
		if cand, found := promptCandidateMap[primary.ID]; found {
			enrichWardrobeItemRes(primaryDTO, cand)
		}

		valid = append(valid, &dto.RecommendedItemGroup{
			Role:         actualRole,
			Primary:      primaryDTO,
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, promptCandidateMap, actualRole, primary.ID, group.AlternativeIDs),
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
func resolveAlternativeCandidates(
	candidateMap map[uuid.UUID]*entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	promptCandidateMap map[uuid.UUID]types.CandidateForPrompt,
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

		altDTO := mapper.MapToWardrobeItemRes(altItem)
		if cand, found := promptCandidateMap[altItem.ID]; found {
			enrichWardrobeItemRes(altDTO, cand)
		}
		alternatives = append(alternatives, altDTO)
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

		altDTO := mapper.MapToWardrobeItemRes(item)
		enrichWardrobeItemRes(altDTO, candidate)
		alternatives = append(alternatives, altDTO)
		if len(alternatives) == 2 {
			break
		}
	}

	return alternatives
}

func enrichWardrobeItemRes(res *dto.WardrobeItemRes, cand types.CandidateForPrompt) {
	if res == nil {
		return
	}
	res.ItemContext = string(cand.ItemContext)
	if res.ItemContext == "" {
		res.ItemContext = string(outfititemcontext.UserWardrobe)
	}
	if cand.ItemContext == outfititemcontext.BrandItem && cand.BrandItem != nil {
		brandName := ""
		if cand.BrandItem.Brand != nil {
			brandName = cand.BrandItem.Brand.Name
		}
		res.BrandItem = &dto.BrandItemBriefRes{
			ID:        cand.BrandItem.ID,
			BrandID:   cand.BrandItem.BrandID,
			BrandName: brandName,
			ItemType:  string(cand.BrandItem.ItemType),
			Name:      cand.BrandItem.Name,
			Price:     cand.BrandItem.Price,
		}
	}
}

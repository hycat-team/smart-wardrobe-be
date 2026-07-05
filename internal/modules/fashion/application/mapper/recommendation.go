package mapper

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobe_mapper "smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/outfititemcontext"
	wardrobe_entities "smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
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
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) []*dto.RecommendedItemGroup {
	candidateMap := map[uuid.UUID]*wardrobe_entities.WardrobeItem{}
	promptCandidateMap := map[uuid.UUID]types.CandidateForPrompt{}
	for _, candidate := range candidates {
		candidateMap[candidate.Item.ID] = candidate.Item
		promptCandidateMap[candidate.Item.ID] = candidate
	}

	valid := make([]*dto.RecommendedItemGroup, 0)
	var hasFullbody bool

	for _, group := range llmRes.Items {
		role := strings.ToLower(group.Role)
		primary := resolvePrimaryCandidate(candidateMap, candidates, role, group.PrimaryID, fashionItemMap)
		if primary == nil {
			continue
		}

		actualRole := role
		if fi := fashionItemMap[*primary.FashionItemID]; fi != nil && fi.CategoryName != nil {
			actualRole = CategorySlugToWearingRole(*fi.CategoryName)
		}

		if actualRole == "fullbody" {
			hasFullbody = true
		}

		primaryDTO := wardrobe_mapper.MapToWardrobeItemRes(primary, fashionItemMap[*primary.FashionItemID])
		if cand, found := promptCandidateMap[primary.ID]; found {
			enrichWardrobeItemRes(primaryDTO, cand)
		}

		valid = append(valid, &dto.RecommendedItemGroup{
			Role:         actualRole,
			Primary:      primaryDTO,
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, promptCandidateMap, actualRole, primary.ID, group.AlternativeIDs, fashionItemMap),
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
	candidateMap map[uuid.UUID]*wardrobe_entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID string,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) *wardrobe_entities.WardrobeItem {
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

	for _, c := range candidates {
		if c.Item.FashionItemID != nil {
			if fi := fashionItemMap[*c.Item.FashionItemID]; fi != nil && fi.CategoryName != nil {
				if CategorySlugToWearingRole(*fi.CategoryName) == role {
					return c.Item
				}
			}
		}
	}

	return nil
}

// resolveAlternativeCandidates tìm kiếm tối đa 2 món đồ thay thế cho vai trò được đề xuất.
func resolveAlternativeCandidates(
	candidateMap map[uuid.UUID]*wardrobe_entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	promptCandidateMap map[uuid.UUID]types.CandidateForPrompt,
	role string,
	primaryID uuid.UUID,
	alternativeIDs []string,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) []*wardrobeDto.WardrobeItemRes {
	alternatives := make([]*wardrobeDto.WardrobeItemRes, 0, 2)
	for _, altID := range alternativeIDs {
		var altItem *wardrobe_entities.WardrobeItem
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
		if altItem.ID == primaryID {
			continue
		}
		if !matchesRole(altItem, role, fashionItemMap) {
			continue
		}

		altDTO := wardrobe_mapper.MapToWardrobeItemRes(altItem, fashionItemMap[*altItem.FashionItemID])
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
		if item.ID == primaryID {
			continue
		}
		if !matchesRole(item, role, fashionItemMap) {
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

		altDTO := wardrobe_mapper.MapToWardrobeItemRes(item, fashionItemMap[*item.FashionItemID])
		enrichWardrobeItemRes(altDTO, candidate)
		alternatives = append(alternatives, altDTO)
		if len(alternatives) == 2 {
			break
		}
	}

	return alternatives
}

func matchesRole(item *wardrobe_entities.WardrobeItem, role string, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	if item.FashionItemID != nil {
		if fi := fashionItemMap[*item.FashionItemID]; fi != nil && fi.CategoryName != nil {
			return CategorySlugToWearingRole(*fi.CategoryName) == role
		}
	}
	return true
}

func enrichWardrobeItemRes(res *wardrobeDto.WardrobeItemRes, cand types.CandidateForPrompt) {
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
		res.BrandItem = &wardrobeDto.BrandItemBriefRes{
			ID:        cand.BrandItem.ID,
			BrandID:   cand.BrandItem.BrandID,
			BrandName: brandName,
			ItemType:  string(cand.BrandItem.ItemType),
			Name:      cand.BrandItem.Name,
			Price:     cand.BrandItem.Price,
		}
	}
}

package mapper

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

// CategorySlugToWearingRole maps database category slugs to wearing position roles.
// Slug được chuẩn hoá về chữ thường và cắt khoảng trắng thừa, khớp với cách
// itemCategorySlug ở luồng HSL fallback xử lý, tránh việc slug bẩn âm thầm rơi về "other".
func CategorySlugToWearingRole(slug string) string {
	switch strings.ToLower(strings.TrimSpace(slug)) {
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

func MapToRecommendedItemRes(item *wardrobeDto.WardrobeItemContractDTO, fi *dto.FashionItemContractDTO, cand types.CandidateForPrompt) *dto.RecommendedItemRes {
	if item == nil {
		return nil
	}
	var fRes *dto.RecommendedFashionItemBriefRes
	if fi != nil {
		fRes = &dto.RecommendedFashionItemBriefRes{
			ID:       fi.ID,
			Category: fi.Category,
			ImageUrl: fi.ImageUrl,
			Color:    fi.Color,
			ColorHex: fi.ColorHex,
			Style:    fi.Style,
		}
	}
	res := &dto.RecommendedItemRes{
		ID:          item.ID,
		FashionItem: fRes,
	}
	res.ItemContext = string(cand.ItemContext)
	if res.ItemContext == "" {
		res.ItemContext = string(outfititemcontext.UserWardrobe)
	}
	if cand.ItemContext == outfititemcontext.BrandItem && cand.BrandItem != nil {
		res.BrandItem = &shared_dto.BrandItemBriefRes{
			ID:          cand.BrandItem.ID,
			BrandID:     cand.BrandItem.BrandID,
			BrandName:   cand.BrandItem.BrandName,
			ProductCode: cand.BrandItem.ProductCode,
			ItemType:    string(cand.BrandItem.ItemType),
			Name:        cand.BrandItem.Name,
			Price:       cand.BrandItem.Price,
			Status:      string(cand.BrandItem.Status),
		}
	}
	return res
}

func FashionForItemFromMap(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) *dto.FashionItemContractDTO {
	if item == nil {
		return nil
	}
	if item.FashionItemID != nil {
		return fashionItemMap[*item.FashionItemID]
	}
	return nil
}

// coreSeparatesRoles là các vai trò lõi chỉ cần bù lại khi outfit không dùng đầm,
// vì đầm (fullbody) đã thay thế chúng trong cấu trúc một-piece.
var coreSeparatesRoles = []string{"top", "bottom"}

// optionalLayerRoles là các vai trò lớp phụ luôn được bù lại vì chúng độc lập với
// cấu trúc lõi và trung lập với thời tiết. outerwear cố ý không nằm trong nhóm này:
// nó phụ thuộc thời tiết, mùa và ý định layering nên để lớp scoring và prompt quyết định.
var optionalLayerRoles = []string{"headwear", "accessory"}

// MapLLMResponseToGroups ánh xạ các gợi ý về ID và vai trò trang phục trong phản hồi của AI/LLM về lại các món đồ ứng viên thực tế trong tủ đồ của người dùng.
//
// Hành vi:
//  1. Ánh xạ từng mục LLM trả về thành group thực tế, gán lại vai trò theo danh mục trong DB.
//  2. Loại bỏ top và bottom nếu outfit có đầm (fullbody).
//  3. Khử trùng lặp vai trò, giữ lại group đầu tiên cho mỗi vai trò.
//  4. Bù lại top và bottom còn thiếu khi outfit không dùng đầm.
//  5. Bù lại headwear và accessory còn thiếu.
//
// Bước 4 và 5 bảo đảm các vai trò này luôn xuất hiện khi tủ đồ có ứng viên phù hợp,
// kể cả khi LLM bỏ sót, nên response không bao giờ thiếu nhóm mũ/phụ kiện một cách ngẫu nhiên.
func MapLLMResponseToGroups(
	candidates []types.CandidateForPrompt,
	llmRes types.LlmOutfitResponse,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) []*dto.RecommendedItemGroup {
	candidateMap, promptCandidateMap := indexPromptCandidates(candidates)

	groups, hasFullbody := mapLLMItemsToGroups(llmRes, candidates, candidateMap, promptCandidateMap, fashionItemMap)
	groups = dropTopBottomWhenFullbody(groups, hasFullbody)
	groups = dedupeGroupsByRole(groups)

	if !hasFullbody {
		groups = appendMissingRoleGroups(groups, candidates, candidateMap, promptCandidateMap, fashionItemMap, coreSeparatesRoles)
	}
	return appendMissingRoleGroups(groups, candidates, candidateMap, promptCandidateMap, fashionItemMap, optionalLayerRoles)
}

// indexPromptCandidates lập chỉ mục ứng viên theo ID để tra cứu nhanh khi ánh xạ phản hồi LLM.
func indexPromptCandidates(candidates []types.CandidateForPrompt) (map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO, map[uuid.UUID]types.CandidateForPrompt) {
	candidateMap := make(map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO, len(candidates))
	promptCandidateMap := make(map[uuid.UUID]types.CandidateForPrompt, len(candidates))
	for _, candidate := range candidates {
		candidateMap[candidate.Item.ID] = candidate.Item
		promptCandidateMap[candidate.Item.ID] = candidate
	}
	return candidateMap, promptCandidateMap
}

// mapLLMItemsToGroups ánh xạ từng mục LLM sang group thực tế. Vai trò luôn được gán lại theo
// danh mục trong DB nên giá trị role do LLM khai báo không quyết định kết quả cuối cùng.
func mapLLMItemsToGroups(
	llmRes types.LlmOutfitResponse,
	candidates []types.CandidateForPrompt,
	candidateMap map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO,
	promptCandidateMap map[uuid.UUID]types.CandidateForPrompt,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) ([]*dto.RecommendedItemGroup, bool) {
	valid := make([]*dto.RecommendedItemGroup, 0, len(llmRes.Items))
	var hasFullbody bool

	for _, group := range llmRes.Items {
		role := strings.ToLower(group.Role)
		primary := resolvePrimaryCandidate(candidateMap, candidates, role, group.PrimaryID, fashionItemMap)
		if primary == nil {
			continue
		}

		actualRole := role
		fashionItemRes := FashionForItemFromMap(primary, fashionItemMap)
		if fashionItemRes != nil && fashionItemRes.Category != nil {
			actualRole = CategorySlugToWearingRole(fashionItemRes.Category.Slug)
		}
		if actualRole == "fullbody" {
			hasFullbody = true
		}

		cand, _ := promptCandidateMap[primary.ID]
		valid = append(valid, &dto.RecommendedItemGroup{
			Role:         actualRole,
			Primary:      MapToRecommendedItemRes(primary, fashionItemRes, cand),
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, promptCandidateMap, actualRole, primary.ID, group.AlternativeIDs, fashionItemMap),
		})
	}

	return valid, hasFullbody
}

// dropTopBottomWhenFullbody loại bỏ top và bottom khi outfit đã có đầm.
func dropTopBottomWhenFullbody(groups []*dto.RecommendedItemGroup, hasFullbody bool) []*dto.RecommendedItemGroup {
	if !hasFullbody {
		return groups
	}
	filtered := make([]*dto.RecommendedItemGroup, 0, len(groups))
	for _, group := range groups {
		if group.Role != "top" && group.Role != "bottom" {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

// dedupeGroupsByRole giữ lại group đầu tiên cho mỗi vai trò.
func dedupeGroupsByRole(groups []*dto.RecommendedItemGroup) []*dto.RecommendedItemGroup {
	seenRoles := make(map[string]bool, len(groups))
	deduped := make([]*dto.RecommendedItemGroup, 0, len(groups))
	for _, group := range groups {
		if seenRoles[group.Role] {
			continue
		}
		seenRoles[group.Role] = true
		deduped = append(deduped, group)
	}
	return deduped
}

// appendMissingRoleGroups bù lại group cho mừng vai trò trong roles còn thiếu, lấy ứng viên
// phù hợp đầu tiên tìm được trong pool. Vai trò không có ứng viên sẽ bị bỏ qua.
func appendMissingRoleGroups(
	groups []*dto.RecommendedItemGroup,
	candidates []types.CandidateForPrompt,
	candidateMap map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO,
	promptCandidateMap map[uuid.UUID]types.CandidateForPrompt,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
	roles []string,
) []*dto.RecommendedItemGroup {
	present := make(map[string]bool, len(groups))
	for _, group := range groups {
		present[group.Role] = true
	}

	for _, role := range roles {
		if present[role] {
			continue
		}
		candidate := findCandidateByRole(candidates, role, fashionItemMap)
		if candidate == nil {
			continue
		}
		fashionItemRes := FashionForItemFromMap(candidate.Item, fashionItemMap)
		groups = append(groups, &dto.RecommendedItemGroup{
			Role:         role,
			Primary:      MapToRecommendedItemRes(candidate.Item, fashionItemRes, *candidate),
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, promptCandidateMap, role, candidate.Item.ID, nil, fashionItemMap),
		})
		present[role] = true
	}

	return groups
}

func findCandidateByRole(candidates []types.CandidateForPrompt, role string, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) *types.CandidateForPrompt {
	for i := range candidates {
		if fi := FashionForItemFromMap(candidates[i].Item, fashionItemMap); fi != nil && fi.Category != nil {
			if CategorySlugToWearingRole(fi.Category.Slug) == role {
				return &candidates[i]
			}
		}
	}
	return nil
}

// resolvePrimaryCandidate tìm kiếm món đồ chính được đề xuất bởi AI.
func resolvePrimaryCandidate(
	candidateMap map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID string,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) *wardrobeDto.WardrobeItemContractDTO {
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
		if fi := FashionForItemFromMap(c.Item, fashionItemMap); fi != nil && fi.Category != nil {
			if CategorySlugToWearingRole(fi.Category.Slug) == role {
				return c.Item
			}
		}
	}

	return nil
}

// resolveAlternativeCandidates tìm kiếm tối đa 2 món đồ thay thế cho vai trò được đề xuất.
func resolveAlternativeCandidates(
	candidateMap map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO,
	candidates []types.CandidateForPrompt,
	promptCandidateMap map[uuid.UUID]types.CandidateForPrompt,
	role string,
	primaryID uuid.UUID,
	alternativeIDs []string,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) []*dto.RecommendedItemRes {
	alternatives := make([]*dto.RecommendedItemRes, 0, 2)
	for _, altID := range alternativeIDs {
		var altItem *wardrobeDto.WardrobeItemContractDTO
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

		fashionItemRes := FashionForItemFromMap(altItem, fashionItemMap)
		altDTO := MapToRecommendedItemRes(altItem, fashionItemRes, promptCandidateMap[altItem.ID])
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

		fashionItemRes := FashionForItemFromMap(item, fashionItemMap)
		altDTO := MapToRecommendedItemRes(item, fashionItemRes, candidate)
		alternatives = append(alternatives, altDTO)
		if len(alternatives) == 2 {
			break
		}
	}

	return alternatives
}

func matchesRole(item *wardrobeDto.WardrobeItemContractDTO, role string, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	if fi := FashionForItemFromMap(item, fashionItemMap); fi != nil && fi.Category != nil {
		return CategorySlugToWearingRole(fi.Category.Slug) == role
	}
	return true
}

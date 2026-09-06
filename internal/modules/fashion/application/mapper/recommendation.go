package mapper

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/outfititemcontext"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
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

// MapLLMResponseToGroups ánh xạ các gợi ý về ID và vai trò trang phục trong phản hồi của AI/LLM về lại các món đồ ứng viên thực tế trong tủ đồ của người dùng.
func MapLLMResponseToGroups(
	candidates []types.CandidateForPrompt,
	llmRes types.LlmOutfitResponse,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) []*dto.RecommendedItemGroup {
	candidateMap := map[uuid.UUID]*wardrobeDto.WardrobeItemContractDTO{}
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
		fashionItemRes := FashionForItemFromMap(primary, fashionItemMap)
		if fashionItemRes != nil && fashionItemRes.Category != nil {
			actualRole = CategorySlugToWearingRole(fashionItemRes.Category.Slug)
		}

		if actualRole == "fullbody" {
			hasFullbody = true
		}

		cand, _ := promptCandidateMap[primary.ID]
		primaryDTO := MapToRecommendedItemRes(primary, fashionItemRes, cand)

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

	// Deduplicate by role: only keep the first group for each role
	seenRoles := make(map[string]bool)
	deduped := make([]*dto.RecommendedItemGroup, 0, len(valid))
	for _, g := range valid {
		if seenRoles[g.Role] {
			continue
		}
		seenRoles[g.Role] = true
		deduped = append(deduped, g)
	}

	if !hasFullbody {
		hasTop := false
		hasBottom := false
		for _, g := range deduped {
			switch g.Role {
			case "top":
				hasTop = true
			case "bottom":
				hasBottom = true
			}
		}

		if !hasTop {
			if topCand := findCandidateByRole(candidates, "top", fashionItemMap); topCand != nil {
				fashionItemRes := FashionForItemFromMap(topCand.Item, fashionItemMap)
				topDTO := MapToRecommendedItemRes(topCand.Item, fashionItemRes, *topCand)
				deduped = append(deduped, &dto.RecommendedItemGroup{
					Role:         "top",
					Primary:      topDTO,
					Alternatives: resolveAlternativeCandidates(candidateMap, candidates, promptCandidateMap, "top", topCand.Item.ID, nil, fashionItemMap),
				})
				hasTop = true
			}
		}

		if !hasBottom {
			if bottomCand := findCandidateByRole(candidates, "bottom", fashionItemMap); bottomCand != nil {
				fashionItemRes := FashionForItemFromMap(bottomCand.Item, fashionItemMap)
				bottomDTO := MapToRecommendedItemRes(bottomCand.Item, fashionItemRes, *bottomCand)
				deduped = append(deduped, &dto.RecommendedItemGroup{
					Role:         "bottom",
					Primary:      bottomDTO,
					Alternatives: resolveAlternativeCandidates(candidateMap, candidates, promptCandidateMap, "bottom", bottomCand.Item.ID, nil, fashionItemMap),
				})
				hasBottom = true
			}
		}
	}

	return deduped
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

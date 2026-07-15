package mapper

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/outfititemcontext"

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

func cloneWardrobeItemRes(item *wardrobeDto.WardrobeItemRes, fi *dto.FashionItemRes) *wardrobeDto.WardrobeItemRes {
	if item == nil {
		return nil
	}
	var fRes *wardrobeDto.FashionItemRes
	if fi != nil {
		fRes = &wardrobeDto.FashionItemRes{
			ID:              fi.ID,
			CategoryID:      fi.CategoryID,
			CategoryName:    fi.CategoryName,
			CategorySlug:    fi.CategorySlug,
			ImageUrl:        fi.ImageUrl,
			Color:           fi.Color,
			ColorHex:        fi.ColorHex,
			ColorHue:        fi.ColorHue,
			ColorSaturation: fi.ColorSaturation,
			ColorLightness:  fi.ColorLightness,
			Style:           fi.Style,
			Material:        fi.Material,
			Pattern:         fi.Pattern,
			Fit:             fi.Fit,
			Seasonality:     fi.Seasonality,
			Description:     fi.Description,
			CreatedAt:       fi.CreatedAt,
			UpdatedAt:       fi.UpdatedAt,
		}
	} else if item.FashionItem != nil {
		fRes = &wardrobeDto.FashionItemRes{
			ID: item.FashionItem.ID,
		}
	}
	return &wardrobeDto.WardrobeItemRes{
		ID:          item.ID,
		UserID:      item.UserID,
		Price:       item.Price,
		Status:      item.Status,
		LastUsedAt:  item.LastUsedAt,
		IsDeleted:   item.IsDeleted,
		FashionItem: fRes,
	}
}

// MapLLMResponseToGroups ánh xạ các gợi ý về ID và vai trò trang phục trong phản hồi của AI/LLM về lại các món đồ ứng viên thực tế trong tủ đồ của người dùng.
func MapLLMResponseToGroups(
	candidates []types.CandidateForPrompt,
	llmRes types.LlmOutfitResponse,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) []*dto.RecommendedItemGroup {
	candidateMap := map[uuid.UUID]*wardrobeDto.WardrobeItemRes{}
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
		if primary.FashionItem != nil {
			if fi := fashionItemMap[primary.FashionItem.ID]; fi != nil && fi.CategorySlug != nil {
				actualRole = CategorySlugToWearingRole(*fi.CategorySlug)
			}
		}

		if actualRole == "fullbody" {
			hasFullbody = true
		}

		var fashionItemRes *dto.FashionItemRes
		if primary.FashionItem != nil {
			fashionItemRes = fashionItemMap[primary.FashionItem.ID]
		}
		primaryDTO := cloneWardrobeItemRes(primary, fashionItemRes)
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

// resolvePrimaryCandidate tìm kiếm món đồ chính được đề xuất bởi AI.
func resolvePrimaryCandidate(
	candidateMap map[uuid.UUID]*wardrobeDto.WardrobeItemRes,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID string,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) *wardrobeDto.WardrobeItemRes {
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
		if c.Item.FashionItem != nil {
			if fi := fashionItemMap[c.Item.FashionItem.ID]; fi != nil && fi.CategoryName != nil {
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
	candidateMap map[uuid.UUID]*wardrobeDto.WardrobeItemRes,
	candidates []types.CandidateForPrompt,
	promptCandidateMap map[uuid.UUID]types.CandidateForPrompt,
	role string,
	primaryID uuid.UUID,
	alternativeIDs []string,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) []*wardrobeDto.WardrobeItemRes {
	alternatives := make([]*wardrobeDto.WardrobeItemRes, 0, 2)
	for _, altID := range alternativeIDs {
		var altItem *wardrobeDto.WardrobeItemRes
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

		var fashionItemRes *dto.FashionItemRes
		if altItem.FashionItem != nil {
			fashionItemRes = fashionItemMap[altItem.FashionItem.ID]
		}
		altDTO := cloneWardrobeItemRes(altItem, fashionItemRes)
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

		var fashionItemRes *dto.FashionItemRes
		if item.FashionItem != nil {
			fashionItemRes = fashionItemMap[item.FashionItem.ID]
		}
		altDTO := cloneWardrobeItemRes(item, fashionItemRes)
		enrichWardrobeItemRes(altDTO, candidate)
		alternatives = append(alternatives, altDTO)
		if len(alternatives) == 2 {
			break
		}
	}

	return alternatives
}

func matchesRole(item *wardrobeDto.WardrobeItemRes, role string, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	if item.FashionItem != nil {
		if fi := fashionItemMap[item.FashionItem.ID]; fi != nil && fi.CategoryName != nil {
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
		res.BrandItem = &wardrobeDto.BrandItemBriefRes{
			ID:        cand.BrandItem.ID,
			BrandID:   cand.BrandItem.BrandID,
			BrandName: cand.BrandItem.BrandName,
			ItemType:  string(cand.BrandItem.ItemType),
			Name:      cand.BrandItem.Name,
			Price:     cand.BrandItem.Price,
		}
	}
}

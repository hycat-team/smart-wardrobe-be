package ranking

import (
	"math"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// ============================================================================
// HSL Color Compatibility Helpers
// ============================================================================

// isNeutral kiểm tra xem màu sắc của món đồ có phải là màu trung tính (ví dụ: đen, trắng, xám hoặc màu có độ bão hòa rất thấp) dựa trên độ sáng (lightness) và độ bão hòa (saturation).
func isNeutral(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	fashion := fashionForItemFromMap(item, fashionItemMap)
	if fashion == nil || fashion.ColorLightness == nil || fashion.ColorSaturation == nil {
		return true
	}

	lightness := *fashion.ColorLightness
	saturation := *fashion.ColorSaturation
	return lightness <= 10.0 || lightness >= 90.0 || saturation <= 10.0
}

// colorsMatch kiểm tra xem hai món đồ có màu sắc phối hợp hài hòa với nhau theo thuật toán HSL hay không.
func colorsMatch(item1, item2 *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	if isNeutral(item1, fashionItemMap) || isNeutral(item2, fashionItemMap) {
		return true
	}
	fashion1 := fashionForItemFromMap(item1, fashionItemMap)
	fashion2 := fashionForItemFromMap(item2, fashionItemMap)
	if fashion1 == nil || fashion2 == nil || fashion1.ColorHue == nil || fashion2.ColorHue == nil {
		return true
	}

	h1 := *fashion1.ColorHue
	h2 := *fashion2.ColorHue
	diff := math.Abs(h1 - h2)
	deltaH := math.Min(diff, 360.0-diff)

	if deltaH < 30.0 {
		lightness1, lightness2 := 0.0, 0.0
		if fashion1.ColorLightness != nil {
			lightness1 = *fashion1.ColorLightness
		}
		if fashion2.ColorLightness != nil {
			lightness2 = *fashion2.ColorLightness
		}
		if math.Abs(lightness1-lightness2) >= 15.0 {
			return true
		}
	}

	return deltaH >= 165.0 && deltaH <= 195.0
}

func fashionForItemFromMap(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) *dto.FashionItemRes {
	return mapper.FashionForItemFromMap(item, fashionItemMap)
}

// ============================================================================
// Category & Role Normalization Helpers
// ============================================================================

func itemCategorySlug(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) string {
	if fi := fashionForItemFromMap(item, fashionItemMap); fi != nil && fi.Category != nil {
		slug := strings.ToLower(strings.TrimSpace(fi.Category.Slug))
		if strings.Contains(slug, "ao-khoac") {
			return "ao-khoac"
		}
		if strings.Contains(slug, "chan-vay") {
			return "chan-vay"
		}
		if strings.Contains(slug, "ao") {
			return "ao"
		}
		if strings.Contains(slug, "quan") {
			return "quan"
		}
		if strings.Contains(slug, "giay") {
			return "giay"
		}
		if strings.Contains(slug, "dam") {
			return "dam"
		}
		return slug
	}
	return ""
}

func roleForItem(item *wardrobeDto.WardrobeItemContractDTO, fallback string, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) string {
	if slug := itemCategorySlug(item, fashionItemMap); slug != "" {
		return mapper.CategorySlugToWearingRole(slug)
	}
	return mapper.CategorySlugToWearingRole(fallback)
}

// ============================================================================
// DTO Mapping & Group Builder Helpers
// ============================================================================

// buildItemGroup xây dựng cấu trúc [RecommendedItemGroup] cho một vai trò phối đồ cụ thể, bao gồm món đồ chính và tối đa 2 món đồ thay thế từ danh sách.
func buildItemGroup(
	role string,
	primary *wardrobeDto.WardrobeItemContractDTO,
	all []*wardrobeDto.WardrobeItemContractDTO,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
	candidateMap map[uuid.UUID]types.CandidateForPrompt,
) *dto.RecommendedItemGroup {
	alternatives := make([]*dto.RecommendedItemRes, 0, 2)
	for _, item := range all {
		if item.ID == primary.ID {
			continue
		}
		if len(alternatives) >= 2 {
			break
		}
		fi := fashionForItemFromMap(item, fashionItemMap)
		cand := candidateMap[item.ID]
		alternatives = append(alternatives, mapper.MapToRecommendedItemRes(item, fi, cand))
	}

	primaryFI := fashionForItemFromMap(primary, fashionItemMap)
	cand := candidateMap[primary.ID]
	return &dto.RecommendedItemGroup{
		Role:         role,
		Primary:      mapper.MapToRecommendedItemRes(primary, primaryFI, cand),
		Alternatives: alternatives,
	}
}

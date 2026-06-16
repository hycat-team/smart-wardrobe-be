// Package ranking implements rule-based candidate scoring, fallback candidate expansion, and list diversification.
package ranking

import (
	"math"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func isNeutral(item *entities.WardrobeItem) bool {
	if item.ColorLightness == nil || item.ColorSaturation == nil {
		return true
	}

	lightness := *item.ColorLightness
	saturation := *item.ColorSaturation
	return lightness <= 10.0 || lightness >= 90.0 || saturation <= 10.0
}

func colorsMatch(item1, item2 *entities.WardrobeItem) bool {
	if isNeutral(item1) || isNeutral(item2) {
		return true
	}
	if item1.ColorHue == nil || item2.ColorHue == nil {
		return true
	}

	h1 := *item1.ColorHue
	h2 := *item2.ColorHue
	diff := math.Abs(h1 - h2)
	deltaH := math.Min(diff, 360.0-diff)

	if deltaH < 30.0 {
		lightness1, lightness2 := 0.0, 0.0
		if item1.ColorLightness != nil {
			lightness1 = *item1.ColorLightness
		}
		if item2.ColorLightness != nil {
			lightness2 = *item2.ColorLightness
		}
		if math.Abs(lightness1-lightness2) >= 15.0 {
			return true
		}
	}

	return deltaH >= 165.0 && deltaH <= 195.0
}

// RunLocalHSLMatching executes the HSL color matching algorithm to construct a fallback recommended outfit.
func RunLocalHSLMatching(
	candidates []types.CandidateForPrompt,
	input dto.RecommendOutfitReq,
) *dto.RecommendedOutfitRes {
	var tops, bottoms, shoes, outerwears, dresses, accessories []*entities.WardrobeItem

	for _, candidate := range candidates {
		item := candidate.Item
		if item.Category == nil {
			accessories = append(accessories, item)
			continue
		}

		switch item.Category.Slug {
		case "ao":
			tops = append(tops, item)
		case "quan":
			bottoms = append(bottoms, item)
		case "giay":
			shoes = append(shoes, item)
		case "ao-khoac":
			outerwears = append(outerwears, item)
		case "vay":
			dresses = append(dresses, item)
		default:
			accessories = append(accessories, item)
		}
	}

	isCold := false
	if input.Weather != nil {
		weather := strings.ToLower(*input.Weather)
		if strings.Contains(weather, "lạnh") ||
			strings.Contains(weather, "mát") ||
			strings.Contains(weather, "cold") ||
			strings.Contains(weather, "cool") ||
			strings.Contains(weather, "mưa") ||
			strings.Contains(weather, "rain") {
			isCold = true
		}
	}

	var items []*dto.RecommendedItemGroup
	if len(dresses) > 0 && len(shoes) > 0 && (len(tops) == 0 || len(bottoms) == 0) {
		primaryDress := dresses[0]
		primaryShoes := shoes[0]
		for _, shoe := range shoes {
			if colorsMatch(primaryDress, shoe) {
				primaryShoes = shoe
				break
			}
		}

		items = append(items, buildItemGroup("vay", primaryDress, dresses))
		items = append(items, buildItemGroup("giay", primaryShoes, shoes))

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryDress, outerwear) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup("ao-khoac", primaryOuter, outerwears))
		}
	} else if len(tops) > 0 && len(bottoms) > 0 {
		primaryTop, primaryBottom := tops[0], bottoms[0]
		found := false
		for _, top := range tops {
			for _, bottom := range bottoms {
				if colorsMatch(top, bottom) {
					primaryTop = top
					primaryBottom = bottom
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		items = append(items, buildItemGroup("ao", primaryTop, tops))
		items = append(items, buildItemGroup("quan", primaryBottom, bottoms))

		if len(shoes) > 0 {
			primaryShoes := shoes[0]
			for _, shoe := range shoes {
				if colorsMatch(primaryTop, shoe) || colorsMatch(primaryBottom, shoe) {
					primaryShoes = shoe
					break
				}
			}
			items = append(items, buildItemGroup("giay", primaryShoes, shoes))
		}

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryTop, outerwear) || colorsMatch(primaryBottom, outerwear) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup("ao-khoac", primaryOuter, outerwears))
		}
	} else {
		if len(dresses) > 0 {
			items = append(items, buildItemGroup("vay", dresses[0], dresses))
		} else {
			if len(tops) > 0 {
				items = append(items, buildItemGroup("ao", tops[0], tops))
			}
			if len(bottoms) > 0 {
				items = append(items, buildItemGroup("quan", bottoms[0], bottoms))
			}
		}
		if len(shoes) > 0 {
			items = append(items, buildItemGroup("giay", shoes[0], shoes))
		}
	}

	explanation := "Hệ thống đã tự động phối đồ bằng thuật toán phân tích màu sắc HSL dự phòng, đảm bảo tính hài hòa tương đồng hoặc tương phản của trang phục."
	if isCold {
		explanation += " Do thời tiết lạnh/ẩm, hệ thống đã phối thêm áo khoác giữ ấm."
	}

	return &dto.RecommendedOutfitRes{
		Title:       "Bộ phối màu hài hòa",
		Explanation: explanation,
		Items:       items,
		IsFallback:  true,
	}
}

func buildItemGroup(
	role string,
	primary *entities.WardrobeItem,
	all []*entities.WardrobeItem,
) *dto.RecommendedItemGroup {
	alternatives := make([]*dto.WardrobeItemRes, 0, 2)
	for _, item := range all {
		if item.ID == primary.ID {
			continue
		}
		if len(alternatives) >= 2 {
			break
		}
		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(item))
	}

	return &dto.RecommendedItemGroup{
		Role:         role,
		Primary:      mapper.MapToWardrobeItemRes(primary),
		Alternatives: alternatives,
	}
}

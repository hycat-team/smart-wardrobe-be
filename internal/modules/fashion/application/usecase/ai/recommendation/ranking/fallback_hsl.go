// Package ranking implements rule-based candidate scoring, fallback candidate expansion, and list diversification.
package ranking

import (
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// RunLocalHSLMatching thực hiện thuật toán so khớp màu sắc HSL cục bộ để tạo ra một gợi ý phối đồ dự phòng (fallback) khi AI bị lỗi.
//
// Hành vi:
//  1. Phân loại các món đồ ứng viên thành các nhóm danh mục: Áo (tops), Quần và chân váy (bottoms), Giày (shoes), Áo khoác (outerwears), Đầm (dresses), Phụ kiện (accessories).
//  2. Xác định thời tiết có lạnh hay mưa không dựa trên thông tin đầu vào.
//  3. Nếu có váy và giày, chọn váy làm món đồ chính, sau đó duyệt danh sách giày để tìm đôi giày có màu phối hợp tốt nhất qua [colorsMatch].
//  4. Nếu có áo và quần, duyệt tìm cặp áo và quần đầu tiên phối hợp màu sắc hài hòa nhất làm đồ chính. Sau đó tìm giày và áo khoác (nếu trời lạnh) có màu hợp với áo hoặc quần chính.
//  5. Nếu không tìm được cặp phối hợp, chọn ngẫu nhiên món đồ đầu tiên của mỗi danh mục.
//  6. Trả về cấu trúc [RecommendedOutfitRes] kèm tiêu đề và giải thích lý do phối màu.
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForPrompt{...}
//	input: dto.RecommendOutfitReq{Weather: pointer to "lạnh"}
//
// Đầu ra mẫu:
//
//	*dto.RecommendedOutfitRes{Title: "Bộ phối màu hài hòa", IsFallback: true, ...}
func RunLocalHSLMatching(
	candidates []types.CandidateForPrompt,
	input dto.RecommendOutfitReq,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) *dto.RecommendedOutfitRes {
	var tops, bottoms, shoes, outerwears, dresses, accessories []*wardrobeDto.WardrobeItemRes

	for _, candidate := range candidates {
		item := candidate.Item
		categorySlug := itemCategorySlug(item, fashionItemMap)

		switch categorySlug {
		case "ao":
			tops = append(tops, item)
		case "quan", "chan-vay":
			bottoms = append(bottoms, item)
		case "giay":
			shoes = append(shoes, item)
		case "ao-khoac":
			outerwears = append(outerwears, item)
		case "dam":
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
			if colorsMatch(primaryDress, shoe, fashionItemMap) {
				primaryShoes = shoe
				break
			}
		}

		items = append(items, buildItemGroup(roleForItem(primaryDress, "dam", fashionItemMap), primaryDress, dresses, fashionItemMap))
		items = append(items, buildItemGroup(mapper.CategorySlugToWearingRole("giay"), primaryShoes, shoes, fashionItemMap))

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryDress, outerwear, fashionItemMap) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryOuter, "ao-khoac", fashionItemMap), primaryOuter, outerwears, fashionItemMap))
		}
	} else if len(tops) > 0 && len(bottoms) > 0 {
		primaryTop, primaryBottom := tops[0], bottoms[0]
		found := false
		for _, top := range tops {
			for _, bottom := range bottoms {
				if colorsMatch(top, bottom, fashionItemMap) {
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

		items = append(items, buildItemGroup(mapper.CategorySlugToWearingRole("ao"), primaryTop, tops, fashionItemMap))
		items = append(items, buildItemGroup(roleForItem(primaryBottom, "quan", fashionItemMap), primaryBottom, bottoms, fashionItemMap))

		if len(shoes) > 0 {
			primaryShoes := shoes[0]
			for _, shoe := range shoes {
				if colorsMatch(primaryTop, shoe, fashionItemMap) || colorsMatch(primaryBottom, shoe, fashionItemMap) {
					primaryShoes = shoe
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryShoes, "giay", fashionItemMap), primaryShoes, shoes, fashionItemMap))
		}

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryTop, outerwear, fashionItemMap) || colorsMatch(primaryBottom, outerwear, fashionItemMap) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryOuter, "ao-khoac", fashionItemMap), primaryOuter, outerwears, fashionItemMap))
		}
	} else {
		if len(dresses) > 0 {
			items = append(items, buildItemGroup(roleForItem(dresses[0], "dam", fashionItemMap), dresses[0], dresses, fashionItemMap))
		} else {
			if len(tops) > 0 {
				items = append(items, buildItemGroup(roleForItem(tops[0], "ao", fashionItemMap), tops[0], tops, fashionItemMap))
			}
			if len(bottoms) > 0 {
				items = append(items, buildItemGroup(roleForItem(bottoms[0], "quan", fashionItemMap), bottoms[0], bottoms, fashionItemMap))
			}
		}
		if len(shoes) > 0 {
			items = append(items, buildItemGroup(roleForItem(shoes[0], "giay", fashionItemMap), shoes[0], shoes, fashionItemMap))
		}
	}

	explanation := msgFallbackHslExplain
	if isCold {
		explanation += msgFallbackHslExplainAddOuter
	}

	return &dto.RecommendedOutfitRes{
		Title:       msgFallbackHslTitle,
		Explanation: explanation,
		Items:       items,
		IsFallback:  true,
	}
}

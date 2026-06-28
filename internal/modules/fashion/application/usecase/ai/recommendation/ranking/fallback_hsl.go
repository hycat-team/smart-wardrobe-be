// Package ranking implements rule-based candidate scoring, fallback candidate expansion, and list diversification.
package ranking

import (
	"math"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// isNeutral kiểm tra xem màu sắc của món đồ có phải là màu trung tính (ví dụ: đen, trắng, xám hoặc màu có độ bão hòa rất thấp) dựa trên độ sáng (lightness) và độ bão hòa (saturation).
func isNeutral(item *entities.WardrobeItem) bool {
	fashion := fashionForItem(item)
	if fashion == nil || fashion.ColorLightness == nil || fashion.ColorSaturation == nil {
		return true
	}

	lightness := *fashion.ColorLightness
	saturation := *fashion.ColorSaturation
	return lightness <= 10.0 || lightness >= 90.0 || saturation <= 10.0
}

// colorsMatch kiểm tra xem hai món đồ có màu sắc phối hợp hài hòa với nhau theo thuật toán HSL hay không.
//
// Quy tắc phối màu:
// 1. Nếu một trong hai món đồ có màu trung tính (neutral), chúng luôn phối hợp được với nhau.
// 2. Nếu hai màu có góc màu (Hue) lệch nhau dưới 30 độ (tương đồng), chúng chỉ được coi là phối hợp khi có độ sáng (Lightness) lệch nhau ít nhất 15% (tránh bị chìm màu).
// 3. Nếu hai màu đối diện nhau trên bánh xe màu (góc màu lệch từ 165 đến 195 độ - tương phản), chúng phối hợp tốt với nhau.
//
// Đầu vào mẫu:
//
//	item1: WardrobeItem có Hue = 200 (xanh lam)
//	item2: WardrobeItem có Hue = 20 (cam - góc lệch 180 độ)
//
// Đầu ra mẫu:
//
//	true
func colorsMatch(item1, item2 *entities.WardrobeItem) bool {
	if isNeutral(item1) || isNeutral(item2) {
		return true
	}
	fashion1 := fashionForItem(item1)
	fashion2 := fashionForItem(item2)
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
) *dto.RecommendedOutfitRes {
	var tops, bottoms, shoes, outerwears, dresses, accessories []*entities.WardrobeItem

	for _, candidate := range candidates {
		item := candidate.Item
		category := item.FashionCategory()
		if category == nil {
			accessories = append(accessories, item)
			continue
		}

		switch category.Slug {
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
			if colorsMatch(primaryDress, shoe) {
				primaryShoes = shoe
				break
			}
		}

		items = append(items, buildItemGroup(roleForItem(primaryDress, "dam"), primaryDress, dresses))
		items = append(items, buildItemGroup("giay", primaryShoes, shoes))

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryDress, outerwear) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryOuter, "ao-khoac"), primaryOuter, outerwears))
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
		items = append(items, buildItemGroup(roleForItem(primaryBottom, "quan"), primaryBottom, bottoms))

		if len(shoes) > 0 {
			primaryShoes := shoes[0]
			for _, shoe := range shoes {
				if colorsMatch(primaryTop, shoe) || colorsMatch(primaryBottom, shoe) {
					primaryShoes = shoe
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryShoes, "giay"), primaryShoes, shoes))
		}

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryTop, outerwear) || colorsMatch(primaryBottom, outerwear) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryOuter, "ao-khoac"), primaryOuter, outerwears))
		}
	} else {
		if len(dresses) > 0 {
			items = append(items, buildItemGroup(roleForItem(dresses[0], "dam"), dresses[0], dresses))
		} else {
			if len(tops) > 0 {
				items = append(items, buildItemGroup(roleForItem(tops[0], "ao"), tops[0], tops))
			}
			if len(bottoms) > 0 {
				items = append(items, buildItemGroup(roleForItem(bottoms[0], "quan"), bottoms[0], bottoms))
			}
		}
		if len(shoes) > 0 {
			items = append(items, buildItemGroup(roleForItem(shoes[0], "giay"), shoes[0], shoes))
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

// buildItemGroup xây dựng cấu trúc [RecommendedItemGroup] cho một vai trò phối đồ cụ thể, bao gồm món đồ chính và tối đa 2 món đồ thay thế từ danh sách.
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

func roleForItem(item *entities.WardrobeItem, fallback string) string {
	if category := item.FashionCategory(); category != nil && category.Slug != "" {
		return category.Slug
	}
	return fallback
}

func fashionForItem(item *entities.WardrobeItem) *entities.FashionItem {
	if item == nil {
		return nil
	}
	return item.FashionItem
}

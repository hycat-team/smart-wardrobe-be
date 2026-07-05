// Package ranking implements rule-based candidate scoring, fallback candidate expansion, and list diversification.
package ranking

import (
	"math"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// isNeutral kiểm tra xem màu sắc của món đồ có phải là màu trung tính (ví dụ: đen, trắng, xám hoặc màu có độ bão hòa rất thấp) dựa trên độ sáng (lightness) và độ bão hòa (saturation).
func isNeutral(item *wardrobeDto.WardrobeItemRes, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
	fashion := fashionForItemFromMap(item, fashionItemMap)
	if fashion == nil || fashion.ColorLightness == nil || fashion.ColorSaturation == nil {
		return true
	}

	lightness := *fashion.ColorLightness
	saturation := *fashion.ColorSaturation
	return lightness <= 10.0 || lightness >= 90.0 || saturation <= 10.0
}

// colorsMatch kiểm tra xem hai món đồ có màu sắc phối hợp hài hòa với nhau theo thuật toán HSL hay không.
func colorsMatch(item1, item2 *wardrobeDto.WardrobeItemRes, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) bool {
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
func itemCategorySlug(item *wardrobeDto.WardrobeItemRes, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) string {
	if item.FashionItem != nil {
		if fi := fashionItemMap[item.FashionItem.ID]; fi != nil && fi.CategoryName != nil {
			name := strings.ToLower(strings.TrimSpace(*fi.CategoryName))
			if strings.Contains(name, "áo khoác") || strings.Contains(name, "ao khoac") || name == "ao-khoac" {
				return "ao-khoac"
			}
			if strings.Contains(name, "chân váy") || strings.Contains(name, "chan vay") || name == "chan-vay" {
				return "chan-vay"
			}
			if strings.Contains(name, "áo") || strings.Contains(name, "ao") {
				return "ao"
			}
			if strings.Contains(name, "quần") || strings.Contains(name, "quan") {
				return "quan"
			}
			if strings.Contains(name, "giày") || strings.Contains(name, "giay") {
				return "giay"
			}
			if strings.Contains(name, "đầm") || strings.Contains(name, "dam") {
				return "dam"
			}
			return name
		}
	}
	return ""
}

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

		items = append(items, buildItemGroup(roleForItem(primaryDress, "dam"), primaryDress, dresses, fashionItemMap))
		items = append(items, buildItemGroup("giay", primaryShoes, shoes, fashionItemMap))

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryDress, outerwear, fashionItemMap) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryOuter, "ao-khoac"), primaryOuter, outerwears, fashionItemMap))
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

		items = append(items, buildItemGroup("ao", primaryTop, tops, fashionItemMap))
		items = append(items, buildItemGroup(roleForItem(primaryBottom, "quan"), primaryBottom, bottoms, fashionItemMap))

		if len(shoes) > 0 {
			primaryShoes := shoes[0]
			for _, shoe := range shoes {
				if colorsMatch(primaryTop, shoe, fashionItemMap) || colorsMatch(primaryBottom, shoe, fashionItemMap) {
					primaryShoes = shoe
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryShoes, "giay"), primaryShoes, shoes, fashionItemMap))
		}

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, outerwear := range outerwears {
				if colorsMatch(primaryTop, outerwear, fashionItemMap) || colorsMatch(primaryBottom, outerwear, fashionItemMap) {
					primaryOuter = outerwear
					break
				}
			}
			items = append(items, buildItemGroup(roleForItem(primaryOuter, "ao-khoac"), primaryOuter, outerwears, fashionItemMap))
		}
	} else {
		if len(dresses) > 0 {
			items = append(items, buildItemGroup(roleForItem(dresses[0], "dam"), dresses[0], dresses, fashionItemMap))
		} else {
			if len(tops) > 0 {
				items = append(items, buildItemGroup(roleForItem(tops[0], "ao"), tops[0], tops, fashionItemMap))
			}
			if len(bottoms) > 0 {
				items = append(items, buildItemGroup(roleForItem(bottoms[0], "quan"), bottoms[0], bottoms, fashionItemMap))
			}
		}
		if len(shoes) > 0 {
			items = append(items, buildItemGroup(roleForItem(shoes[0], "giay"), shoes[0], shoes, fashionItemMap))
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
	primary *wardrobeDto.WardrobeItemRes,
	all []*wardrobeDto.WardrobeItemRes,
	fashionItemMap map[uuid.UUID]*dto.FashionItemRes,
) *dto.RecommendedItemGroup {
	alternatives := make([]*wardrobeDto.WardrobeItemRes, 0, 2)
	for _, item := range all {
		if item.ID == primary.ID {
			continue
		}
		if len(alternatives) >= 2 {
			break
		}
		var fi *dto.FashionItemRes
		if item.FashionItem != nil {
			fi = fashionItemMap[item.FashionItem.ID]
		}
		alternatives = append(alternatives, mapToWardrobeItemResWithFashion(item, fi))
	}

	var primaryFI *dto.FashionItemRes
	if primary.FashionItem != nil {
		primaryFI = fashionItemMap[primary.FashionItem.ID]
	}
	return &dto.RecommendedItemGroup{
		Role:         role,
		Primary:      mapToWardrobeItemResWithFashion(primary, primaryFI),
		Alternatives: alternatives,
	}
}

func roleForItem(item *wardrobeDto.WardrobeItemRes, fallback string) string {
	if item.FashionItem != nil {
		// caller should use fashionItemMap; fallback is used here
		_ = item.FashionItem.ID
	}
	return fallback
}

func fashionForItemFromMap(item *wardrobeDto.WardrobeItemRes, fashionItemMap map[uuid.UUID]*dto.FashionItemRes) *dto.FashionItemRes {
	if item == nil || item.FashionItem == nil {
		return nil
	}
	return fashionItemMap[item.FashionItem.ID]
}

func mapToWardrobeItemResWithFashion(item *wardrobeDto.WardrobeItemRes, fi *dto.FashionItemRes) *wardrobeDto.WardrobeItemRes {
	if item == nil {
		return nil
	}
	res := *item
	if fi != nil {
		res.FashionItem = &wardrobeDto.FashionItemRes{
			ID:              fi.ID,
			CategoryID:      fi.CategoryID,
			CategoryName:    fi.CategoryName,
			ImageUrl:        fi.ImageUrl,
			ImagePublicID:   fi.ImagePublicID,
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
		if fi.CategoryName != nil {
			res.Category = &wardrobeDto.CategoryRes{
				ID:   *fi.CategoryID,
				Name: *fi.CategoryName,
			}
		}
	}
	return &res
}

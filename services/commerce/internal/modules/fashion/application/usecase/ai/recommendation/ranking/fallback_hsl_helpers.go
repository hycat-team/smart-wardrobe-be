package ranking

import (
	"math"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

// ============================================================================
// HSL Color Compatibility Helpers
// ============================================================================

// isNeutral kiểm tra xem màu sắc của món đồ có phải là màu trung tính (ví dụ: đen, trắng, xám hoặc màu có độ bão hòa rất thấp) dựa trên độ sáng (lightness) và độ bão hòa (saturation).
func isNeutral(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	fashion := fashionForItemFromMap(item, fashionItemMap)
	if fashion == nil || fashion.ColorLightness == nil || fashion.ColorSaturation == nil {
		return true
	}

	lightness := *fashion.ColorLightness
	saturation := *fashion.ColorSaturation
	return lightness <= 10.0 || lightness >= 90.0 || saturation <= 10.0
}

// colorsMatch kiểm tra xem hai món đồ có màu sắc phối hợp hài hòa với nhau theo thuật toán HSL hay không.
func colorsMatch(item1, item2 *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
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

func fashionForItemFromMap(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) *dto.FashionItemContractDTO {
	return mapper.FashionForItemFromMap(item, fashionItemMap)
}

// ============================================================================
// Category & Role Normalization Helpers
// ============================================================================

func itemCategorySlug(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
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

func roleForItem(item *wardrobeDto.WardrobeItemContractDTO, fallback string, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
	if slug := itemCategorySlug(item, fashionItemMap); slug != "" {
		return mapper.CategorySlugToWearingRole(slug)
	}
	return mapper.CategorySlugToWearingRole(fallback)
}

// ============================================================================
// Outerwear Layering Decision Helpers
// ============================================================================

// outerwearHotLikeTerms là các từ khóa cho thấy thời tiết nóng, không nên thêm lớp ngoài.
var outerwearHotLikeTerms = []string{"nóng", "nong", "nắng", "nang", "hot", "sunny", "summer", "he"}

// outerwearColdLikeTerms là các từ khóa cho thấy thời tiết lạnh hoặc mát, nên thêm lớp ngoài.
var outerwearColdLikeTerms = []string{"lạnh", "lanh", "mát", "mat", "cold", "cool", "mưa", "mua", "rain"}

// outerwearLayeringIntentTerms là các từ khóa thể hiện người dùng muốn phối đồ theo lớp.
var outerwearLayeringIntentTerms = []string{"layer", "khoác", "khoac"}

// shouldIncludeOuterwear quyết định outfit có nên thêm lớp ngoài (outerwear) hay không.
//
// Quy tắc:
//  1. Thời tiết nóng hoặc mùa hè → false, tránh outerwear dày/nặng.
//  2. Thời tiết lạnh/mát hoặc mùa đông/thu → true.
//  3. Không có tín hiệu thời tiết → true chỉ khi ngữ cảnh yêu cầu có dấu hiệu layering.
func shouldIncludeOuterwear(input dto.RecommendOutfitReq) bool {
	if matchesAnyWeatherTerm(input, outerwearHotLikeTerms) || isSummerSeason(input) {
		return false
	}
	if matchesAnyWeatherTerm(input, outerwearColdLikeTerms) || isColdSeason(input) {
		return true
	}
	return hasLayeringIntent(input)
}

// isColdSeason kiểm tra mùa đông hoặc thu, dùng như tín hiệu thời tiết lạnh.
func isColdSeason(input dto.RecommendOutfitReq) bool {
	season := seasonValue(input)
	return season == "winter" || season == "dong" || season == "autumn" || season == "fall" || season == "thu"
}

// isSummerSeason kiểm tra mùa hè, dùng như tín hiệu thời tiết nóng.
func isSummerSeason(input dto.RecommendOutfitReq) bool {
	season := seasonValue(input)
	return season == "summer" || season == "he"
}

// seasonValue trả về giá trị mùa ở dạng chữ thường, chuỗi rỗng nếu không có.
func seasonValue(input dto.RecommendOutfitReq) string {
	if input.Season == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(*input.Season)))
}

// matchesAnyWeatherTerm kiểm tra chuỗi thời tiết do người dùng nhập có chứa bất kỳ từ khóa nào không.
func matchesAnyWeatherTerm(input dto.RecommendOutfitReq, terms []string) bool {
	if input.Weather == nil {
		return false
	}
	weather := strings.ToLower(strings.TrimSpace(*input.Weather))
	if weather == "" {
		return false
	}
	for _, term := range terms {
		if strings.Contains(weather, term) {
			return true
		}
	}
	return false
}

// hasLayeringIntent kiểm tra ngữ cảnh yêu cầu (phong cách, dịp, ghi chú) có nhắc tới
// phối đồ nhiều lớp hay không.
func hasLayeringIntent(input dto.RecommendOutfitReq) bool {
	contexts := make([]string, 0, 3)
	if input.StyleTarget != nil {
		contexts = append(contexts, *input.StyleTarget)
	}
	if input.Occasion != nil {
		contexts = append(contexts, *input.Occasion)
	}
	if input.Details != nil {
		contexts = append(contexts, *input.Details)
	}
	for _, value := range contexts {
		normalized := strings.ToLower(value)
		for _, term := range outerwearLayeringIntentTerms {
			if strings.Contains(normalized, term) {
				return true
			}
		}
	}
	return false
}

// ============================================================================
// DTO Mapping & Group Builder Helpers
// ============================================================================

// buildItemGroup xây dựng cấu trúc [RecommendedItemGroup] cho một vai trò phối đồ cụ thể, bao gồm món đồ chính và tối đa 2 món đồ thay thế từ danh sách.
func buildItemGroup(
	role string,
	primary *wardrobeDto.WardrobeItemContractDTO,
	all []*wardrobeDto.WardrobeItemContractDTO,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
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

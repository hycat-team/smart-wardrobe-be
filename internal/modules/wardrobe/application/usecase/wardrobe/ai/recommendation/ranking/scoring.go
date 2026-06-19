// Package ranking implements rule-based candidate scoring, fallback candidate expansion, and list diversification.
package ranking

import (
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/retrieval"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

const (
	recommendationWeatherUnspecified = "unspecified"
	recommendationWeatherColdLike    = "cold-like"
	recommendationWeatherHotLike     = "hot-like"
	recommendationWeatherRainy       = "rainy"

	candidateSourceFallback        = "fallback"
	candidateSourceStrictFallback  = "strict-fallback"
	candidateSourceRelaxedFallback = "relaxed-fallback"
	candidateSourceGeneralFallback = "general-fallback"
)

// ScoreCandidateItem chấm điểm một món đồ ứng viên dựa trên mức độ phù hợp của nó với phong cách, dịp, tông màu và thời tiết yêu cầu.
//
// Hành vi:
// 1. **Phong cách (Style)**: Cộng 0.3 điểm nếu phong cách của món đồ khớp với phong cách yêu cầu.
// 2. **Dịp (Occasion)**: Cộng 0.2 điểm nếu khớp trực tiếp, hoặc cộng 0.08 điểm nếu khớp qua từ điển mở rộng (taxonomy).
// 3. **Tông màu (Color Tone)**: Cộng 0.2 điểm nếu màu sắc món đồ (dựa trên độ sáng Lightness, góc màu Hue, độ bão hòa Saturation) phù hợp tông màu mong muốn.
// 4. **Tránh Phong cách/Tông màu**: Trừ điểm (tương ứng 0.35 và 0.25) nếu món đồ thuộc phong cách hoặc màu sắc mà người dùng muốn tránh.
// 5. **Màu trung tính**: Cộng 0.1 điểm vì dễ phối đồ.
// 6. **Thời tiết**:
//   - Nếu trời lạnh, cộng 0.4 điểm cho áo khoác. Nếu trời nóng, trừ 0.3 điểm cho áo khoác dày.
//   - Nếu trời mưa, cộng 0.15 điểm cho các đồ thích hợp đi mưa.
//   - Trừ điểm phạt nếu món đồ rơi vào kiểu thời tiết người dùng muốn tránh (ví dụ: tránh đồ lạnh khi trời nóng).
//
// 7. **So khớp Lexical**: Cộng thêm điểm cho các từ khóa tìm kiếm thô khớp với metadata của món đồ (tối đa cộng 0.5 điểm).
// 8. **Tần suất sử dụng**:
//   - Trừ điểm phạt (tối đa 0.4) nếu món đồ vừa mới mặc gần đây (trong khoảng [recentlyWornDays]).
//   - Cộng điểm thưởng 0.15 nếu món đồ đã lâu không mặc hoặc là đồ mới chưa từng mặc.
//
// Đầu vào mẫu:
//
//	item: entities.WardrobeItem{Style: pointer to "Minimalist", LastUsedAt: nil}
//	intent: dto.ParsedIntent{StyleTarget: []string{"minimalist"}, Occasion: []string{"work"}}
//
// Đầu ra mẫu:
//
//	(1.65, []string{"style-match:minimalist", "new-item-bonus", ...})
func ScoreCandidateItem(
	item *entities.WardrobeItem,
	intent dto.ParsedIntent,
	retrievalQuery types.RecommendationRetrievalQuery,
	recentlyWornDays,
	longUnwornDays int,
) (float64, []string) {
	score := 1.0
	var reasonTags []string
	styleMatched := false
	matchedTerms := map[string]bool{}

	if item.Style != nil && len(intent.StyleTarget) > 0 {
		itemStyle := strings.ToLower(*item.Style)
		for _, target := range intent.StyleTarget {
			targetNorm := strings.ToLower(target)
			if strings.Contains(itemStyle, targetNorm) {
				score += 0.3
				styleMatched = true
				reasonTags = append(reasonTags, "style-match:"+target)
				matchedTerms[targetNorm] = true
				break
			}
		}
	}

	if len(intent.Occasion) > 0 {
		for _, occ := range intent.Occasion {
			occasion := parser.NormalizeText(occ)
			if CandidateMatchesAnyTerm(item, []string{occasion}) {
				score += 0.2
				reasonTags = append(reasonTags, "occasion-match:"+occasion)
				matchedTerms[occasion] = true
				break
			} else {
				expanded := retrieval.ExpandTaxonomyTermValues("occasion", []string{occasion})
				matchedExp := ""
				for _, exp := range expanded {
					expNorm := parser.NormalizeText(exp)
					if CandidateMatchesAnyTerm(item, []string{expNorm}) {
						matchedExp = expNorm
						break
					}
				}
				if matchedExp != "" {
					score += 0.08
					reasonTags = append(reasonTags, "occasion-match:taxonomy:"+occ)
					matchedTerms[matchedExp] = true
					break
				}
			}
		}
	}

	if len(intent.ColorTone) > 0 && item.ColorLightness != nil {
		for _, t := range intent.ColorTone {
			tone := strings.ToLower(t)
			lightness := *item.ColorLightness
			if tone == "light" && lightness >= 60 {
				score += 0.2
				reasonTags = append(reasonTags, "color-tone:light")
				break
			} else if tone == "dark" && lightness <= 40 {
				score += 0.2
				reasonTags = append(reasonTags, "color-tone:dark")
				break
			} else if tone == "earthy" && item.ColorHue != nil && item.ColorSaturation != nil {
				hue := *item.ColorHue
				saturation := *item.ColorSaturation
				if ((hue >= 20 && hue <= 50) || (hue >= 80 && hue <= 120)) &&
					saturation <= 50 && lightness >= 20 && lightness <= 70 {
					score += 0.2
					reasonTags = append(reasonTags, "color-tone:earthy")
					break
				}
			}
		}
	}

	if item.Style != nil && len(intent.ExcludedStyles) > 0 {
		itemStyle := strings.ToLower(*item.Style)
		for _, excluded := range intent.ExcludedStyles {
			excluded = strings.ToLower(excluded)
			if strings.Contains(itemStyle, excluded) {
				score -= 0.35
				reasonTags = append(reasonTags, "avoid-style:"+excluded)
				break
			}
		}
	}

	if len(intent.ExcludedColorTones) > 0 && item.ColorLightness != nil {
		for _, excluded := range intent.ExcludedColorTones {
			tone := strings.ToLower(excluded)
			lightness := *item.ColorLightness
			if tone == "light" && lightness >= 60 {
				score -= 0.25
				reasonTags = append(reasonTags, "avoid-color-tone:light")
				break
			} else if tone == "dark" && lightness <= 40 {
				score -= 0.25
				reasonTags = append(reasonTags, "avoid-color-tone:dark")
				break
			} else if tone == "earthy" && item.ColorHue != nil && item.ColorSaturation != nil {
				hue := *item.ColorHue
				saturation := *item.ColorSaturation
				if ((hue >= 20 && hue <= 50) || (hue >= 80 && hue <= 120)) &&
					saturation <= 50 && lightness >= 20 && lightness <= 70 {
					score -= 0.25
					reasonTags = append(reasonTags, "avoid-color-tone:earthy")
					break
				}
			}
		}
	}

	if isNeutral(item) {
		score += 0.1
		reasonTags = append(reasonTags, "neutral-versatility")
	}

	weatherState := recommendationWeatherState(intent.PositiveConstraints)
	if isOuterwearCategory(item.Category) {
		switch weatherState {
		case recommendationWeatherColdLike:
			score += 0.4
			reasonTags = append(reasonTags, "weather-appropriate:outerwear")
		case recommendationWeatherHotLike:
			score -= 0.3
			reasonTags = append(reasonTags, "weather-mismatch:heavy-outerwear")
		}
	}
	if weatherState == recommendationWeatherRainy && CandidateMatchesRainySignal(item) {
		score += 0.15
		reasonTags = append(reasonTags, "weather-appropriate:rainy")
	}

	if penalty, tag := excludedWeatherPenalty(item, intent.ExcludedWeather); penalty > 0 {
		score -= penalty
		reasonTags = append(reasonTags, tag)
	}

	lexicalBoost := 0.0
	for _, term := range retrievalQuery.LexicalTerms {
		termVal := strings.ToLower(term.Value)
		if termVal == "" || matchedTerms[termVal] {
			continue
		}
		if CandidateMatchesAnyTerm(item, []string{termVal}) {
			boost := 0.0
			switch term.Source {
			case types.RetrievalTermSourceDictionary:
				boost = 0.15
			case types.RetrievalTermSourceRaw:
				boost = 0.15
			case types.RetrievalTermSourceTaxonomy:
				boost = 0.05
			}
			if boost > 0 {
				lexicalBoost += boost
				reasonTags = append(reasonTags, fmt.Sprintf("lexical-match:%s:%s", term.Source, term.Value))
				matchedTerms[termVal] = true
			}
		}
	}
	if lexicalBoost > 0.5 {
		lexicalBoost = 0.5
	}
	score += lexicalBoost

	if item.LastUsedAt != nil {
		days := int(time.Since(*item.LastUsedAt).Hours() / 24)
		if days < recentlyWornDays {
			penalty := float64(recentlyWornDays-days) / float64(recentlyWornDays) * 0.4
			score -= penalty
			reasonTags = append(reasonTags, fmt.Sprintf("recently-worn-penalty:-%.2f", penalty))
		} else if days > longUnwornDays && (styleMatched || len(intent.Occasion) > 0) {
			score += 0.15
			reasonTags = append(reasonTags, "long-unworn-bonus")
		}
	} else if styleMatched || len(intent.Occasion) > 0 {
		score += 0.15
		reasonTags = append(reasonTags, "new-item-bonus")
	}

	return score, reasonTags
}

// recommendationWeatherState phân loại các từ khóa mô tả thời tiết thành các trạng thái chuẩn hóa (cold-like, hot-like, rainy, hoặc unspecified).
func recommendationWeatherState(values []string) string {
	for _, value := range values {
		switch parser.NormalizeText(value) {
		case "cold", "cool", "winter", "autumn":
			return recommendationWeatherColdLike
		case "hot", "summer":
			return recommendationWeatherHotLike
		case "rainy", "rain":
			return recommendationWeatherRainy
		}
	}
	return recommendationWeatherUnspecified
}

// isOuterwearCategory kiểm tra xem danh mục của món đồ có thuộc nhóm áo khoác/đồ mặc ngoài hay không.
func isOuterwearCategory(category *entities.Category) bool {
	if category == nil {
		return false
	}
	text := parser.NormalizeText(category.Slug + " " + category.Name)
	for _, term := range retrieval.OuterwearCategoryTerms() {
		if strings.Contains(text, parser.NormalizeText(term)) {
			return true
		}
	}
	return false
}

// CandidateMatchesAnyTerm kiểm tra xem tài liệu tìm kiếm của món đồ có chứa bất kỳ từ khóa nào trong danh sách hay không.
func CandidateMatchesAnyTerm(item *entities.WardrobeItem, terms []string) bool {
	if item == nil {
		return false
	}
	document := FallbackSearchDocument(item)
	for _, term := range terms {
		term = parser.NormalizeText(term)
		if term != "" && strings.Contains(document, term) {
			return true
		}
	}
	return false
}

// CandidateMatchesRainySignal kiểm tra xem món đồ có chứa các thuộc tính phù hợp để đi mưa hay không.
func CandidateMatchesRainySignal(item *entities.WardrobeItem) bool {
	return CandidateMatchesAnyTerm(item, retrieval.RainyWeatherTerms())
}

// excludedWeatherPenalty tính điểm phạt (penalty) và trả về nhãn lý do nếu món đồ không phù hợp với kiểu thời tiết người dùng muốn tránh.
func excludedWeatherPenalty(item *entities.WardrobeItem, excludedWeather []string) (float64, string) {
	if item == nil {
		return 0, ""
	}
	for _, excluded := range excludedWeather {
		state := recommendationWeatherState([]string{excluded})
		switch state {
		case recommendationWeatherColdLike:
			if isOuterwearCategory(item.Category) || CandidateMatchesAnyTerm(item, retrieval.ColdLikeWeatherTerms()) {
				return 0.3, "avoid-weather:cold-like"
			}
		case recommendationWeatherHotLike:
			if CandidateMatchesAnyTerm(item, retrieval.HotLikeWeatherTerms()) {
				return 0.2, "avoid-weather:hot-like"
			}
		case recommendationWeatherRainy:
			if CandidateMatchesRainySignal(item) {
				return 0.25, "avoid-weather:rainy"
			}
		}
	}
	return 0, ""
}

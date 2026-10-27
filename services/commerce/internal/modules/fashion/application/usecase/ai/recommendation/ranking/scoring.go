// Package ranking implements rule-based candidate scoring, fallback candidate expansion, and list diversification.
package ranking

import (
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/parser"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/retrieval"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	personalization_shared "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

const (
	recommendationWeatherUnspecified = "unspecified"
	recommendationWeatherColdLike    = "cold-like"
	recommendationWeatherHotLike     = "hot-like"
	recommendationWeatherRainy       = "rainy"
)

// Personal profile score deltas (033 plan D4, T031). They are soft signals: a
// matched item gets a bonus or a penalty, never an inclusion/exclusion decision.
const (
	// personalStyleMatchScore applies when the request carries no style target,
	// so the saved profile is the only style signal.
	personalStyleMatchScore = 0.25
	// personalStyleMatchWithRequestScore applies when the request already chose
	// a style: the request wins and the profile only nudges the ranking.
	personalStyleMatchWithRequestScore = 0.10
	// learnedStyleMatchScore is the default weight of a learned-style match. The
	// effective weight comes from personalization.learned_style_weight (carried
	// on the intent); this constant only applies when it is not configured.
	learnedStyleMatchScore = 0.10
	// avoidStylePenalty applies to items matching a declared excluded style.
	avoidStylePenalty = 0.30
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
// 8. **Tần suất sử dụng (033 US3/T051)**:
//   - Trừ điểm phạt theo cửa sổ `RecencyWindow.RecentlyWornDays` (mặc định 7 ngày), tối đa
//     0.45 khi vừa mặc hôm nay và giảm tuyến tính về 0 tại mép cửa sổ.
//   - Cộng 0.15 khi món chưa từng dùng hoặc lâu hơn `LongUnwornDays`, bất kể yêu cầu có nêu
//     phong cách hay dịp hay không (FR-056). Món được tạo trong cửa sổ đó được gắn nhãn
//     `new-item-bonus`, còn lại là `long-unworn-bonus` (FR-057: "món mới" theo `created_at`).
//   - Trừ 0.12 khi món vừa được gợi ý trong `RecentlyRecommendedDays` (mặc định 3 ngày) và
//     chưa bị phạt vì vừa mặc, để không lặp lại cùng một món qua nhiều lượt gợi ý.
//   - Món trong vùng làm mát (vừa mặc hoặc vừa được gợi ý) mang nhãn `recency-cooldown`.
//
// 9. **Gu cá nhân (mềm, 033 US1)**:
//   - Cộng 0.25 (hoặc 0.10 nếu request đã có style target) khi món đồ khớp gu khai báo.
//   - Cộng theo trọng số `personalization.learned_style_weight` (mặc định 0.10) khi món đồ khớp gu
//     học được từ hành vi lưu outfit.
//   - Trừ 0.30 khi món đồ khớp gu người dùng muốn tránh.
//   - Không bao giờ loại món đồ khỏi pool vì gu.
//
// Đầu vào mẫu:
//
//	item: wardrobeDto.WardrobeItemRes{Style: pointer to "Minimalist", LastUsedAt: nil}
//	intent: dto.ParsedIntent{StyleTarget: []string{"minimalist"}, Occasion: []string{"work"}}
//
// Đầu ra mẫu:
//
//	(1.65, []string{"style-match:minimalist", "new-item-bonus", ...})
func ScoreCandidateItem(
	item *wardrobeDto.WardrobeItemContractDTO,
	intent dto.ParsedIntent,
	retrievalQuery types.RecommendationRetrievalQuery,
	recency RecencyWindow,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) (float64, []string) {
	score := 1.0
	var reasonTags []string
	matchedTerms := map[string]bool{}
	fashion := fashionForItemFromMap(item, fashionItemMap)

	if fashion != nil && fashion.Style != nil && len(intent.StyleTarget) > 0 {
		itemStyle := strings.ToLower(*fashion.Style)
		for _, target := range intent.StyleTarget {
			targetNorm := strings.ToLower(target)
			if strings.Contains(itemStyle, targetNorm) {
				score += 0.3
				reasonTags = append(reasonTags, "style-match:"+target)
				matchedTerms[targetNorm] = true
				break
			}
		}
	}

	if len(intent.Occasion) > 0 {
		for _, occ := range intent.Occasion {
			occasion := parser.NormalizeText(occ)
			if CandidateMatchesAnyTerm(item, []string{occasion}, fashionItemMap) {
				score += 0.2
				reasonTags = append(reasonTags, "occasion-match:"+occasion)
				matchedTerms[occasion] = true
				break
			} else {
				expanded := retrieval.ExpandTaxonomyTermValues("occasion", []string{occasion})
				matchedExp := ""
				for _, exp := range expanded {
					expNorm := parser.NormalizeText(exp)
					if CandidateMatchesAnyTerm(item, []string{expNorm}, fashionItemMap) {
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

	if fashion != nil && len(intent.ColorTone) > 0 && fashion.ColorLightness != nil {
		for _, t := range intent.ColorTone {
			tone := strings.ToLower(t)
			lightness := *fashion.ColorLightness
			if tone == "light" && lightness >= 60 {
				score += 0.2
				reasonTags = append(reasonTags, "color-tone:light")
				break
			} else if tone == "dark" && lightness <= 40 {
				score += 0.2
				reasonTags = append(reasonTags, "color-tone:dark")
				break
			} else if tone == "earthy" && fashion.ColorHue != nil && fashion.ColorSaturation != nil {
				hue := *fashion.ColorHue
				saturation := *fashion.ColorSaturation
				if ((hue >= 20 && hue <= 50) || (hue >= 80 && hue <= 120)) &&
					saturation <= 50 && lightness >= 20 && lightness <= 70 {
					score += 0.2
					reasonTags = append(reasonTags, "color-tone:earthy")
					break
				}
			}
		}
	}

	if fashion != nil && fashion.Style != nil && len(intent.ExcludedStyles) > 0 {
		itemStyle := strings.ToLower(*fashion.Style)
		for _, excluded := range intent.ExcludedStyles {
			excluded = strings.ToLower(excluded)
			if strings.Contains(itemStyle, excluded) {
				score -= 0.35
				reasonTags = append(reasonTags, "avoid-style:"+excluded)
				break
			}
		}
	}

	// Soft personal profile signals (033 US1/T031). They never filter a
	// candidate out; they only move it up or down the ranked pool.
	if itemMatchesAnyStyleLabel(fashion, intent.AvoidStyle) {
		score -= avoidStylePenalty
		reasonTags = append(reasonTags, "avoid-style")
	}
	if itemMatchesAnyStyleLabel(fashion, intent.PersonalStyle) {
		if len(intent.StyleTarget) > 0 {
			score += personalStyleMatchWithRequestScore
		} else {
			score += personalStyleMatchScore
		}
		reasonTags = append(reasonTags, "personal-style-match")
	}
	if itemMatchesAnyStyleLabel(fashion, intent.LearnedStyle) {
		score += learnedStyleWeight(intent)
		reasonTags = append(reasonTags, "learned-style-match")
	}

	if fashion != nil && len(intent.ExcludedColorTones) > 0 && fashion.ColorLightness != nil {
		for _, excluded := range intent.ExcludedColorTones {
			tone := strings.ToLower(excluded)
			lightness := *fashion.ColorLightness
			if tone == "light" && lightness >= 60 {
				score -= 0.25
				reasonTags = append(reasonTags, "avoid-color-tone:light")
				break
			} else if tone == "dark" && lightness <= 40 {
				score -= 0.25
				reasonTags = append(reasonTags, "avoid-color-tone:dark")
				break
			} else if tone == "earthy" && fashion.ColorHue != nil && fashion.ColorSaturation != nil {
				hue := *fashion.ColorHue
				saturation := *fashion.ColorSaturation
				if ((hue >= 20 && hue <= 50) || (hue >= 80 && hue <= 120)) &&
					saturation <= 50 && lightness >= 20 && lightness <= 70 {
					score -= 0.25
					reasonTags = append(reasonTags, "avoid-color-tone:earthy")
					break
				}
			}
		}
	}

	if isNeutral(item, fashionItemMap) {
		score += 0.1
		reasonTags = append(reasonTags, "neutral-versatility")
	}

	weatherState := recommendationWeatherState(intent.PositiveConstraints)
	if isOuterwearItem(item, fashionItemMap) {
		switch weatherState {
		case recommendationWeatherColdLike:
			score += 0.4
			reasonTags = append(reasonTags, "weather-appropriate:outerwear")
		case recommendationWeatherHotLike:
			score -= 0.3
			reasonTags = append(reasonTags, "weather-mismatch:heavy-outerwear")
		}
	}
	if weatherState == recommendationWeatherRainy && CandidateMatchesRainySignal(item, fashionItemMap) {
		score += 0.15
		reasonTags = append(reasonTags, "weather-appropriate:rainy")
	}

	// Mũ và phụ kiện là lớp bổ sung luôn hợp lệ với mọi outfit và không phụ thuộc
	// thời tiết. Cộng nhẹ để chúng không bị đẩy xuống cuối pool bởi những tín hiệu
	// không liên quan, nhờ vậy vẫn nằm trong cửa sổ prompt thay vì bị cắt từ đuôi.
	if isAccessoryLayerItem(item, fashionItemMap) {
		score += 0.05
		reasonTags = append(reasonTags, "layer-role-bonus")
	}

	if penalty, tag := excludedWeatherPenalty(item, intent.ExcludedWeather, fashionItemMap); penalty > 0 {
		score -= penalty
		reasonTags = append(reasonTags, tag)
	}

	lexicalBoost := 0.0
	for _, term := range retrievalQuery.LexicalTerms {
		termVal := strings.ToLower(term.Value)
		if termVal == "" || matchedTerms[termVal] {
			continue
		}
		if CandidateMatchesAnyTerm(item, []string{termVal}, fashionItemMap) {
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

	// Rotation signals (033 US3/T051). They are additive and soft: the item is
	// never removed from the pool, only moved up or down.
	now := time.Now()
	window := recency.Normalized()
	daysSinceUsed, hasBeenUsed := 0, item.LastUsedAt != nil
	if hasBeenUsed {
		daysSinceUsed = DaysSince(*item.LastUsedAt, now)
	}
	recentlyWorn := hasBeenUsed && daysSinceUsed < window.RecentlyWornDays
	if recentlyWorn {
		penalty := (1 - float64(daysSinceUsed)/float64(window.RecentlyWornDays)) * recentlyWornPenaltyMax
		score -= penalty
		reasonTags = append(reasonTags, fmt.Sprintf("recently-worn-penalty:-%.2f", penalty))
	}

	// FR-056: the long-unworn bonus no longer depends on style/occasion, and a
	// never-used item counts as long-unworn. FR-057: the "new item" label comes
	// from created_at, so a brand-new item is tagged as new while an old item
	// that was never worn keeps the plain long-unworn tag. Both get one 0.15
	// bonus, never two.
	if !hasBeenUsed || daysSinceUsed > window.LongUnwornDays {
		score += longUnwornBonus
		if isNewItem(item, now, window.LongUnwornDays) {
			reasonTags = append(reasonTags, "new-item-bonus")
		} else {
			reasonTags = append(reasonTags, "long-unworn-bonus")
		}
	}

	recentlyRecommended := false
	if item.LastRecommendedAt != nil {
		daysSinceRecommended := DaysSince(*item.LastRecommendedAt, now)
		recentlyRecommended = daysSinceRecommended < window.RecentlyRecommendedDays
		// A recently worn item already carries the larger worn penalty; stacking
		// the recommendation penalty would double-punish the same signal.
		if recentlyRecommended && !recentlyWorn {
			score -= recentlyRecommendedPenalty
			reasonTags = append(reasonTags, fmt.Sprintf("recently-recommended-penalty:-%.2f", recentlyRecommendedPenalty))
		}
	}
	if recentlyWorn || recentlyRecommended {
		reasonTags = append(reasonTags, RecencyCooldownTag)
	}

	return score, reasonTags
}

// learnedStyleWeight returns the effective score bonus of a learned-style
// match: personalization.learned_style_weight when the personalization helper
// supplied it, otherwise the built-in default.
func learnedStyleWeight(intent dto.ParsedIntent) float64 {
	if intent.LearnedStyleWeight > 0 {
		return intent.LearnedStyleWeight
	}
	return learnedStyleMatchScore
}

// itemMatchesAnyStyleLabel reports whether the item's style metadata matches any
// of the given style labels. It normalises both sides with the existing parser
// and expands each label through the style taxonomy, so a Vietnamese item style
// ("Tối giản") matches the canonical label ("minimalist") and vice versa.
//
// The profile labels were already normalised when the context was built; this
// helper repeats the normalisation so it is also correct when called directly.
func itemMatchesAnyStyleLabel(fashion *dto.FashionItemContractDTO, labels []string) bool {
	if fashion == nil || fashion.Style == nil || len(labels) == 0 {
		return false
	}
	itemStyle := parser.NormalizeText(*fashion.Style)
	if itemStyle == "" {
		return false
	}
	for _, label := range personalization_shared.NormalizeStyleLabels(labels) {
		if styleLabelMatchesItemStyle(label, itemStyle) {
			return true
		}
	}
	return false
}

// styleLabelMatchesItemStyle checks one canonical style label (and its taxonomy
// synonyms) against the already-normalised item style text.
func styleLabelMatchesItemStyle(label, itemStyle string) bool {
	terms := append([]string{label}, retrieval.ExpandTaxonomyTermValues("style", []string{label})...)
	for _, term := range terms {
		normalized := parser.NormalizeText(term)
		if normalized != "" && strings.Contains(itemStyle, normalized) {
			return true
		}
	}
	return false
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

// isOuterwearItem kiểm tra xem danh mục của món đồ có thuộc nhóm áo khoác/đồ mặc ngoài hay không.
func isOuterwearItem(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	if item == nil {
		return false
	}
	fi := fashionForItemFromMap(item, fashionItemMap)
	if fi == nil || fi.Category == nil {
		return false
	}
	text := parser.NormalizeText(fi.Category.Slug)
	for _, term := range retrieval.OuterwearCategoryTerms() {
		if strings.Contains(text, parser.NormalizeText(term)) {
			return true
		}
	}
	return false
}

// isAccessoryLayerItem kiểm tra món đồ có thuộc nhóm lớp phụ (mũ hoặc phụ kiện) hay không.
func isAccessoryLayerItem(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	fi := fashionForItemFromMap(item, fashionItemMap)
	if fi == nil || fi.Category == nil {
		return false
	}
	switch mapper.CategorySlugToWearingRole(fi.Category.Slug) {
	case "headwear", "accessory":
		return true
	default:
		return false
	}
}

// CandidateMatchesAnyTerm kiểm tra xem tài liệu tìm kiếm của món đồ có chứa bất kỳ từ khóa nào trong danh sách hay không.
func CandidateMatchesAnyTerm(item *wardrobeDto.WardrobeItemContractDTO, terms []string, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	if item == nil {
		return false
	}
	document := FallbackSearchDocument(item, fashionItemMap)
	for _, term := range terms {
		term = parser.NormalizeText(term)
		if term != "" && strings.Contains(document, term) {
			return true
		}
	}
	return false
}

// CandidateMatchesRainySignal kiểm tra xem món đồ có chứa các thuộc tính phù hợp để đi mưa hay không.
func CandidateMatchesRainySignal(item *wardrobeDto.WardrobeItemContractDTO, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) bool {
	return CandidateMatchesAnyTerm(item, retrieval.RainyWeatherTerms(), fashionItemMap)
}

// excludedWeatherPenalty tính điểm phạt (penalty) và trả về nhãn lý do nếu món đồ không phù hợp với kiểu thời tiết người dùng muốn tránh.
func excludedWeatherPenalty(item *wardrobeDto.WardrobeItemContractDTO, excludedWeather []string, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) (float64, string) {
	if item == nil {
		return 0, ""
	}
	for _, excluded := range excludedWeather {
		state := recommendationWeatherState([]string{excluded})
		switch state {
		case recommendationWeatherColdLike:
			if isOuterwearItem(item, fashionItemMap) || CandidateMatchesAnyTerm(item, retrieval.ColdLikeWeatherTerms(), fashionItemMap) {
				return 0.3, "avoid-weather:cold-like"
			}
		case recommendationWeatherHotLike:
			if CandidateMatchesAnyTerm(item, retrieval.HotLikeWeatherTerms(), fashionItemMap) {
				return 0.2, "avoid-weather:hot-like"
			}
		case recommendationWeatherRainy:
			if CandidateMatchesRainySignal(item, fashionItemMap) {
				return 0.25, "avoid-weather:rainy"
			}
		}
	}
	return 0, ""
}

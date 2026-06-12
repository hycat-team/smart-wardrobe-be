package ai

import (
	"fmt"
	"math"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func buildRecommendationPrompt(candidates []*entities.WardrobeItem, input dto.RecommendOutfitReq) string {
	var builder strings.Builder
	builder.WriteString(`You are a professional AI fashion stylist. Your task is to recommend exactly 1 styled outfit from the user's available wardrobe candidates based on their weather and occasion requirements.

OUTPUT REQUIREMENT:
Return ONLY a valid JSON string. Do NOT wrap the response in markdown formatting (like json code blocks). The JSON object MUST follow this exact structure:
{
  "title": "A creative name for the recommended outfit in Vietnamese (e.g. 'Casual Đi Chơi Cuối Tuần')",
  "explanation": "A concise explanation in Vietnamese explaining why this combination looks good and is suitable for the context",
  "items": [
    {
      "role": "The role/category of the item in lowercase without accents (e.g. 'ao', 'quan', 'giay', 'ao-khoac', 'vay', 'phu-kien', 'mu')",
      "primary_id": "The UUID of the selected primary item",
      "alternative_ids": ["The UUID of the first alternative item", "The UUID of the second alternative item"]
    }
  ]
}

STYLING RULES:
1. You MUST ONLY choose wardrobe item IDs from the CANDIDATE LIST provided below. Never make up or hallucinate IDs.
2. Based on the weather and occasion, determine the roles to include:
   - If cold/cool/rainy/windy, include an outerwear ('ao-khoac').
   - If a dress ('vay') is selected, do not include both a top ('ao') and a bottom ('quan'). Just match Dress + Shoes (+ Outerwear if cold).
   - Default combination: Top ('ao') + Bottom ('quan') + Shoes ('giay').
3. For each role, select exactly 1 primary item and up to 2 alternative items of the same role from the candidate list.
4. Try to coordinate colors harmoniously (complementary or analogous color coordination based on the HSL values).

CONTEXT AND REQUIREMENTS:
`)
	if input.Occasion != nil {
		fmt.Fprintf(&builder, "- Occasion: %s\n", *input.Occasion)
	}
	if input.StyleTarget != nil {
		fmt.Fprintf(&builder, "- Style Target: %s\n", *input.StyleTarget)
	}
	if input.Season != nil {
		fmt.Fprintf(&builder, "- Season: %s\n", *input.Season)
	}
	if input.Weather != nil {
		fmt.Fprintf(&builder, "- Weather: %s\n", *input.Weather)
	}
	if input.Details != nil {
		fmt.Fprintf(&builder, "- User Details: %s\n", *input.Details)
	}
	if input.ColorTone != nil {
		fmt.Fprintf(&builder, "- Color Tone: %s\n", *input.ColorTone)
	}

	builder.WriteString("\nCANDIDATE WARDROBE ITEMS (ONLY CHOOSE FROM HERE):\n")
	for _, item := range candidates {
		catSlug := ""
		if item.Category != nil {
			catSlug = item.Category.Slug
		}
		colorStr := ""
		if item.Color != nil {
			colorStr = *item.Color
		}
		hueStr := "nil"
		if item.ColorHue != nil {
			hueStr = fmt.Sprintf("%.1f", *item.ColorHue)
		}
		satStr := "nil"
		if item.ColorSaturation != nil {
			satStr = fmt.Sprintf("%.1f", *item.ColorSaturation)
		}
		lightStr := "nil"
		if item.ColorLightness != nil {
			lightStr = fmt.Sprintf("%.1f", *item.ColorLightness)
		}
		styleStr := ""
		if item.Style != nil {
			styleStr = *item.Style
		}
		descStr := ""
		if item.Description != nil {
			descStr = *item.Description
		}
		fmt.Fprintf(&builder, "- ID: %s | Category: %s | Color: %s (H:%s, S:%s, L:%s) | Style: %s | Description: %s\n",
			item.ID, catSlug, colorStr, hueStr, satStr, lightStr, styleStr, descStr)
	}

	return builder.String()
}

func buildChatSystemPrompt(summary string, wardrobeItems []*entities.WardrobeItem, recent []*entities.Message) string {
	var builder strings.Builder
	builder.WriteString("You are the AI fashion stylist of Smart Wardrobe. You must reply to the user in natural, friendly Vietnamese. Only recommend items from the user's available wardrobe items listed below. Do not suggest buying external products.\n")
	if strings.TrimSpace(summary) != "" {
		builder.WriteString("Summary of previous conversation:\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}

	builder.WriteString("Available wardrobe items:\n")
	limit := min(len(wardrobeItems), 20)
	for i := range limit {
		item := wardrobeItems[i]
		builder.WriteString("- ")
		if item.Category != nil {
			builder.WriteString(item.Category.Name)
			builder.WriteString(" ")
		}
		if item.Color != nil {
			builder.WriteString(*item.Color)
			builder.WriteString(" ")
		}
		if item.Style != nil {
			builder.WriteString(*item.Style)
			builder.WriteString(" ")
		}
		builder.WriteString("\n")
	}

	builder.WriteString("5 most recent messages:\n")
	for _, item := range recent {
		fmt.Fprintf(&builder, "%s: %s\n", item.Sender, item.Content)
	}

	return builder.String()
}

func isOutfitIntent(content string) bool {
	lowered := strings.ToLower(content)
	keywords := []string{"phoi do", "outfit", "mac gi", "goi y do", "chon quan ao", "phoi cho toi"}
	for _, keyword := range keywords {
		if strings.Contains(lowered, keyword) {
			return true
		}
	}
	return false
}

func isNeutral(item *entities.WardrobeItem) bool {
	if item.ColorLightness == nil || item.ColorSaturation == nil {
		return true
	}
	l := *item.ColorLightness
	s := *item.ColorSaturation
	return l <= 10.0 || l >= 90.0 || s <= 10.0
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

	// 1. Tổ hợp màu tương đồng: delta H < 30 và chênh lệch L >= 15
	if deltaH < 30.0 {
		l1 := 0.0
		l2 := 0.0
		if item1.ColorLightness != nil {
			l1 = *item1.ColorLightness
		}
		if item2.ColorLightness != nil {
			l2 = *item2.ColorLightness
		}
		if math.Abs(l1-l2) >= 15.0 {
			return true
		}
	}

	// 2. Tổ hợp màu bổ sung: 165 <= delta H <= 195
	if deltaH >= 165.0 && deltaH <= 195.0 {
		return true
	}

	return false
}

func runLocalHSLMatching(candidates []*entities.WardrobeItem, input dto.RecommendOutfitReq) *dto.RecommendedOutfitRes {
	var tops []*entities.WardrobeItem
	var bottoms []*entities.WardrobeItem
	var shoes []*entities.WardrobeItem
	var outerwears []*entities.WardrobeItem
	var dresses []*entities.WardrobeItem
	var accessories []*entities.WardrobeItem

	for _, item := range candidates {
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
		wLower := strings.ToLower(*input.Weather)
		if strings.Contains(wLower, "lạnh") || strings.Contains(wLower, "mát") ||
			strings.Contains(wLower, "cold") || strings.Contains(wLower, "cool") ||
			strings.Contains(wLower, "mưa") || strings.Contains(wLower, "rain") {
			isCold = true
		}
	}

	var items []*dto.RecommendedItemGroup

	if len(dresses) > 0 && len(shoes) > 0 && (len(tops) == 0 || len(bottoms) == 0) {
		primaryDress := dresses[0]
		var primaryShoes *entities.WardrobeItem
		for _, s := range shoes {
			if colorsMatch(primaryDress, s) {
				primaryShoes = s
				break
			}
		}
		if primaryShoes == nil {
			primaryShoes = shoes[0]
		}

		items = append(items, buildItemGroup("vay", primaryDress, dresses))
		items = append(items, buildItemGroup("giay", primaryShoes, shoes))

		if isCold && len(outerwears) > 0 {
			primaryOuter := outerwears[0]
			for _, o := range outerwears {
				if colorsMatch(primaryDress, o) {
					primaryOuter = o
					break
				}
			}
			items = append(items, buildItemGroup("ao-khoac", primaryOuter, outerwears))
		}
	} else if len(tops) > 0 && len(bottoms) > 0 {
		var primaryTop *entities.WardrobeItem
		var primaryBottom *entities.WardrobeItem

		foundMatch := false
		for _, t := range tops {
			for _, b := range bottoms {
				if colorsMatch(t, b) {
					primaryTop = t
					primaryBottom = b
					foundMatch = true
					break
				}
			}
			if foundMatch {
				break
			}
		}

		if !foundMatch {
			primaryTop = tops[0]
			primaryBottom = bottoms[0]
		}

		items = append(items, buildItemGroup("ao", primaryTop, tops))
		items = append(items, buildItemGroup("quan", primaryBottom, bottoms))

		if len(shoes) > 0 {
			var primaryShoes *entities.WardrobeItem
			for _, s := range shoes {
				if colorsMatch(primaryTop, s) || colorsMatch(primaryBottom, s) {
					primaryShoes = s
					break
				}
			}
			if primaryShoes == nil {
				primaryShoes = shoes[0]
			}
			items = append(items, buildItemGroup("giay", primaryShoes, shoes))
		}

		if isCold && len(outerwears) > 0 {
			var primaryOuter *entities.WardrobeItem
			for _, o := range outerwears {
				if colorsMatch(primaryTop, o) || colorsMatch(primaryBottom, o) {
					primaryOuter = o
					break
				}
			}
			if primaryOuter == nil {
				primaryOuter = outerwears[0]
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
		Title:       "Bộ phối màu hài hòa (Dự phòng HSL)",
		Explanation: explanation,
		Items:       items,
		IsFallback:  true,
	}
}

func buildItemGroup(role string, primary *entities.WardrobeItem, all []*entities.WardrobeItem) *dto.RecommendedItemGroup {
	var alternatives []*dto.WardrobeItemRes
	count := 0
	for _, item := range all {
		if item.ID == primary.ID {
			continue
		}
		if count >= 2 {
			break
		}
		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(item))
		count++
	}

	return &dto.RecommendedItemGroup{
		Role:         role,
		Primary:      mapper.MapToWardrobeItemRes(primary),
		Alternatives: alternatives,
	}
}

func mapChatSession(item *entities.ConversationalContext) *dto.ChatSessionRes {
	return &dto.ChatSessionRes{
		ID:             item.ID,
		Title:          item.Title,
		ContextSummary: item.ContextSummary,
		IsArchived:     item.IsArchived,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func mapChatMessage(item *entities.Message) *dto.ChatMessageRes {
	return &dto.ChatMessageRes{
		ID:        item.ID,
		Sender:    item.Sender,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}

// scoreCandidateItem runs a rule-based re-ranking score for a single wardrobe item based on parsed user intent:
// - Style matching (+0.3 bonus)
// - Occasion matching (+0.2 bonus)
// - Color tone matching (light/dark/earthy) (+0.2 bonus)
// - Neutral color versatility (+0.1 bonus)
// - Weather appropriateness (outerwear bonus in cold weather, penalty in hot weather)
// - Recently Worn Penalty (up to -0.4 penalty if worn within recentlyWornDays)
// - Long Unworn / New Item Bonus (+0.15 bonus if not worn recently and matches style or occasion)
func scoreCandidateItem(
	item *entities.WardrobeItem,
	intent dto.ParsedIntent,
	recentlyWornDays int,
	longUnwornDays int,
) (float64, []string) {
	var score float64 = 1.0
	var reasonTags []string

	// Rule 1: Style Matching
	styleMatched := false
	if item.Style != nil && len(intent.StyleTarget) > 0 {
		itemStyle := strings.ToLower(*item.Style)
		for _, target := range intent.StyleTarget {
			if strings.Contains(itemStyle, strings.ToLower(target)) {
				score += 0.3
				styleMatched = true
				reasonTags = append(reasonTags, "style-match:"+target)
				break
			}
		}
	}

	// Rule 2: Occasion Matching (checked against item description and style properties)
	if intent.Occasion != "" {
		occ := strings.ToLower(intent.Occasion)
		descMatch := false
		if item.Description != nil && strings.Contains(strings.ToLower(*item.Description), occ) {
			descMatch = true
		}
		if item.Style != nil && strings.Contains(strings.ToLower(*item.Style), occ) {
			descMatch = true
		}
		if descMatch {
			score += 0.2
			reasonTags = append(reasonTags, "occasion-match:"+occ)
		}
	}

	// Rule 3: Color Tone Matching using HSL values
	if intent.ColorTone != "" && item.ColorLightness != nil {
		tone := strings.ToLower(intent.ColorTone)
		l := *item.ColorLightness
		if tone == "light" && l >= 60.0 {
			// Light color lightness threshold >= 60%
			score += 0.2
			reasonTags = append(reasonTags, "color-tone:light")
		} else if tone == "dark" && l <= 40.0 {
			// Dark color lightness threshold <= 40%
			score += 0.2
			reasonTags = append(reasonTags, "color-tone:dark")
		} else if tone == "earthy" && item.ColorHue != nil && item.ColorSaturation != nil {
			h := *item.ColorHue
			s := *item.ColorSaturation
			// Earthy tones (browns, beiges, olives): Hue 20-50 or 80-120, Saturation <= 50, Lightness 20-70
			if ((h >= 20.0 && h <= 50.0) || (h >= 80.0 && h <= 120.0)) && s <= 50.0 && l >= 20.0 && l <= 70.0 {
				score += 0.2
				reasonTags = append(reasonTags, "color-tone:earthy")
			}
		}
	}

	// Rule 4: Neutral Color Versatility (neutrals coordinate easily, so they receive a small bonus)
	if isNeutral(item) {
		score += 0.1
		reasonTags = append(reasonTags, "neutral-versatility")
	}

	// Rule 5: Weather Appropriateness
	isColdContext := false
	for _, pc := range intent.PositiveConstraints {
		if pc == "cold" || pc == "cool" || pc == "rainy" {
			isColdContext = true
			break
		}
	}

	if item.Category != nil {
		slug := item.Category.Slug
		if isColdContext && slug == "ao-khoac" {
			// Cold weather demands outerwear (+0.4)
			score += 0.4
			reasonTags = append(reasonTags, "weather-appropriate:outerwear")
		} else if !isColdContext && slug == "ao-khoac" {
			// Hot weather discourages outerwear (-0.3)
			score -= 0.3
		}
	}

	// Rule 6: Wear Frequency Scoring (Recently Worn Penalty / Long Unworn Bonus)
	if item.LastUsedAt != nil {
		duration := time.Since(*item.LastUsedAt)
		daysSinceUsed := int(duration.Hours() / 24)

		if daysSinceUsed < recentlyWornDays {
			// Penalize linearly based on how recently the item was worn (up to -0.4)
			penalty := float64(recentlyWornDays-daysSinceUsed) / float64(recentlyWornDays) * 0.4
			score -= penalty
			reasonTags = append(reasonTags, fmt.Sprintf("recently-worn-penalty:-%.2f", penalty))
		} else if daysSinceUsed > longUnwornDays {
			// Award a bonus if the item hasn't been worn in a long time and fits the context
			if styleMatched || intent.Occasion != "" {
				score += 0.15
				reasonTags = append(reasonTags, "long-unworn-bonus")
			}
		}
	} else {
		// Award bonus to new/never-worn items if they match style or occasion
		if styleMatched || intent.Occasion != "" {
			score += 0.15
			reasonTags = append(reasonTags, "new-item-bonus")
		}
	}

	return score, reasonTags
}



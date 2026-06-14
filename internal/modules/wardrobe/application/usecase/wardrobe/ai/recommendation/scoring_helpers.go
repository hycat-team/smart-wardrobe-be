package recommendation

import (
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func scoreCandidateItem(
	item *entities.WardrobeItem,
	intent dto.ParsedIntent,
	recentlyWornDays,
	longUnwornDays int,
) (float64, []string) {
	score := 1.0
	var reasonTags []string
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

	if intent.Occasion != "" {
		occasion := strings.ToLower(intent.Occasion)
		descMatch := item.Description != nil && strings.Contains(strings.ToLower(*item.Description), occasion) ||
			item.Style != nil && strings.Contains(strings.ToLower(*item.Style), occasion)
		if descMatch {
			score += 0.2
			reasonTags = append(reasonTags, "occasion-match:"+occasion)
		}
	}

	if intent.ColorTone != "" && item.ColorLightness != nil {
		tone := strings.ToLower(intent.ColorTone)
		lightness := *item.ColorLightness
		if tone == "light" && lightness >= 60 {
			score += 0.2
			reasonTags = append(reasonTags, "color-tone:light")
		} else if tone == "dark" && lightness <= 40 {
			score += 0.2
			reasonTags = append(reasonTags, "color-tone:dark")
		} else if tone == "earthy" && item.ColorHue != nil && item.ColorSaturation != nil {
			hue := *item.ColorHue
			saturation := *item.ColorSaturation
			if ((hue >= 20 && hue <= 50) || (hue >= 80 && hue <= 120)) &&
				saturation <= 50 && lightness >= 20 && lightness <= 70 {
				score += 0.2
				reasonTags = append(reasonTags, "color-tone:earthy")
			}
		}
	}

	if isNeutral(item) {
		score += 0.1
		reasonTags = append(reasonTags, "neutral-versatility")
	}

	isCold := false
	for _, constraint := range intent.PositiveConstraints {
		if constraint == "cold" || constraint == "cool" || constraint == "rainy" {
			isCold = true
			break
		}
	}

	if item.Category != nil {
		slug := item.Category.Slug
		if isCold && slug == "ao-khoac" {
			score += 0.4
			reasonTags = append(reasonTags, "weather-appropriate:outerwear")
		} else if !isCold && slug == "ao-khoac" {
			score -= 0.3
		}
	}

	if item.LastUsedAt != nil {
		days := int(time.Since(*item.LastUsedAt).Hours() / 24)
		if days < recentlyWornDays {
			penalty := float64(recentlyWornDays-days) / float64(recentlyWornDays) * 0.4
			score -= penalty
			reasonTags = append(reasonTags, fmt.Sprintf("recently-worn-penalty:-%.2f", penalty))
		} else if days > longUnwornDays && (styleMatched || intent.Occasion != "") {
			score += 0.15
			reasonTags = append(reasonTags, "long-unworn-bonus")
		}
	} else if styleMatched || intent.Occasion != "" {
		score += 0.15
		reasonTags = append(reasonTags, "new-item-bonus")
	}

	return score, reasonTags
}

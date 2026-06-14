package recommendation

import (
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func buildRecommendationPrompt(candidates []CandidateForPrompt, input dto.RecommendOutfitReq) string {
	var builder strings.Builder
	builder.WriteString(`You are a professional AI fashion stylist. Your task is to recommend exactly 1 styled outfit from the user's available wardrobe candidates based on their weather and occasion requirements.
OUTPUT REQUIREMENT:
Return ONLY a valid JSON string. Do NOT wrap the response in markdown formatting (like json code blocks). The JSON object MUST follow this exact structure:
{"title":"","explanation":"","items":[{"role":"","primary_id":"","alternative_ids":["",""]}]}
STYLING RULES:
1. You MUST ONLY choose wardrobe item IDs from the CANDIDATE LIST provided below. Never make up or hallucinate IDs.
2. Based on the weather and occasion, determine the roles to include.
3. For each role, select exactly 1 primary item and up to 2 alternative items of the same role from the candidate list.
4. Try to coordinate colors harmoniously.
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
	for _, candidate := range candidates {
		item := candidate.Item
		catSlug, colorStr, styleStr, descStr := "", "", "", ""

		if item.Category != nil {
			catSlug = item.Category.Slug
		}
		if item.Color != nil {
			colorStr = *item.Color
		}
		if item.Style != nil {
			styleStr = *item.Style
		}
		if item.Description != nil {
			descStr = *item.Description
		}
		if len(candidate.Tags) > 0 {
			if descStr != "" {
				descStr += " "
			}
			descStr += "[Fashion Tags: " + strings.Join(candidate.Tags, ", ") + "]"
		}

		hueStr, satStr, lightStr := "nil", "nil", "nil"
		if item.ColorHue != nil {
			hueStr = fmt.Sprintf("%.1f", *item.ColorHue)
		}
		if item.ColorSaturation != nil {
			satStr = fmt.Sprintf("%.1f", *item.ColorSaturation)
		}
		if item.ColorLightness != nil {
			lightStr = fmt.Sprintf("%.1f", *item.ColorLightness)
		}

		fmt.Fprintf(
			&builder,
			"- ID: %s | Category: %s | Color: %s (H:%s, S:%s, L:%s) | Style: %s | Description: %s\n",
			item.ID,
			catSlug,
			colorStr,
			hueStr,
			satStr,
			lightStr,
			styleStr,
			descStr,
		)
	}

	return builder.String()
}

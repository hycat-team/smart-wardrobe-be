package recommendation

import (
	"encoding/json"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func buildRecommendationPrompt(candidates []CandidateForPrompt, input dto.RecommendOutfitReq) string {
	var builder strings.Builder
	builder.WriteString("Role: senior fashion stylist and wardrobe editor.\n")
	builder.WriteString("Task: recommend exactly one outfit from the provided wardrobe candidates.\n")
	builder.WriteString("Decision priorities: occasion fit, weather fit, season fit, silhouette balance, color harmony, style consistency, and practical wearability.\n")
	builder.WriteString("Editorial rule: stay honest to the actual items. Do not describe a graphic or visually loud item as fully minimalist, formal, or understated unless the item data clearly supports that claim.\n")
	builder.WriteString("Constraint rule: if the candidate pool is imperfect, choose the most suitable combination available and explain it truthfully rather than overselling it.\n")
	builder.WriteString("Alternative rule: items in alternative_ids must be viable, high-quality fashion substitutes for the primary_id that maintain the overall color harmony, style target, weather compatibility, and aesthetic of the recommended outfit.\n")
	builder.WriteString("Output contract: return exactly one minified JSON object with keys title, explanation, items.\n")
	builder.WriteString("Language: title and explanation must be natural Vietnamese with proper diacritics.\n")
	builder.WriteString("Rules: use only candidate IDs from CANDIDATES; each item entry must contain role (must match the candidate's category slug exactly, e.g. ao, quan, giay, ao-khoac, vay), primary_id, alternative_ids; do not output markdown, bullets, labels, or prose outside JSON; do not copy placeholder words such as string or uuid.\n")

	contextPayload := map[string]string{}
	if input.Occasion != nil {
		contextPayload["occasion"] = *input.Occasion
	}
	if input.StyleTarget != nil {
		contextPayload["style_target"] = *input.StyleTarget
	}
	if input.Season != nil {
		contextPayload["season"] = string(*input.Season)
	}
	if input.Weather != nil {
		contextPayload["weather"] = *input.Weather
	}
	if input.Details != nil {
		contextPayload["details"] = *input.Details
	}
	if input.ColorTone != nil {
		contextPayload["color_tone"] = *input.ColorTone
	}

	contextBytes, _ := json.Marshal(contextPayload)
	builder.WriteString("Required JSON shape example: {\"title\":\"Bộ đồ dạo phố gọn gàng cho ngày xuân\",\"explanation\":\"Bộ phối này ưu tiên sự dễ mặc, cân bằng màu sắc và phù hợp với thời tiết ấm. Nếu một món đồ có họa tiết nổi bật, phần mô tả phải phản ánh trung thực thay vì gọi đó là tối giản tuyệt đối.\",\"items\":[{\"role\":\"ao\",\"primary_id\":\"11111111-1111-1111-1111-111111111111\",\"alternative_ids\":[]},{\"role\":\"quan\",\"primary_id\":\"22222222-2222-2222-2222-222222222222\",\"alternative_ids\":[\"33333333-3333-3333-3333-333333333333\"]}]}\n")
	builder.WriteString("CONTEXT=")
	builder.Write(contextBytes)
	builder.WriteString("\n")
	builder.WriteString("CANDIDATES=\n")

	for _, candidate := range candidates {
		item := candidate.Item
		candidatePayload := map[string]any{
			"id": item.ID.String(),
		}

		if item.Category != nil {
			candidatePayload["category"] = item.Category.Slug
		}
		if item.Color != nil {
			candidatePayload["color"] = *item.Color
		}
		if item.Style != nil {
			candidatePayload["style"] = *item.Style
		}
		descStr := ""
		if item.Description != nil {
			descStr = *item.Description
		}
		if len(candidate.Tags) > 0 {
			if descStr != "" {
				descStr += " "
			}
			descStr += "[Fashion Tags: " + strings.Join(candidate.Tags, ", ") + "]"
		}
		if descStr != "" {
			candidatePayload["description"] = descStr
		}

		if item.ColorHue != nil {
			candidatePayload["color_h"] = fmt.Sprintf("%.1f", *item.ColorHue)
		}
		if item.ColorSaturation != nil {
			candidatePayload["color_s"] = fmt.Sprintf("%.1f", *item.ColorSaturation)
		}
		if item.ColorLightness != nil {
			candidatePayload["color_l"] = fmt.Sprintf("%.1f", *item.ColorLightness)
		}
		payloadBytes, _ := json.Marshal(candidatePayload)
		builder.Write(payloadBytes)
		builder.WriteString("\n")
	}

	return builder.String()
}

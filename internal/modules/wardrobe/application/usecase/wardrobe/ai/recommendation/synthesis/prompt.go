// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"encoding/json"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
)

type PromptLimits struct {
	CandidateLimit, DescriptionMaxCharacters, TagsLimit, PromptMaxCharacters int
}

// BuildRecommendationPrompt định dạng vai trò của stylist, các ràng buộc của prompt, ngữ cảnh yêu cầu và danh sách các ứng viên món đồ thành một prompt người dùng hoàn chỉnh gửi cho LLM.
//
// Hành vi:
// 1. Ghi thông tin vai trò stylist, nhiệm vụ, mức độ ưu tiên quyết định (phù hợp thời tiết, màu sắc, v.v.).
// 2. Định nghĩa các quy tắc đầu ra: ngôn ngữ tiếng Việt tự nhiên, cấu trúc JSON trả về với các khóa title, explanation, items.
// 3. Đóng gói ngữ cảnh yêu cầu của người dùng (dịp, phong cách, thời tiết, v.v.) dưới dạng JSON gắn sau nhãn "CONTEXT=".
// 4. Duyệt qua từng ứng viên và tạo thông tin chi tiết (bao gồm cả các tag đặc trưng thời trang được tính toán từ các bước trước), đóng gói thành các dòng JSON thô dưới nhãn "CANDIDATES=".
// 5. Trả về toàn bộ chuỗi prompt đã dựng.
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForPrompt{...}
//	input: dto.RecommendOutfitReq{Occasion: pointer to "đi chơi"}
//
// Đầu ra mẫu:
//
//	"Role: senior fashion stylist and wardrobe editor...\nCONTEXT={\"occasion\":\"đi chơi\"}\nCANDIDATES=\n{\"id\":\"uuid-1\",...}\n"
func BuildRecommendationPrompt(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq) string {
	prompt, _ := BuildRecommendationPromptWithLimits(candidates, input, PromptLimits{})
	return prompt
}

func BuildRecommendationPromptWithLimits(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, limits PromptLimits) (string, error) {
	if limits.CandidateLimit > 0 && len(candidates) > limits.CandidateLimit {
		candidates = candidates[:limits.CandidateLimit]
	}
	for len(candidates) > 0 {
		prompt := buildRecommendationPrompt(candidates, input, limits)
		if limits.PromptMaxCharacters <= 0 || len([]rune(prompt)) <= limits.PromptMaxCharacters {
			return prompt, nil
		}
		candidates = candidates[:len(candidates)-1]
	}
	return "", fmt.Errorf("recommendation prompt exceeds configured character limit")
}

func buildRecommendationPrompt(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, limits PromptLimits) string {
	var builder strings.Builder
	builder.WriteString("Role: senior fashion stylist and wardrobe editor.\n")
	builder.WriteString("Task: recommend exactly one outfit from the provided wardrobe candidates.\n")
	builder.WriteString("Decision priorities: occasion fit, weather fit, season fit, silhouette balance, color harmony, style consistency, and practical wearability.\n")
	builder.WriteString("Editorial rule: stay honest to the actual items. Do not describe a graphic or visually loud item as fully minimalist, formal, or understated unless the item data clearly supports that claim.\n")
	builder.WriteString("Constraint rule: if the candidate pool is imperfect, choose the most suitable combination available and explain it truthfully rather than overselling it.\n")
	builder.WriteString("Alternative rule: items in alternative_ids must be viable, high-quality fashion substitutes for the primary_id that maintain the overall color harmony, style target, weather compatibility, and aesthetic of the recommended outfit.\n")
	builder.WriteString("Output contract: return exactly one minified JSON object with keys title, explanation, items.\n")
	builder.WriteString("Language: title and explanation must be natural Vietnamese with proper diacritics.\n")
	builder.WriteString("Rules: use only candidate aliases from CANDIDATES (for example A1); each item entry must contain role matching the category slug, primary_id, alternative_ids; do not output markdown or prose outside JSON.\n")

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
	builder.WriteString("Required JSON shape example: {\"title\":\"Bộ đồ phù hợp\",\"explanation\":\"Giải thích ngắn gọn bằng tiếng Việt.\",\"items\":[{\"role\":\"ao\",\"primary_id\":\"A1\",\"alternative_ids\":[]},{\"role\":\"quan\",\"primary_id\":\"A2\",\"alternative_ids\":[\"A3\"]}]}\n")
	builder.WriteString("CONTEXT=")
	builder.Write(contextBytes)
	builder.WriteString("\n")
	builder.WriteString("CANDIDATES=\n")

	for candidateIndex, candidate := range candidates {
		item := candidate.Item
		candidatePayload := map[string]any{
			"id": fmt.Sprintf("A%d", candidateIndex+1),
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
			descStr = truncateRunes(*item.Description, limits.DescriptionMaxCharacters)
		}
		tags := candidate.Tags
		if limits.TagsLimit > 0 && len(tags) > limits.TagsLimit {
			tags = tags[:limits.TagsLimit]
		}
		if len(tags) > 0 {
			if descStr != "" {
				descStr += " "
			}
			descStr += "[Fashion Tags: " + strings.Join(tags, ", ") + "]"
		}
		if descStr != "" {
			candidatePayload["description"] = descStr
		}

		payloadBytes, _ := json.Marshal(candidatePayload)
		builder.Write(payloadBytes)
		builder.WriteString("\n")
	}

	return builder.String()
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

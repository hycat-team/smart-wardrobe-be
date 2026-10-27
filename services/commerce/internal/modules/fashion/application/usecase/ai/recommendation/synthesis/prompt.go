// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"encoding/json"
	"fmt"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_mapper "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/shared/pkg/utils/stringutils"

	"github.com/google/uuid"
)

type PromptLimits struct {
	CandidateLimit, DescriptionMaxCharacters, TagsLimit, PromptMaxCharacters int
}

// candidateRoleInventory summarizes which wearing roles are actually available
// in the prompt candidate pool. The small local model needs this explicit
// contract: when the pool only has a fullbody item, the example must show a
// fullbody outfit instead of teaching it to invent top/bottom aliases.
type candidateRoleInventory struct {
	aliases                        map[string][]string
	hasTop, hasBottom, hasFullbody bool
}

var promptWearingRoles = map[string]bool{
	"top": true, "bottom": true, "fullbody": true, "outerwear": true,
	"footwear": true, "headwear": true, "accessory": true,
}

func inventoryCandidateRoles(candidates []types.CandidateForPrompt, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) candidateRoleInventory {
	inventory := candidateRoleInventory{aliases: map[string][]string{}}
	for i, candidate := range candidates {
		if candidate.Item == nil || candidate.Item.FashionItemID == nil {
			continue
		}
		fashion := fashionItemMap[*candidate.Item.FashionItemID]
		if fashion == nil || fashion.Category == nil {
			continue
		}
		role := fashion_mapper.CategorySlugToWearingRole(fashion.Category.Slug)
		if !promptWearingRoles[role] {
			continue
		}
		inventory.aliases[role] = append(inventory.aliases[role], fmt.Sprintf("A%d", i+1))
		switch role {
		case "top":
			inventory.hasTop = true
		case "bottom":
			inventory.hasBottom = true
		case "fullbody":
			inventory.hasFullbody = true
		}
	}
	return inventory
}

func (inventory candidateRoleInventory) coreStrategy() string {
	if inventory.hasTop && inventory.hasBottom {
		return "separates"
	}
	if inventory.hasFullbody {
		return "fullbody"
	}
	return ""
}

func buildPromptExample(inventory candidateRoleInventory, strategy string) string {
	items := make([]map[string]any, 0, 5)
	add := func(role string) {
		aliases := inventory.aliases[role]
		if len(aliases) == 0 {
			return
		}
		items = append(items, map[string]any{
			"role":            role,
			"primary_id":      aliases[0],
			"alternative_ids": []string{},
		})
	}

	switch strategy {
	case "fullbody":
		add("fullbody")
	case "separates":
		add("top")
		add("bottom")
	default:
		if len(inventory.aliases["top"]) > 0 && len(inventory.aliases["bottom"]) > 0 {
			add("top")
			add("bottom")
		} else {
			add("fullbody")
		}
	}
	add("footwear")
	add("headwear")
	add("accessory")

	example := map[string]any{
		"title":       "Bộ đồ phù hợp",
		"explanation": "Mô tả ngắn gọn vì sao các món đồ này hợp với bối cảnh.",
		"items":       items,
	}
	raw, _ := json.Marshal(example)
	return string(raw)
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
func BuildRecommendationPrompt(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
	prompt, _ := BuildRecommendationPromptWithLimits(candidates, input, PromptLimits{}, fashionItemMap)
	return prompt
}

func BuildRecommendationPromptWithLimits(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, limits PromptLimits, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) (string, error) {
	if limits.CandidateLimit > 0 && len(candidates) > limits.CandidateLimit {
		candidates = candidates[:limits.CandidateLimit]
	}
	for len(candidates) > 0 {
		prompt := buildRecommendationPrompt(candidates, input, limits, fashionItemMap)
		if limits.PromptMaxCharacters <= 0 || len([]rune(prompt)) <= limits.PromptMaxCharacters {
			return prompt, nil
		}
		candidates = candidates[:len(candidates)-1]
	}
	return "", fmt.Errorf("recommendation prompt exceeds configured character limit")
}

func buildRecommendationPrompt(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, limits PromptLimits, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
	inventory := inventoryCandidateRoles(candidates, fashionItemMap)
	strategy := inventory.coreStrategy()
	roleAliases, _ := json.Marshal(inventory.aliases)

	var builder strings.Builder
	builder.WriteString("Role: senior fashion stylist and wardrobe editor.\n")
	builder.WriteString("Task: recommend exactly one outfit from the provided wardrobe candidates.\n")
	builder.WriteString("Decision priorities: occasion fit, weather fit, season fit, silhouette balance, color harmony, style consistency, and practical wearability.\n")
	builder.WriteString("Editorial rule: stay honest to the actual items. Do not describe a graphic or visually loud item as fully minimalist, formal, or understated unless the item data clearly supports that claim.\n")
	builder.WriteString("Constraint rule: if the candidate pool is imperfect, choose the most suitable combination available and explain it truthfully rather than overselling it.\n")
	builder.WriteString("Alternative rule: items in alternative_ids must be viable, high-quality fashion substitutes for the primary_id that maintain the overall color harmony, style target, weather compatibility, and aesthetic of the recommended outfit.\n")
	builder.WriteString("Output contract: return exactly one minified JSON object with keys title, explanation, items.\n")
	builder.WriteString("Language: title and explanation must be natural Vietnamese with proper diacritics.\n")
	builder.WriteString("Rules: aliases (A1, A2, ...) are internal IDs and MUST appear ONLY in primary_id and alternative_ids fields; NEVER write aliases in title or explanation (no \"(A1)\", no bare \"A1\"); each item entry must contain role matching the wearing position (top, bottom, fullbody, outerwear, footwear, headwear, accessory); do not output markdown or prose outside JSON.\n")
	builder.WriteString("Brand item rule: candidates tagged with [Fashion Tags: ..., source:brand-item, ...] are brand partner items. Prefer including at least one of them as primary or alternative when they fit the style, occasion, and color harmony.\n")
	builder.WriteString("Outfit rules:\n")
	builder.WriteString("- CORE roles define a complete outfit: exactly one top (\"top\"), one bottom (\"bottom\"), and one footwear (\"footwear\"), OR one fullbody (\"fullbody\") plus one footwear.\n")
	builder.WriteString("- Follow CORE_STRATEGY exactly: \"separates\" means exactly one top and one bottom and NO fullbody; \"fullbody\" means exactly one fullbody and NO top or bottom.\n")
	builder.WriteString("- primary_id MUST be chosen from ROLE_ALIASES for the same role. Never use an alias from another role.\n")
	builder.WriteString("- Each role must appear at most once. Do NOT output two items with the same role.\n")
	builder.WriteString("- If the recommended outfit contains a fullbody item (\"fullbody\"), it MUST NOT include any top (\"top\") or bottom (\"bottom\").\n")
	builder.WriteString("- Outerwear (e.g., jackets, coats) can be layered over either a fullbody dress (\"fullbody\") OR a top-and-bottom combination (\"top\" + \"bottom\").\n")
	builder.WriteString("- If no suitable top candidate exists for a top-and-bottom outfit, fall back to a fullbody outfit instead of omitting the top.\n")
	builder.WriteString("Layering roles rule: headwear (\"headwear\") and accessory (\"accessory\") are independent roles that stack on top of the core outfit.\n")
	builder.WriteString("- These two roles are NOT mutually exclusive. Never choose one role over the other: if the pool has both a hat candidate and an accessory candidate, you MUST include BOTH roles in the same outfit.\n")
	builder.WriteString("- When a pool has a hat candidate, output a headwear entry. When it has an accessory candidate, output an accessory entry. Each of the two must be decided independently.\n")
	builder.WriteString("- Omitting an available hat or accessory makes the outfit incomplete. Only skip a role when the pool truly has no candidate for it.\n")
	builder.WriteString("Outerwear rule: outerwear (\"outerwear\") is an optional layer. Include it when the weather is cold/cool/rainy, when the season is autumn/winter, or when the context explicitly asks for a layered outfit. NEVER include heavy outerwear when the weather is hot or sunny.\n")

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
	builder.WriteString("CORE_STRATEGY=")
	builder.WriteString(strategy)
	builder.WriteString("\n")
	builder.WriteString("ROLE_ALIASES=")
	builder.Write(roleAliases)
	builder.WriteString("\n")
	builder.WriteString("Required JSON shape example: ")
	builder.WriteString(buildPromptExample(inventory, strategy))
	builder.WriteString("\n")
	builder.WriteString("CONTEXT=")
	builder.Write(contextBytes)
	builder.WriteString("\n")
	builder.WriteString("CANDIDATES=\n")

	for candidateIndex, candidate := range candidates {
		fashion := (*dto.FashionItemContractDTO)(nil)
		if candidate.Item.FashionItemID != nil {
			fashion = fashionItemMap[*candidate.Item.FashionItemID]
		}
		candidatePayload := map[string]any{
			"id": fmt.Sprintf("A%d", candidateIndex+1),
		}

		if fashion != nil && fashion.Category != nil {
			candidatePayload["category"] = fashion_mapper.CategorySlugToWearingRole(fashion.Category.Slug)
		}
		if fashion != nil && fashion.Color != nil {
			candidatePayload["color"] = *fashion.Color
		}
		if fashion != nil && fashion.Style != nil {
			candidatePayload["style"] = *fashion.Style
		}
		descStr := ""
		if fashion != nil && fashion.Description != nil {
			descStr = stringutils.TruncateRunes(*fashion.Description, limits.DescriptionMaxCharacters)
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

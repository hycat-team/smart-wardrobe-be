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

// recommendationSystemPrompt holds every instruction the outfit model must obey.
//
// All rules live here and nowhere else: the user prompt carries data only
// (CORE_STRATEGY, ROLE_ALIASES, the JSON example, CONTEXT, CANDIDATES). Before
// this, the rule block existed twice — once here and once in the user prompt —
// so a single behaviour change had to be made in two places and the two copies
// had already started to diverge. A stable system prefix also keeps provider
// prompt caching useful across calls.
//
// Every constraint below is load-bearing; see docs/system-design/ai-pipeline.md
// for why headwear/accessory independence and the outerwear rule exist. Do not
// drop a clause without checking that document and the synthesis tests.
const recommendationSystemPrompt = "Role: senior fashion stylist and wardrobe editor. Recommend exactly one outfit from the provided wardrobe candidates and return exactly one valid minified JSON object.\n" +
	"Decision priorities: occasion fit, weather fit, season fit, silhouette balance, color harmony, style consistency, practical wearability.\n" +
	"Output: exactly one JSON object with keys title, explanation, items. title and explanation MUST be natural Vietnamese with full diacritics. No markdown, no prose outside the JSON.\n" +
	"Outfit structure: follow CORE_STRATEGY exactly — \"separates\" = exactly one top plus one bottom and NO fullbody; \"fullbody\" = exactly one fullbody and NO top or bottom. The outfit also needs exactly one footwear entry, and each role appears at most once. primary_id MUST come from ROLE_ALIASES for that same role only. If the pool has no suitable top candidate, fall back to a fullbody outfit instead of omitting the top.\n" +
	"Layering: headwear, accessory and outerwear stack on the core outfit and are NOT mutually exclusive with each other; decide each one independently. A hat candidate MUST produce a headwear entry and an accessory candidate MUST produce an accessory entry; omitting an available one makes the outfit incomplete. Include outerwear only for cold/cool/rainy weather, autumn/winter, or a layered request; it may layer over a fullbody dress or over a top+bottom pair; never include heavy outerwear in hot or sunny weather.\n" +
	"Alternative rule: every alternative_ids entry must be a viable, high-quality substitute for that role's primary_id and must preserve the outfit's color harmony, style target, weather compatibility and aesthetic.\n" +
	"Honesty: stay faithful to the actual item attributes; never describe a graphic or visually loud item as minimalist, formal or understated unless the item data clearly supports it. With an imperfect pool, choose the most suitable available combination and explain it truthfully instead of overselling it.\n" +
	"Recency: candidates tagged recency-cooldown were worn or recommended very recently. When two same-role candidates are otherwise equivalent, prefer the untagged one so the wardrobe keeps rotating; if every candidate for a role carries the tag, still choose the best one instead of omitting the role.\n" +
	"Internal aliases (A1, A2, ...) MUST appear ONLY in primary_id and alternative_ids; NEVER write an alias in title or explanation (no \"(A1)\", no bare \"A1\").\n" +
	"Brand item rule: candidates tagged with [Fashion Tags: ..., source:brand-item, ...] are brand partner items; prefer including at least one of them as primary or alternative when it fits the style, occasion and color harmony.\n" +
	"Personal style rule: when CONTEXT contains personal_style, prefer items matching it whenever candidate quality is comparable (learned_style is a weaker signal); never present an item whose style is listed in avoid_styles as a style match."

// BuildRecommendationPromptWithLimits dựng prompt người dùng cho LLM gợi ý trang phục.
//
// Hành vi:
// 1. Cắt danh sách ứng viên theo CandidateLimit và lặp lại việc dựng prompt, giảm dần từ cuối danh sách, cho tới khi độ dài ≤ PromptMaxCharacters.
// 2. Với mỗi lần dựng: đóng gói ngữ cảnh yêu cầu (dịp, phong cách, thời tiết, sở thích cá nhân) sau nhãn "CONTEXT=", chọn chiến lược outfit (separates/fullbody), liệt kê alias theo role và từng ứng viên dưới nhãn "CANDIDATES=".
// 3. Toàn bộ LUẬT nằm ở [recommendationSystemPrompt]; hàm này chỉ sinh DỮ LIỆU.
//
// Đầu ra mẫu:
//
//	"CORE_STRATEGY=separates\nROLE_ALIASES={\"top\":[\"A1\"]}\nCONTEXT={\"occasion\":\"đi chơi\"}\nCANDIDATES=\n{\"id\":\"A1\",...}\n"
func BuildRecommendationPrompt(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) string {
	prompt, _ := BuildRecommendationPromptWithLimits(candidates, input, PromptLimits{}, fashionItemMap, types.PersonalizationContext{})
	return prompt
}

// BuildRecommendationPromptWithPersonalization renders the prompt with the
// user's saved profile appended to the context payload (033 US1/T032).
func BuildRecommendationPromptWithPersonalization(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO, pc types.PersonalizationContext) string {
	prompt, _ := BuildRecommendationPromptWithLimits(candidates, input, PromptLimits{}, fashionItemMap, pc)
	return prompt
}

func BuildRecommendationPromptWithLimits(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, limits PromptLimits, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO, pc types.PersonalizationContext) (string, error) {
	if limits.CandidateLimit > 0 && len(candidates) > limits.CandidateLimit {
		candidates = candidates[:limits.CandidateLimit]
	}
	for len(candidates) > 0 {
		prompt := buildRecommendationPrompt(candidates, input, limits, fashionItemMap, pc)
		if limits.PromptMaxCharacters <= 0 || len([]rune(prompt)) <= limits.PromptMaxCharacters {
			return prompt, nil
		}
		candidates = candidates[:len(candidates)-1]
	}
	return "", fmt.Errorf("recommendation prompt exceeds configured character limit")
}

func buildRecommendationPrompt(candidates []types.CandidateForPrompt, input dto.RecommendOutfitReq, limits PromptLimits, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO, pc types.PersonalizationContext) string {
	inventory := inventoryCandidateRoles(candidates, fashionItemMap)
	strategy := inventory.coreStrategy()
	roleAliases, _ := json.Marshal(inventory.aliases)

	var builder strings.Builder
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
	// Personal profile context is only emitted when the user actually has
	// preferences to apply; keys are omitted entirely otherwise (T032).
	if len(pc.ExplicitStyles) > 0 {
		contextPayload["personal_style"] = strings.Join(pc.ExplicitStyles, ", ")
	}
	if len(pc.LearnedStyles) > 0 {
		contextPayload["learned_style"] = strings.Join(pc.LearnedStyles, ", ")
	}
	if len(pc.ExcludedStyles) > 0 {
		contextPayload["avoid_styles"] = strings.Join(pc.ExcludedStyles, ", ")
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

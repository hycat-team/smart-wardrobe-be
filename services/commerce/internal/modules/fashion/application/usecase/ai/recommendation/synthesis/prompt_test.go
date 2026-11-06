package synthesis

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
)

// The user prompt carries data only. Every rule lives in
// recommendationSystemPrompt so a behaviour change has exactly one home.
func TestBuildRecommendationPromptCarriesDataOnly(t *testing.T) {
	top, fTop := newSynthItem()
	fTop.Category.Slug = "ao"
	shoes, fShoes := newSynthItem()
	fShoes.Category.Slug = "giay"
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID:   fTop,
		fShoes.ID: fShoes,
	}
	candidates := []types.CandidateForPrompt{{Item: top}, {Item: shoes}}

	prompt := BuildRecommendationPrompt(candidates, dto.RecommendOutfitReq{}, fashionItemMap)

	for _, data := range []string{"CORE_STRATEGY=", "ROLE_ALIASES=", "CONTEXT=", "CANDIDATES="} {
		if !strings.Contains(prompt, data) {
			t.Fatalf("user prompt is missing data section %q, got:\n%s", data, prompt)
		}
	}

	// Instruction-shaped text must not come back into the user prompt.
	for _, rule := range []string{
		"Role: senior fashion stylist",
		"Outfit rules",
		"Layering roles rule",
		"Decision priorities",
		"Editorial rule",
	} {
		if strings.Contains(prompt, rule) {
			t.Fatalf("user prompt still contains the rule %q, got:\n%s", rule, prompt)
		}
	}
}

// Every constraint that used to live in the user prompt must survive the move.
// Dropping one silently degrades outfit quality, so the clauses are asserted by
// name.
func TestRecommendationSystemPromptKeepsEveryConstraint(t *testing.T) {
	required := []string{
		"senior fashion stylist",
		"Decision priorities",
		"title, explanation, items",
		"natural Vietnamese",
		"CORE_STRATEGY",
		`"separates"`,
		`"fullbody"`,
		"exactly one footwear",
		"at most once",
		"ROLE_ALIASES",
		"fall back to a fullbody",
		"headwear",
		"accessory",
		"outerwear",
		"NOT mutually exclusive",
		"alternative_ids",
		"color harmony",
		"Honesty",
		"recency-cooldown",
		"NEVER write an alias in title or explanation",
		"brand-item",
		"personal_style",
		"learned_style",
		"avoid_styles",
	}

	for _, clause := range required {
		if !strings.Contains(recommendationSystemPrompt, clause) {
			t.Fatalf("system prompt is missing the clause %q", clause)
		}
	}
}

// Guard against the prompt growing back into a second rule book. The merged
// prompt is ~2.9k characters; the old duplicated pair was ~5.1k.
func TestRecommendationSystemPromptStaysCompact(t *testing.T) {
	const maxCharacters = 3000
	if got := len([]rune(recommendationSystemPrompt)); got > maxCharacters {
		t.Fatalf("system prompt is %d characters, want <= %d", got, maxCharacters)
	}
}

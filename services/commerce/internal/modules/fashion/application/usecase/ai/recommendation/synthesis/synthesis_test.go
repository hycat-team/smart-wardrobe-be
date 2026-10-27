package synthesis

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_mapper "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
)

func TestBuildRecommendationPromptFormatsContextAndCandidates(t *testing.T) {
	occasion := "party"
	weather := "cold"
	req := dto.RecommendOutfitReq{
		Occasion: &occasion,
		Weather:  &weather,
	}

	item, fi := newSynthItem()
	desc := "cozy knit sweater"
	fi.Description = &desc
	fi.Category = &dto.CategoryBriefRes{
		ID:   uuid.New(),
		Name: "Ao",
		Slug: "ao",
	}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

	candidates := []types.CandidateForPrompt{
		{
			Item: item,
			Tags: []string{"winter-appropriate:outerwear"},
		},
	}

	prompt := BuildRecommendationPrompt(candidates, req, fashionItemMap)
	if !strings.Contains(prompt, `"occasion":"party"`) {
		t.Fatalf("expected occasion party in context, got: %s", prompt)
	}
	if !strings.Contains(prompt, `"id":"A1"`) {
		t.Fatalf("expected compact candidate alias in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "winter-appropriate:outerwear") {
		t.Fatalf("expected fashion tags in candidate description, got: %s", prompt)
	}
}

func TestBuildRecommendationPromptUsesFullbodyStrategyWhenOnlyFullbodyExists(t *testing.T) {
	dress, fDress := newSynthItem()
	fDress.Category.Slug = "dam"
	shoes, fShoes := newSynthItem()
	fShoes.Category.Slug = "giay"
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fDress.ID: fDress,
		fShoes.ID: fShoes,
	}
	candidates := []types.CandidateForPrompt{{Item: dress}, {Item: shoes}}

	prompt := BuildRecommendationPrompt(candidates, dto.RecommendOutfitReq{}, fashionItemMap)
	if !strings.Contains(prompt, "CORE_STRATEGY=fullbody") {
		t.Fatalf("expected fullbody strategy, got:\n%s", prompt)
	}
	a := strings.Index(prompt, `"fullbody":["A1"]`)
	if a < 0 {
		t.Fatalf("expected fullbody alias inventory, got:\n%s", prompt)
	}
	example := prompt[strings.Index(prompt, "Required JSON shape example:"):]
	if !strings.Contains(example, `"role":"fullbody"`) {
		t.Fatalf("expected fullbody example, got:\n%s", example)
	}
	if strings.Contains(example, `"role":"top"`) || strings.Contains(example, `"role":"bottom"`) {
		t.Fatalf("fullbody example must not teach top/bottom roles, got:\n%s", example)
	}
}

func TestBuildRecommendationPromptUsesSeparatesStrategyWhenTopAndBottomExist(t *testing.T) {
	top, fTop := newSynthItem()
	fTop.Category.Slug = "ao"
	pants, fPants := newSynthItem()
	fPants.Category.Slug = "quan"
	shoes, fShoes := newSynthItem()
	fShoes.Category.Slug = "giay"
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID:   fTop,
		fPants.ID: fPants,
		fShoes.ID: fShoes,
	}
	candidates := []types.CandidateForPrompt{{Item: top}, {Item: pants}, {Item: shoes}}

	prompt := BuildRecommendationPrompt(candidates, dto.RecommendOutfitReq{}, fashionItemMap)
	if !strings.Contains(prompt, "CORE_STRATEGY=separates") {
		t.Fatalf("expected separates strategy, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, `"top":["A1"]`) || !strings.Contains(prompt, `"bottom":["A2"]`) {
		t.Fatalf("expected role alias inventory, got:\n%s", prompt)
	}
	example := prompt[strings.Index(prompt, "Required JSON shape example:"):]
	if !strings.Contains(example, `"role":"top"`) || !strings.Contains(example, `"role":"bottom"`) {
		t.Fatalf("expected separates example, got:\n%s", example)
	}
	if strings.Contains(example, `"role":"fullbody"`) {
		t.Fatalf("separates example must not teach fullbody, got:\n%s", example)
	}
}

func TestBuildRecommendationPromptWithLimitsPreservesSourceAndBudget(t *testing.T) {
	longDescription := strings.Repeat("Ã¡", 300)
	candidates := make([]types.CandidateForPrompt, 4)
	fashionItemMap := make(map[uuid.UUID]*dto.FashionItemContractDTO)
	for i := range candidates {
		item, fi := newSynthItem()
		fi.Description = &longDescription
		fi.Category = &dto.CategoryBriefRes{
			ID:   uuid.New(),
			Name: "ao",
			Slug: "ao",
		}
		fashionItemMap[*item.FashionItemID] = fi
		candidates[i] = types.CandidateForPrompt{
			Item: item,
			Tags: []string{"one", "two", "three"},
		}
	}
	prompt, err := BuildRecommendationPromptWithLimits(candidates, dto.RecommendOutfitReq{}, PromptLimits{
		CandidateLimit: 2, DescriptionMaxCharacters: 20, TagsLimit: 2, PromptMaxCharacters: 5000,
	}, fashionItemMap)
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	if strings.Count(prompt, `"id"`) != 2 {
		t.Fatalf("expected two candidates")
	}
	if strings.Contains(prompt, "three") {
		t.Fatalf("expected excess tags to be omitted")
	}
	if longDescription != strings.Repeat("Ã¡", 300) {
		t.Fatal("source description was mutated")
	}
	if len([]rune(prompt)) > 5000 {
		t.Fatal("prompt exceeded character budget")
	}
}

func TestParseOutfitRecommendationJSONExtractsBalancedJSON(t *testing.T) {
	responseText := "Sure, here is the outfit: ```json\n{\n  \"title\": \"Dáº¡o phá»‘ xuÃ¢n\",\n  \"explanation\": \"Bá»™ trang phá»¥c thoáº£i mÃ¡i cho dáº¡o phá»‘.\",\n  \"items\": [\n    {\n      \"role\": \"ao\",\n      \"primary_id\": \"11111111-1111-1111-1111-111111111111\",\n      \"alternative_ids\": []\n    }\n  ]\n}\n``` Enjoy!"

	res, _, err := ParseOutfitRecommendationJSON(responseText)
	if err != nil {
		t.Fatalf("expected successful parse, got: %v", err)
	}
	if res.Title != "Dáº¡o phá»‘ xuÃ¢n" {
		t.Fatalf("expected title 'Dáº¡o phá»‘ xuÃ¢n', got: %q", res.Title)
	}
}

func TestValidateOutfitRecommendationPayloadRejectsPlaceholders(t *testing.T) {
	invalidPayload := types.LlmOutfitResponse{
		Title:       "Bá»™ Ä‘á»“ xuÃ¢n",
		Explanation: "explanation", // placeholder word
	}

	err := ValidateOutfitRecommendationPayload(invalidPayload)
	if err == nil {
		t.Fatal("expected failure due to placeholder explanation")
	}
}

func TestMapLLMResponseToGroupsHandlesValidCandidates(t *testing.T) {
	itemID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	item, fi := newSynthItem()
	fi.Category = &dto.CategoryBriefRes{
		ID:   uuid.New(),
		Name: "Ao",
		Slug: "ao",
	}
	item.ID = itemID
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}

	candidates := []types.CandidateForPrompt{
		{
			Item: item,
		},
	}

	llmRes := types.LlmOutfitResponse{
		Title:       "Outfit 1",
		Explanation: "Simple outfit",
		Items: []struct {
			Role           string   `json:"role"`
			PrimaryID      string   `json:"primary_id"`
			AlternativeIDs []string `json:"alternative_ids"`
		}{
			{
				Role:           "top",
				PrimaryID:      "A1",
				AlternativeIDs: []string{},
			},
		},
	}

	groups := fashion_mapper.MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)
	if len(groups) != 1 {
		t.Fatalf("expected 1 mapped group, got %d", len(groups))
	}
	if groups[0].Role != "top" || groups[0].Primary.ID != itemID {
		t.Fatalf("unexpected mapped group primary details: %+v", groups[0].Primary)
	}
}

func TestMapLLMResponseToGroupsHandlesNilFashionItemField(t *testing.T) {
	itemID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fashionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            itemID,
		FashionItemID: &fashionID,
	}

	fi := &dto.FashionItemContractDTO{
		ID:       fashionID,
		ImageUrl: "https://example.com/item.jpg",
		Color:    stringPtr("red"),
		Category: &dto.CategoryBriefRes{
			ID:   uuid.New(),
			Name: "Ao",
			Slug: "ao",
		},
	}

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{fashionID: fi}

	candidates := []types.CandidateForPrompt{
		{
			Item: item,
		},
	}

	llmRes := types.LlmOutfitResponse{
		Title:       "Outfit 1",
		Explanation: "Simple outfit",
		Items: []struct {
			Role           string   `json:"role"`
			PrimaryID      string   `json:"primary_id"`
			AlternativeIDs []string `json:"alternative_ids"`
		}{
			{
				Role:           "top",
				PrimaryID:      "A1",
				AlternativeIDs: []string{},
			},
		},
	}

	groups := fashion_mapper.MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)
	if len(groups) != 1 {
		t.Fatalf("expected 1 mapped group, got %d", len(groups))
	}
	primary := groups[0].Primary
	if primary == nil || primary.FashionItem == nil {
		t.Fatalf("expected primary.FashionItem to be populated, got nil")
	}
	if primary.FashionItem.ImageUrl != "https://example.com/item.jpg" {
		t.Fatalf("expected ImageUrl to be populated, got %s", primary.FashionItem.ImageUrl)
	}
}

func TestBuildRecommendationPromptForbidsAliasesInNarrative(t *testing.T) {
	item, fi := newSynthItem()
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Ao", Slug: "ao"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}
	prompt := BuildRecommendationPrompt([]types.CandidateForPrompt{{Item: item}}, dto.RecommendOutfitReq{}, fashionItemMap)
	if !strings.Contains(prompt, "NEVER write aliases in title or explanation") {
		t.Fatalf("expected prompt to forbid aliases in narrative, got: %s", prompt)
	}
}

func TestSanitizeOutfitNarrativeRemovesAliases(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "parenthesized aliases",
			input:    "Lá»±a chá»n chiáº¿c vÃ¡y ngáº¯n mÃ u xÃ¡m than (A7) vá»›i thiáº¿t káº¿ tá»‘i giáº£n. Káº¿t há»£p cÃ¹ng giÃ y sneaker Onitsuka Tiger (A2) hÃ i hÃ²a.",
			contains: []string{"vÃ¡y ngáº¯n mÃ u xÃ¡m than", "Onitsuka Tiger"},
		},
		{
			name:     "bare alias",
			input:    "Phá»‘i Ã¡o A1 cÃ¹ng quáº§n A3 ráº¥t Ä‘áº¹p.",
			contains: []string{"Phá»‘i Ã¡o", "quáº§n"},
		},
		{
			name:     "empty string",
			input:    "",
			contains: []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeOutfitNarrative(tc.input)
			if matched := bareAliasPattern.FindString(got); matched != "" {
				t.Fatalf("expected no alias in output, found %q in %q", matched, got)
			}
			if parenthesizedAliasPattern.FindString(got) != "" {
				t.Fatalf("expected no parenthesized alias in output, got %q", got)
			}
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("expected output to keep %q, got %q", want, got)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestClassifyFallbackTraceClassifiesCorrectly(t *testing.T) {
	err := NewFallbackTraceError("invalid_json", errors.New("unexpected EOF"), "user prompt", "raw response")
	kind, providerHint, promptLen, responsePreview := ClassifyFallbackTrace(err)

	if kind != "invalid_json" {
		t.Fatalf("expected kind 'invalid_json', got %q", kind)
	}
	if promptLen != len("user prompt") {
		t.Fatalf("expected prompt length %d, got %d", len("user prompt"), promptLen)
	}
	if !strings.Contains(responsePreview, "raw response") {
		t.Fatalf("expected response preview to contain raw response, got %q", responsePreview)
	}

	// Test provider classifications
	openaiErr := errors.New("OpenAI service temporary unavailable")
	_, providerHint, _, _ = ClassifyFallbackTrace(openaiErr)
	if providerHint != "openai" {
		t.Fatalf("expected providerHint 'openai', got %q", providerHint)
	}

	googleErr := errors.New("Google API error")
	_, providerHint, _, _ = ClassifyFallbackTrace(googleErr)
	if providerHint != "gemini" {
		t.Fatalf("expected providerHint 'gemini', got %q", providerHint)
	}

	timeoutErr := context.DeadlineExceeded
	_, providerHint, _, _ = ClassifyFallbackTrace(timeoutErr)
	if providerHint != "timeout" {
		t.Fatalf("expected providerHint 'timeout', got %q", providerHint)
	}
}

func TestMapLLMResponseToGroupsFiltersOutTopAndBottomWhenFullbodyIsPresent(t *testing.T) {
	topID := uuid.New()
	fullbodyID := uuid.New()
	accessoryID := uuid.New()

	topItem, topFi := newSynthItem()
	topFi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "ao", Slug: "ao"}
	topItem.ID = topID

	fullbodyItem, fullbodyFi := newSynthItem()
	fullbodyFi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "dam", Slug: "dam"}
	fullbodyItem.ID = fullbodyID

	accessoryItem, accessoryFi := newSynthItem()
	accessoryFi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "phu-kien", Slug: "phu-kien"}
	accessoryItem.ID = accessoryID

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*topItem.FashionItemID:       topFi,
		*fullbodyItem.FashionItemID:  fullbodyFi,
		*accessoryItem.FashionItemID: accessoryFi,
	}

	candidates := []types.CandidateForPrompt{
		{Item: topItem},
		{Item: fullbodyItem},
		{Item: accessoryItem},
	}

	llmRes := types.LlmOutfitResponse{
		Title:       "Mixed Outfit",
		Explanation: "Contains fullbody and top",
		Items: []struct {
			Role           string   `json:"role"`
			PrimaryID      string   `json:"primary_id"`
			AlternativeIDs []string `json:"alternative_ids"`
		}{
			{
				Role:           "top",
				PrimaryID:      "A1",
				AlternativeIDs: []string{},
			},
			{
				Role:           "fullbody",
				PrimaryID:      "A2",
				AlternativeIDs: []string{},
			},
			{
				Role:           "accessory",
				PrimaryID:      "A3",
				AlternativeIDs: []string{},
			},
		},
	}

	groups := fashion_mapper.MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	// Should have filtered out 'top', leaving only 'fullbody' and 'accessory'
	if len(groups) != 2 {
		t.Fatalf("expected 2 mapped groups after filtering, got %d", len(groups))
	}
	for _, g := range groups {
		if g.Role == "top" || g.Role == "bottom" {
			t.Fatalf("found illegal role %q in mapped groups after filtering", g.Role)
		}
	}
}

func newSynthItem() (*wardrobeDto.WardrobeItemContractDTO, *dto.FashionItemContractDTO) {
	fashionID := uuid.New()
	catID := uuid.New()
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionID,
	}
	res := &dto.FashionItemContractDTO{
		ID:       fashionID,
		Category: &dto.CategoryBriefRes{ID: catID},
	}
	return item, res
}

func strPtr(s string) *string { return &s }

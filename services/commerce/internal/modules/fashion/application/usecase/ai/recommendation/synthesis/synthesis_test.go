package synthesis

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_mapper "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/shared/config"
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

func TestBuildRecommendationPromptWithLimitsPreservesSourceAndBudget(t *testing.T) {
	longDescription := strings.Repeat("á", 300)
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
	}, fashionItemMap, types.PersonalizationContext{})
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	if strings.Count(prompt, `"id"`) != 2 {
		t.Fatalf("expected two candidates")
	}
	if strings.Contains(prompt, "three") {
		t.Fatalf("expected excess tags to be omitted")
	}
	if longDescription != strings.Repeat("á", 300) {
		t.Fatal("source description was mutated")
	}
	if len([]rune(prompt)) > 5000 {
		t.Fatal("prompt exceeded character budget")
	}
}

func TestParseOutfitRecommendationJSONExtractsBalancedJSON(t *testing.T) {
	responseText := "Sure, here is the outfit: ```json\n{\n  \"title\": \"Dạo phố xuân\",\n  \"explanation\": \"Bộ trang phục thoải mái cho dạo phố.\",\n  \"items\": [\n    {\n      \"role\": \"ao\",\n      \"primary_id\": \"11111111-1111-1111-1111-111111111111\",\n      \"alternative_ids\": []\n    }\n  ]\n}\n``` Enjoy!"

	res, _, err := ParseOutfitRecommendationJSON(responseText)
	if err != nil {
		t.Fatalf("expected successful parse, got: %v", err)
	}
	if res.Title != "Dạo phố xuân" {
		t.Fatalf("expected title 'Dạo phố xuân', got: %q", res.Title)
	}
}

func TestValidateOutfitRecommendationPayloadRejectsPlaceholders(t *testing.T) {
	invalidPayload := types.LlmOutfitResponse{
		Title:       "Bộ đồ xuân",
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
			input:    "Lựa chọn chiếc váy ngắn màu xám than (A7) với thiết kế tối giản. Kết hợp cùng giày sneaker Onitsuka Tiger (A2) hài hòa.",
			contains: []string{"váy ngắn màu xám than", "Onitsuka Tiger"},
		},
		{
			name:     "bare alias",
			input:    "Phối áo A1 cùng quần A3 rất đẹp.",
			contains: []string{"Phối áo", "quần"},
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

func TestBuildRecommendationPromptOnlyEmitsNonEmptyPersonalizationKeys(t *testing.T) {
	item, fi := newSynthItem()
	fi.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Ao", Slug: "ao"}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{*item.FashionItemID: fi}
	candidates := []types.CandidateForPrompt{{Item: item}}

	keys := []string{"personal_style", "learned_style", "avoid_styles"}

	emptyPrompt := BuildRecommendationPromptWithPersonalization(
		candidates,
		dto.RecommendOutfitReq{},
		fashionItemMap,
		types.PersonalizationContext{},
	)
	for _, key := range keys {
		if strings.Contains(emptyPrompt, key) {
			t.Fatalf("expected no %q key without a profile, got: %s", key, emptyPrompt)
		}
	}

	profilePrompt := BuildRecommendationPromptWithPersonalization(
		candidates,
		dto.RecommendOutfitReq{},
		fashionItemMap,
		types.PersonalizationContext{
			Active:         true,
			ExplicitStyles: []string{"minimalist"},
			LearnedStyles:  []string{"vintage"},
			ExcludedStyles: []string{"streetwear"},
		},
	)
	for _, want := range []string{
		`"personal_style":"minimalist"`,
		`"learned_style":"vintage"`,
		`"avoid_styles":"streetwear"`,
	} {
		if !strings.Contains(profilePrompt, want) {
			t.Fatalf("expected %q in the context payload, got: %s", want, profilePrompt)
		}
	}
}

// fakeSynthesisAI records the prompts and answers with one scripted payload.
type fakeSynthesisAI struct {
	response     string
	err          error
	systemPrompt string
	userPrompt   string
	calls        int
}

func (f *fakeSynthesisAI) AnalyzeImage(context.Context, string, string) (string, error) {
	return "", nil
}

func (f *fakeSynthesisAI) GenerateEmbeddings(context.Context, []string) ([][]float32, error) {
	return nil, nil
}

func (f *fakeSynthesisAI) GenerateChatText(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (string, error) {
	return "", nil
}

func (f *fakeSynthesisAI) GenerateRecommendationText(
	_ context.Context,
	systemPrompt string,
	userPrompt string,
	_ aiexecutor_dto.OrchestratorOptions,
) (string, error) {
	f.calls++
	f.systemPrompt = systemPrompt
	f.userPrompt = userPrompt
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func (f *fakeSynthesisAI) GenerateChatTextStream(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (<-chan string, <-chan error) {
	text := make(chan string)
	errs := make(chan error)
	close(text)
	close(errs)
	return text, errs
}

func TestSystemPromptCarriesThePersonalStyleRule(t *testing.T) {
	top, topFashion := newSynthItem()
	topFashion.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Ao", Slug: "ao"}
	bottom, bottomFashion := newSynthItem()
	bottomFashion.Category = &dto.CategoryBriefRes{ID: uuid.New(), Name: "Quan", Slug: "quan"}

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:    topFashion,
		*bottom.FashionItemID: bottomFashion,
	}
	candidates := []types.CandidateForPrompt{{Item: top}, {Item: bottom}}
	ai := &fakeSynthesisAI{response: `{"title":"Bộ đồ tối giản","explanation":"Áo và quần hài hòa cho ngày làm việc.","items":[{"role":"top","primary_id":"A1","alternative_ids":[]},{"role":"bottom","primary_id":"A2","alternative_ids":[]}]}`}

	res, err := GenerateOutfitRecommendation(
		context.Background(),
		ai,
		uuid.New(),
		candidates,
		dto.RecommendOutfitReq{},
		&config.Config{},
		fashionItemMap,
		types.PersonalizationContext{ExplicitStyles: []string{"minimalist"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected an outfit response")
	}
	if ai.calls != 1 {
		t.Fatalf("expected one provider call, got %d", ai.calls)
	}
	for _, want := range []string{"Personal style rule", "personal_style", "learned_style", "avoid_styles"} {
		if !strings.Contains(ai.systemPrompt, want) {
			t.Fatalf("expected %q in the system prompt, got: %s", want, ai.systemPrompt)
		}
	}
	if !strings.Contains(ai.userPrompt, `"personal_style":"minimalist"`) {
		t.Fatalf("expected the personal style in the user prompt, got: %s", ai.userPrompt)
	}
}

func TestRecommendedOutfitResExposesRecommendationID(t *testing.T) {
	recommendationID := uuid.New()

	payload, err := json.Marshal(dto.RecommendedOutfitRes{
		RecommendationID: recommendationID,
		Title:            "Bộ đồ",
		Explanation:      "Mô tả",
	})
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if !strings.Contains(string(payload), `"recommendationId"`) {
		t.Fatalf("expected the recommendationId JSON field, got: %s", payload)
	}

	decoded := map[string]any{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if got, ok := decoded["recommendationId"]; !ok || got != recommendationID.String() {
		t.Fatalf("expected recommendationId=%s, got %v", recommendationID, got)
	}
}

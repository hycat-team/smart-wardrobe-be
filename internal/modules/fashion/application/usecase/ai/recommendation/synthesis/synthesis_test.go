package synthesis

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_mapper "smart-wardrobe-be/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
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

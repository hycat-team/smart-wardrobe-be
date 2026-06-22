package synthesis

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func TestBuildRecommendationPromptFormatsContextAndCandidates(t *testing.T) {
	occasion := "party"
	weather := "cold"
	req := dto.RecommendOutfitReq{
		Occasion: &occasion,
		Weather:  &weather,
	}

	itemID := uuid.New()
	desc := "cozy knit sweater"
	candidates := []types.CandidateForPrompt{
		{
			Item: &entities.WardrobeItem{
				AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: itemID}},
				Description:     &desc,
				Category:        &entities.Category{Slug: "ao", Name: "Ao"},
			},
			Tags: []string{"winter-appropriate:outerwear"},
		},
	}

	prompt := BuildRecommendationPrompt(candidates, req)
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
	for i := range candidates {
		candidates[i] = types.CandidateForPrompt{
			Item: &entities.WardrobeItem{
				AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
				Description:     &longDescription,
				Category:        &entities.Category{Slug: "ao"},
			},
			Tags: []string{"one", "two", "three"},
		}
	}
	prompt, err := BuildRecommendationPromptWithLimits(candidates, dto.RecommendOutfitReq{}, PromptLimits{
		CandidateLimit: 2, DescriptionMaxCharacters: 20, TagsLimit: 2, PromptMaxCharacters: 5000,
	})
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
	candidates := []types.CandidateForPrompt{
		{
			Item: &entities.WardrobeItem{
				AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: itemID}},
				Category:        &entities.Category{Slug: "ao", Name: "Ao"},
			},
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
				Role:           "ao",
				PrimaryID:      itemID.String(),
				AlternativeIDs: []string{},
			},
		},
	}

	groups := MapLLMResponseToGroups(candidates, llmRes)
	if len(groups) != 1 {
		t.Fatalf("expected 1 mapped group, got %d", len(groups))
	}
	if groups[0].Role != "ao" || groups[0].Primary.ID != itemID {
		t.Fatalf("unexpected mapped group primary details: %+v", groups[0].Primary)
	}
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

package retrieval

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func TestBuildRecommendationSemanticQueryAfterExplicitMerge(t *testing.T) {
	intent := dto.ParsedIntent{
		Occasion:    []string{"work"},
		ColorTone:   []string{"light"},
		StyleTarget: []string{"minimalist"},
	}

	query := BuildRecommendationSemanticQuery(intent, "di tiec mac mau toi", true)
	if !strings.Contains(query, "occasion: work") ||
		!strings.Contains(query, "color tone: light") ||
		!strings.Contains(query, "style: minimalist") {
		t.Fatalf("semantic query does not include final intent fields: %q", query)
	}
	if !strings.Contains(query, "details context: di tiec mac mau toi") {
		t.Fatalf("semantic query should keep details context when explicit options exist: %q", query)
	}
}

func TestBuildRecommendationSemanticQueryFromOptionsOnly(t *testing.T) {
	intent := dto.ParsedIntent{
		Occasion:  []string{"work"},
		ColorTone: []string{"light"},
	}

	query := BuildRecommendationSemanticQuery(intent, "", true)
	if query == "" {
		t.Fatal("expected semantic query from structured options")
	}
}

func TestNormalizeTermSetRemovesStopwordsAndDuplicates(t *testing.T) {
	actual := NormalizeTermSet([]string{"work", "mac", "WORK", "muon", "formal"})
	expected := []string{"formal", "work"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("NormalizeTermSet() = %v, expected %v", actual, expected)
	}
}

func TestLocalRecommendationQueryRewriterBuildsTaxonomyTermsAndHardFilters(t *testing.T) {
	intent := dto.ParsedIntent{
		Occasion:            []string{"work"},
		StyleTarget:         []string{"minimalist"},
		ColorTone:           []string{"light"},
		PositiveConstraints: []string{"winter"},
		ExcludedStyles:      []string{"streetwear"},
		ExcludedColorTones:  []string{"dark"},
		ExcludedWeather:     []string{"rainy"},
		SemanticQuery:       "occasion: work | color tone: light",
	}

	query, err := LocalRecommendationQueryRewriter{}.Rewrite(context.Background(), intent)
	if err != nil {
		t.Fatalf("unexpected local rewriter error: %v", err)
	}
	lexicalStrs := ExtractTermStrings(query.LexicalTerms)
	excludedStrs := ExtractTermStrings(query.ExcludedTerms)
	for _, term := range []string{"basic", "business", "cong so", "cream", "don gian", "office", "toi gian", "van phong", "winter"} {
		if !containsString(lexicalStrs, term) {
			t.Fatalf("expected lexical term %q in %v", term, lexicalStrs)
		}
	}
	for _, forbidden := range []string{"knitwear", "lightweight", "outerwear", "short sleeve"} {
		if containsString(lexicalStrs, forbidden) {
			t.Fatalf("did not expect generic taxonomy term %q in %v", forbidden, lexicalStrs)
		}
	}
	for _, term := range []string{"black", "dark", "den", "duong pho", "rain", "rainy", "streetwear", "waterproof"} {
		if !containsString(excludedStrs, term) {
			t.Fatalf("expected excluded term %q in %v", term, excludedStrs)
		}
	}
	if !reflect.DeepEqual(query.HardFilters.Seasonality, []string{"winter"}) {
		t.Fatalf("expected winter season hard filter, got %v", query.HardFilters.Seasonality)
	}
}

func TestTaxonomyExpansionDoesNotGenerateTermsOutsideAllowlist(t *testing.T) {
	intent := dto.ParsedIntent{
		ColorTone:           []string{"dark", "earthy", "light"},
		PositiveConstraints: []string{"winter", "cold", "hot", "rainy"},
	}

	query, err := LocalRecommendationQueryRewriter{}.Rewrite(context.Background(), intent)
	if err != nil {
		t.Fatalf("unexpected local rewriter error: %v", err)
	}
	lexicalStrs := ExtractTermStrings(query.LexicalTerms)
	for _, forbidden := range []string{"knitwear", "lightweight", "long sleeve", "outerwear", "short sleeve"} {
		if containsString(lexicalStrs, forbidden) {
			t.Fatalf("did not expect generic taxonomy term %q in %v", forbidden, lexicalStrs)
		}
	}
	for _, expected := range []string{"black", "brown", "cream", "den", "dong", "he", "lanh", "mua", "mua dong", "mua he", "nong", "trang", "waterproof"} {
		if !containsString(lexicalStrs, expected) {
			t.Fatalf("expected allowlisted term %q in %v", expected, lexicalStrs)
		}
	}
}

func TestLLMRecommendationQueryRewriterUsesValidatedOutput(t *testing.T) {
	rewriter := LLMRecommendationQueryRewriter{
		aiService: fakeRecommendationAIService{
			chatText: `{"semantic_query":"occasion: work","lexical_terms":["Office","formal","office"],"excluded_terms":["dark"],"hard_filters":{"seasonality":["winter"],"category_slugs":["ao"]}}`,
		},
		cfg: &config.Config{RAG: config.RAG{
			RecommendationRewriterMaxSemanticLength: 512,
			RecommendationRewriterMaxLexicalTerms:   24,
			RecommendationRewriterMaxExcludedTerms:  24,
		}},
	}

	query, err := rewriter.Rewrite(context.Background(), dto.ParsedIntent{Occasion: []string{"work"}, SemanticQuery: "occasion: work"})
	if err != nil {
		t.Fatalf("expected valid LLM rewrite, got %v", err)
	}
	if query.SemanticQuery != "occasion: work" {
		t.Fatalf("unexpected semantic query %q", query.SemanticQuery)
	}
	if !reflect.DeepEqual(ExtractTermStrings(query.LexicalTerms), []string{"formal", "office"}) {
		t.Fatalf("expected normalized lexical terms, got %v", query.LexicalTerms)
	}
	if !reflect.DeepEqual(ExtractTermStrings(query.ExcludedTerms), []string{"dark"}) {
		t.Fatalf("expected excluded terms, got %v", query.ExcludedTerms)
	}
	if !reflect.DeepEqual(query.HardFilters.Seasonality, []string{"winter"}) ||
		!reflect.DeepEqual(query.HardFilters.CategorySlugs, []string{"ao"}) {
		t.Fatalf("unexpected hard filters: %+v", query.HardFilters)
	}
}

func TestLLMRecommendationQueryRewriterRejectsInvalidOutput(t *testing.T) {
	rewriter := LLMRecommendationQueryRewriter{
		aiService: fakeRecommendationAIService{
			chatText: `{"semantic_query":"SELECT * FROM wardrobe_items","lexical_terms":["office"],"excluded_terms":[],"hard_filters":{"seasonality":["monsoon"],"category_slugs":["unknown"]}}`,
		},
		cfg: &config.Config{RAG: config.RAG{
			RecommendationRewriterMaxSemanticLength: 512,
			RecommendationRewriterMaxLexicalTerms:   24,
			RecommendationRewriterMaxExcludedTerms:  24,
		}},
	}

	if _, err := rewriter.Rewrite(context.Background(), dto.ParsedIntent{}); err == nil {
		t.Fatal("expected invalid LLM output to fail validation")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type fakeRecommendationAIService struct {
	chatText string
	err      error
}

func (f fakeRecommendationAIService) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	return "", nil
}

func (f fakeRecommendationAIService) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	return nil, nil
}

func (f fakeRecommendationAIService) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.chatText, nil
}

func (f fakeRecommendationAIService) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	return "", nil
}

func (f fakeRecommendationAIService) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string) (<-chan string, <-chan error) {
	text := make(chan string)
	errs := make(chan error)
	close(text)
	close(errs)
	return text, errs
}

func TestExcludedTaxonomyExpansion(t *testing.T) {
	intent := dto.ParsedIntent{
		NegativeConstraints: []string{"avoid-term:dark"},
	}

	query, err := LocalRecommendationQueryRewriter{}.Rewrite(context.Background(), intent)
	if err != nil {
		t.Fatalf("unexpected local rewriter error: %v", err)
	}

	// Should expand "dark" using taxonomyGroupExcluded to "black", "den", "gray", "grey", "xam", etc.
	expectedTerms := []string{"black", "dark", "den", "gray", "grey", "tram", "xam"}
	excludedStrs := ExtractTermStrings(query.ExcludedTerms)
	for _, term := range expectedTerms {
		if !containsString(excludedStrs, term) {
			t.Errorf("expected excluded term %q in %v", term, excludedStrs)
		}
	}
}

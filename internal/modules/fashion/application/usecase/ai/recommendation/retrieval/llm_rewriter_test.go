package retrieval

import (
	"context"
	"reflect"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
)

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

func (f fakeRecommendationAIService) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.chatText, nil
}

func (f fakeRecommendationAIService) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	return "", nil
}

func (f fakeRecommendationAIService) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (<-chan string, <-chan error) {
	text := make(chan string)
	errs := make(chan error)
	close(text)
	close(errs)
	return text, errs
}

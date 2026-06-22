package recommendation

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/retrieval"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
)

func TestBuildRecommendationRetrievalQueryFallsBackToLocalOnLLMError(t *testing.T) {
	uc := &OutfitRecommendationUseCase{
		cfg: &config.Config{RAG: config.RAG{
			RecommendationLLMRewriterEnabled:        true,
			RecommendationLLMRewriterTimeoutSeconds: 2,
		}},
		aiService: fakeRecommendationAIService{
			err: errors.New("provider unavailable"),
		},
	}
	intent := dto.ParsedIntent{
		Occasion:      []string{"work"},
		SemanticQuery: "occasion: work",
	}

	query := uc.buildRecommendationRetrievalQuery(context.Background(), uuid.Nil, intent)
	if query.SemanticQuery != "occasion: work" || !containsString(retrieval.ExtractTermStrings(query.LexicalTerms), "office") {
		t.Fatalf("expected local fallback query, got %+v", query)
	}
}

func TestBuildRecommendationRetrievalQueryUsesLLMWhenEnabled(t *testing.T) {
	uc := &OutfitRecommendationUseCase{
		cfg: &config.Config{RAG: config.RAG{
			RecommendationLLMRewriterEnabled:        true,
			RecommendationLLMRewriterTimeoutSeconds: 2,
		}},
		aiService: fakeRecommendationAIService{
			chatText: `{"semantic_query":"llm semantic","lexical_terms":["office"],"excluded_terms":["dark"],"hard_filters":{"seasonality":["winter"],"category_slugs":[]}}`,
		},
	}

	query := uc.buildRecommendationRetrievalQuery(context.Background(), uuid.Nil, dto.ParsedIntent{SemanticQuery: "local semantic"})
	if query.SemanticQuery != "llm semantic" {
		t.Fatalf("expected LLM query, got %+v", query)
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

package recommendation

import (
	"context"
	"errors"
	"math"
	"testing"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/shared/config"
)

// fakeQueryEmbeddingAI is a scripted IAIOrchestrator for the query-vector tests.
type fakeQueryEmbeddingAI struct {
	vector   []float32
	err      error
	calls    int
	requests []string
}

func (f *fakeQueryEmbeddingAI) AnalyzeImage(context.Context, string, string) (string, error) {
	return "", nil
}

func (f *fakeQueryEmbeddingAI) GenerateEmbeddings(_ context.Context, chunks []string) ([][]float32, error) {
	f.calls++
	f.requests = append(f.requests, chunks...)
	if f.err != nil {
		return nil, f.err
	}
	return [][]float32{f.vector}, nil
}

func (f *fakeQueryEmbeddingAI) GenerateChatText(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (string, error) {
	return "", nil
}

func (f *fakeQueryEmbeddingAI) GenerateRecommendationText(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (string, error) {
	return "", nil
}

func (f *fakeQueryEmbeddingAI) GenerateChatTextStream(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (<-chan string, <-chan error) {
	text := make(chan string)
	errs := make(chan error)
	close(text)
	close(errs)
	return text, errs
}

func queryVectorHarness(ai *fakeQueryEmbeddingAI) *OutfitRecommendationUseCase {
	return &OutfitRecommendationUseCase{
		cfg: &config.Config{RAG: config.RAG{
			RecommendationEmbeddingDimension:      768,
			RecommendationEmbeddingTimeoutSeconds: 2,
		}},
		logger:    nopRecommendationLogger{},
		aiService: ai,
	}
}

func TestBuildRecommendationQueryVectorBlendsNormalizedVectors(t *testing.T) {
	query := oneHotVector(0)
	style := oneHotVector(1)
	ai := &fakeQueryEmbeddingAI{vector: query}
	uc := queryVectorHarness(ai)

	got := uc.buildRecommendationQueryVector(context.Background(), "occasion: work", style, 0.5)

	if len(got) != 768 {
		t.Fatalf("expected 768 dimensions, got %d", len(got))
	}
	expected := 1 / math.Sqrt2
	if math.Abs(float64(got[0])-expected) > 1e-5 || math.Abs(float64(got[1])-expected) > 1e-5 {
		t.Fatalf("expected a balanced blend of the two vectors, got %.6f/%.6f", got[0], got[1])
	}
	if norm := vectorNorm(got); math.Abs(norm-1) > 1e-5 {
		t.Fatalf("expected a unit vector, got norm %.6f", norm)
	}
	if ai.calls != 1 || len(ai.requests) != 1 || ai.requests[0] != "occasion: work" {
		t.Fatalf("expected one embedding request for the semantic query, got %+v", ai.requests)
	}
}

func TestBuildRecommendationQueryVectorUsesStyleWhenQueryIsEmpty(t *testing.T) {
	style := oneHotVector(2)
	ai := &fakeQueryEmbeddingAI{vector: oneHotVector(5)}
	uc := queryVectorHarness(ai)

	got := uc.buildRecommendationQueryVector(context.Background(), "", style, 0.3)

	assertNormalizedVector(t, got, style)
	if ai.calls != 0 {
		t.Fatalf("expected no embedding call without a semantic query, got %d", ai.calls)
	}
}

func TestBuildRecommendationQueryVectorKeepsQueryWhenStyleIsEmpty(t *testing.T) {
	query := oneHotVector(4)
	ai := &fakeQueryEmbeddingAI{vector: query}
	uc := queryVectorHarness(ai)

	got := uc.buildRecommendationQueryVector(context.Background(), "occasion: work", nil, 0.3)

	assertNormalizedVector(t, got, query)
	if len(got) != 768 {
		t.Fatalf("expected 768 dimensions, got %d", len(got))
	}
}

func TestBuildRecommendationQueryVectorFallsBackToStyleWhenQueryEmbeddingFails(t *testing.T) {
	style := oneHotVector(6)
	ai := &fakeQueryEmbeddingAI{err: errors.New("provider unavailable")}
	uc := queryVectorHarness(ai)

	got := uc.buildRecommendationQueryVector(context.Background(), "occasion: work", style, 0.3)

	assertNormalizedVector(t, got, style)
	if ai.calls != 1 {
		t.Fatalf("expected one failed provider call, got %d", ai.calls)
	}
}

func TestBuildRecommendationQueryVectorReturnsNilWithoutInputs(t *testing.T) {
	uc := queryVectorHarness(&fakeQueryEmbeddingAI{})

	if got := uc.buildRecommendationQueryVector(context.Background(), "", nil, 0.3); got != nil {
		t.Fatalf("expected nil without query and style, got %v", got)
	}
}

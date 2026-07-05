package persistence

import (
	"math"
	"strings"
	"testing"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"

	"github.com/google/uuid"
)

func TestMergeHybridCandidateRecordsByRRF(t *testing.T) {
	sharedID := uuid.New()
	vectorOnlyID := uuid.New()
	lexicalOnlyID := uuid.New()

	merged := mergeHybridCandidateRecordsByRRF(
		[]hybridCandidateRecord{
			{
				ID:          sharedID,
				VectorScore: 0.9,
			},
			{
				ID:          vectorOnlyID,
				VectorScore: 0.8,
			},
		},
		[]hybridCandidateRecord{
			{
				ID:           lexicalOnlyID,
				LexicalScore: 1.2,
			},
			{
				ID:           sharedID,
				LexicalScore: 0.7,
			},
		},
		30,
		10,
	)

	if len(merged) != 3 {
		t.Fatalf("expected three merged records, got %d", len(merged))
	}

	shared := findHybridRecord(t, merged, sharedID)
	expectedSharedScore := 1.0/31.0 + 1.0/32.0
	if math.Abs(shared.RetrievalScore-expectedSharedScore) > 0.000001 {
		t.Fatalf("unexpected shared RRF score: got %.8f want %.8f", shared.RetrievalScore, expectedSharedScore)
	}
	if shared.RetrievalSource != repositories.HybridCandidateSourceHybrid {
		t.Fatalf("expected shared source hybrid, got %q", shared.RetrievalSource)
	}
	if shared.VectorScore != 0.9 || shared.LexicalScore != 0.7 {
		t.Fatalf("expected raw scores preserved, got vector %.2f lexical %.2f", shared.VectorScore, shared.LexicalScore)
	}

	vectorOnly := findHybridRecord(t, merged, vectorOnlyID)
	if vectorOnly.RetrievalSource != repositories.HybridCandidateSourceVector {
		t.Fatalf("expected vector-only source vector, got %q", vectorOnly.RetrievalSource)
	}

	lexicalOnly := findHybridRecord(t, merged, lexicalOnlyID)
	if lexicalOnly.RetrievalSource != repositories.HybridCandidateSourceLexical {
		t.Fatalf("expected lexical-only source lexical, got %q", lexicalOnly.RetrievalSource)
	}
}

func TestMergeHybridCandidateRecordsByRRFTieBreaksByID(t *testing.T) {
	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	merged := mergeHybridCandidateRecordsByRRF(
		[]hybridCandidateRecord{
			{ID: secondID},
			{ID: firstID},
		},
		nil,
		30,
		10,
	)

	if len(merged) != 2 {
		t.Fatalf("expected two merged records, got %d", len(merged))
	}
	if merged[0].ID != secondID || merged[1].ID != firstID {
		t.Fatalf("expected vector rank to beat ID tie-break when scores differ, got %s then %s", merged[0].ID, merged[1].ID)
	}

	merged = mergeHybridCandidateRecordsByRRF(
		[]hybridCandidateRecord{
			{ID: secondID},
		},
		[]hybridCandidateRecord{
			{ID: firstID},
		},
		30,
		10,
	)

	if merged[0].ID != firstID || merged[1].ID != secondID {
		t.Fatalf("expected equal RRF scores to tie-break by ID, got %s then %s", merged[0].ID, merged[1].ID)
	}
}

func TestBuildRecommendationSeasonalityConditionAllowsAllSeasonAndMissingMetadata(t *testing.T) {
	sql, args := buildRecommendationSeasonalityCondition([]string{"winter"})

	if sql == "" {
		t.Fatal("expected seasonality condition SQL")
	}
	for _, fragment := range []string{
		"recommendation_fashion_items.seasonality IS NULL",
		"btrim(recommendation_fashion_items.seasonality) = ''",
		"immutable_unaccent(lower(coalesce(recommendation_fashion_items.seasonality, '')))",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL fragment %q in %s", fragment, sql)
		}
	}
	for _, expected := range []any{"%winter%", "%bon mua%", "%quanh nam%", "%all-season%"} {
		if !containsAnyArg(args, expected) {
			t.Fatalf("expected arg %v in %v", expected, args)
		}
	}
}

func containsAnyArg(values []any, target any) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func findHybridRecord(t *testing.T, records []hybridCandidateRecord, id uuid.UUID) hybridCandidateRecord {
	t.Helper()
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	t.Fatalf("record %s not found", id)
	return hybridCandidateRecord{}
}

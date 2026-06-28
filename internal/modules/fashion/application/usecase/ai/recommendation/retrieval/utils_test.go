package retrieval

import (
	"reflect"
	"strings"
	"testing"

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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

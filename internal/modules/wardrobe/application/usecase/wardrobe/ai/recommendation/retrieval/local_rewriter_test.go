package retrieval

import (
	"context"
	"reflect"
	"testing"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

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

package retrieval

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

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

func TestRecommendationCategoryTaxonomyIncludesDamAndChanVay(t *testing.T) {
	slugs := RecommendationAllowedCategorySlugs()
	if !containsString(slugs, "dam") {
		t.Fatalf("expected dam in allowlist, got %v", slugs)
	}
	if !containsString(slugs, "chan-vay") {
		t.Fatalf("expected chan-vay in allowlist, got %v", slugs)
	}
	if containsString(slugs, "vay") {
		t.Fatalf("did not expect legacy vay in allowlist, got %v", slugs)
	}

	damTerms := ExpandTaxonomyTermValues(taxonomyGroupCategory, []string{"dam"})
	if !containsString(damTerms, "dress") {
		t.Fatalf("expected dress synonym for dam, got %v", damTerms)
	}

	skirtTerms := ExpandTaxonomyTermValues(taxonomyGroupCategory, []string{"chan-vay"})
	if !containsString(skirtTerms, "skirt") {
		t.Fatalf("expected skirt synonym for chan-vay, got %v", skirtTerms)
	}
}

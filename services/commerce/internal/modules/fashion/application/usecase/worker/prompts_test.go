package worker

import (
	"strings"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
)

func TestVisionPromptExplainsDamAndChanVayDistinction(t *testing.T) {
	prompt := getVisionSystemPrompt([]dto.AICategoryRef{
		{Name: "Đầm", Slug: "dam"},
		{Name: "Chân váy", Slug: "chan-vay"},
	})

	for _, expected := range []string{
		`Use "dam" only for a one-piece garment`,
		`Use "chan-vay" only for a separate lower-body garment`,
		`prioritize "chan-vay"`,
		`prioritize "dam"`,
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q, got: %s", expected, prompt)
		}
	}
}

func TestVisionSystemPromptFailsFast(t *testing.T) {
	prompt := getVisionSystemPrompt([]dto.AICategoryRef{{Name: "Áo", Slug: "ao"}})

	for _, expected := range []string{
		`"error": "No fashion item detected"`,
		`"is_single_item": false`,
		`"category_confident": false`,
		`"review_reason": "uncertain_category"`,
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected fail-fast prompt to contain %q", expected)
		}
	}
}

func TestVisionMetadataPromptIsMetadataOnly(t *testing.T) {
	prompt := getVisionMetadataPrompt("Đầm")

	if !strings.Contains(prompt, "Đầm") {
		t.Fatalf("expected prompt to embed the fixed category name")
	}
	for _, forbidden := range []string{"category_slug", "category_confident", "is_single_item"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("metadata-only prompt must not contain %q", forbidden)
		}
	}
	if !strings.Contains(prompt, `"error": "No fashion item detected"`) {
		t.Fatalf("metadata-only prompt must still fail fast on no item")
	}
}

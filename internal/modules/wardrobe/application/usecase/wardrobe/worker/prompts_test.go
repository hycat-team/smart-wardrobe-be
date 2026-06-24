package worker

import (
	"strings"
	"testing"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
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

package validation

import (
	"testing"
)

func TestIsBrandSlugFormatValid(t *testing.T) {
	valid := []string{"mori-studio", "a1", "brand-2026", "x9-y"}
	for _, slug := range valid {
		if !IsBrandSlugFormatValid(slug) {
			t.Errorf("expected %q to be valid", slug)
		}
	}

	invalid := []string{
		"",
		"a",
		"Mori-Studio",
		"-mori",
		"mori-",
		"mori--studio",
		"mori_studio",
		"mori studio",
		"mori/studio",
		"123e4567-e89b-12d3-a456-426614174000",
	}
	for _, slug := range invalid {
		if IsBrandSlugFormatValid(slug) {
			t.Errorf("expected %q to be invalid", slug)
		}
	}
}

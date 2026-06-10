package ai

import (
	"testing"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func float64Ptr(val float64) *float64 {
	return &val
}

func stringPtr(val string) *string {
	return &val
}

func TestIsNeutral(t *testing.T) {
	tests := []struct {
		name       string
		lightness  *float64
		saturation *float64
		expected   bool
	}{
		{"nil values", nil, nil, true},
		{"black low lightness", float64Ptr(5), float64Ptr(50), true},
		{"white high lightness", float64Ptr(95), float64Ptr(50), true},
		{"grey low saturation", float64Ptr(50), float64Ptr(5), true},
		{"colored chroma", float64Ptr(50), float64Ptr(50), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &entities.WardrobeItem{
				ColorLightness:  tt.lightness,
				ColorSaturation: tt.saturation,
			}
			if got := isNeutral(item); got != tt.expected {
				t.Errorf("isNeutral() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestColorsMatch(t *testing.T) {
	tests := []struct {
		name     string
		item1    *entities.WardrobeItem
		item2    *entities.WardrobeItem
		expected bool
	}{
		{
			name: "neutral item 1 matches anything",
			item1: &entities.WardrobeItem{
				ColorLightness:  float64Ptr(5), // Black
				ColorSaturation: float64Ptr(50),
			},
			item2: &entities.WardrobeItem{
				ColorHue:        float64Ptr(120),
				ColorLightness:  float64Ptr(50),
				ColorSaturation: float64Ptr(50),
			},
			expected: true,
		},
		{
			name: "analogous match: hue diff < 30, lightness diff >= 15",
			item1: &entities.WardrobeItem{
				ColorHue:        float64Ptr(10),
				ColorLightness:  float64Ptr(40),
				ColorSaturation: float64Ptr(50),
			},
			item2: &entities.WardrobeItem{
				ColorHue:        float64Ptr(25),
				ColorLightness:  float64Ptr(60),
				ColorSaturation: float64Ptr(50),
			},
			expected: true,
		},
		{
			name: "analogous mismatch: hue diff < 30, lightness diff < 15",
			item1: &entities.WardrobeItem{
				ColorHue:        float64Ptr(10),
				ColorLightness:  float64Ptr(40),
				ColorSaturation: float64Ptr(50),
			},
			item2: &entities.WardrobeItem{
				ColorHue:        float64Ptr(25),
				ColorLightness:  float64Ptr(50),
				ColorSaturation: float64Ptr(50),
			},
			expected: false,
		},
		{
			name: "complementary match: hue diff ~ 180",
			item1: &entities.WardrobeItem{
				ColorHue:        float64Ptr(10),
				ColorLightness:  float64Ptr(50),
				ColorSaturation: float64Ptr(50),
			},
			item2: &entities.WardrobeItem{
				ColorHue:        float64Ptr(185),
				ColorLightness:  float64Ptr(50),
				ColorSaturation: float64Ptr(50),
			},
			expected: true,
		},
		{
			name: "no match",
			item1: &entities.WardrobeItem{
				ColorHue:        float64Ptr(10),
				ColorLightness:  float64Ptr(50),
				ColorSaturation: float64Ptr(50),
			},
			item2: &entities.WardrobeItem{
				ColorHue:        float64Ptr(100),
				ColorLightness:  float64Ptr(50),
				ColorSaturation: float64Ptr(50),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colorsMatch(tt.item1, tt.item2); got != tt.expected {
				t.Errorf("colorsMatch() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRunLocalHSLMatching(t *testing.T) {
	catTop := &entities.Category{Name: "Áo", Slug: "ao"}
	catTop.ID = uuid.New()

	catBottom := &entities.Category{Name: "Quần", Slug: "quan"}
	catBottom.ID = uuid.New()

	catShoes := &entities.Category{Name: "Giày", Slug: "giay"}
	catShoes.ID = uuid.New()

	catOuter := &entities.Category{Name: "Áo khoác", Slug: "ao-khoac"}
	catOuter.ID = uuid.New()

	item1 := &entities.WardrobeItem{
		Category:        catTop,
		ColorHue:        float64Ptr(120),
		ColorLightness:  float64Ptr(50),
		ColorSaturation: float64Ptr(50),
	}
	item1.ID = uuid.New()

	item2 := &entities.WardrobeItem{
		Category:        catBottom,
		ColorHue:        float64Ptr(130),
		ColorLightness:  float64Ptr(70), // Analogous to Top
		ColorSaturation: float64Ptr(50),
	}
	item2.ID = uuid.New()

	item3 := &entities.WardrobeItem{
		Category:        catShoes,
		ColorHue:        float64Ptr(125),
		ColorLightness:  float64Ptr(50),
		ColorSaturation: float64Ptr(50),
	}
	item3.ID = uuid.New()

	item4 := &entities.WardrobeItem{
		Category:        catOuter,
		ColorHue:        float64Ptr(300), // Complementary to top/bottom
		ColorLightness:  float64Ptr(50),
		ColorSaturation: float64Ptr(50),
	}
	item4.ID = uuid.New()

	candidates := []*entities.WardrobeItem{item1, item2, item3, item4}

	req := dto.RecommendOutfitReq{
		Weather: stringPtr("Trời lạnh"),
	}

	res := runLocalHSLMatching(candidates, req)

	if res == nil {
		t.Fatal("expected non-nil response")
	}
	if !res.IsFallback {
		t.Error("expected IsFallback to be true")
	}
	if len(res.Items) < 3 {
		t.Errorf("expected at least 3 items (ao, quan, giay, outerwear), got %d", len(res.Items))
	}

	foundOuter := false
	for _, item := range res.Items {
		if item.Role == "ao-khoac" {
			foundOuter = true
			break
		}
	}
	if !foundOuter {
		t.Error("expected outerwear in recommendation groups since weather was cold")
	}
}

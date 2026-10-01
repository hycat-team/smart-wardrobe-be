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
		`"is_single_item": false`,
		`"category_confident": false`,
		`"review_reason": "uncertain_category"`,
		`"review_reason": "no_fashion_item_detected"`,
		`"review_reason": "multiple_items_detected"`,
		`"review_reason": "full_body_outfit_detected"`,
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected fail-fast prompt to contain %q", expected)
		}
	}
}

// Prompt phải đưa "nhiều món / toàn thân" lên TRƯỚC "không phải trang phục":
// ảnh vật lạ cạnh một món trang phục vẫn phải được phân loại như một món.
func TestVisionSystemPromptChecksMultipleItemsBeforeNoGarment(t *testing.T) {
	prompt := getVisionSystemPrompt([]dto.AICategoryRef{{Name: "Áo", Slug: "ao"}})

	multiIdx := strings.Index(prompt, "Rule 1 — The image shows two or more garments")
	noItemIdx := strings.Index(prompt, "Rule 2 — The image contains NO garment at all")

	if multiIdx < 0 || noItemIdx < 0 {
		t.Fatalf("expected both decision rules in prompt, got: %s", prompt)
	}
	if multiIdx > noItemIdx {
		t.Fatalf("multi-item rule must be checked before the no-garment rule")
	}
}

// is_single_item đo số món TRANG PHỤC, không phải câu hỏi "đây có phải trang phục
// không" — nếu không nói rõ, model sẽ set false cho ảnh không có trang phục và
// đẩy về multiple_items_detected.
func TestVisionSystemPromptSeparatesItemCountFromGarmentPresence(t *testing.T) {
	prompt := getVisionSystemPrompt([]dto.AICategoryRef{{Name: "Áo", Slug: "ao"}})

	if !strings.Contains(prompt, `It is NOT a "is this a garment" test`) {
		t.Fatalf("prompt must state is_single_item is not a garment-presence test")
	}
	if !strings.Contains(prompt, "Only GARMENTS count towards the item count") {
		t.Fatalf("prompt must state only garments count towards the item count")
	}
}

// Kênh free-text "error" không còn nằm trong schema nên từng mâu thuẫn với model.
func TestVisionPromptsDropLegacyErrorChannel(t *testing.T) {
	prompts := map[string]string{
		"system":   getVisionSystemPrompt([]dto.AICategoryRef{{Name: "Áo", Slug: "ao"}}),
		"metadata": getVisionMetadataPrompt("Đầm"),
	}

	for name, prompt := range prompts {
		if strings.Contains(prompt, "No fashion item detected") {
			t.Fatalf("%s prompt must not use the legacy free-text error channel", name)
		}
	}
}

func TestVisionMetadataPromptIsMetadataOnly(t *testing.T) {
	prompt := getVisionMetadataPrompt("Đầm")

	if !strings.Contains(prompt, "Đầm") {
		t.Fatalf("expected prompt to embed the fixed category name")
	}
	// Danh mục vẫn bị cấm vì fixed-category không được phân loại lại, nhưng
	// is_single_item phải được phép để chặn ảnh nhiều món / không có trang phục.
	for _, forbidden := range []string{"category_slug", "category_confident"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("metadata-only prompt must not contain %q", forbidden)
		}
	}
	if !strings.Contains(prompt, "The category is already decided: do NOT classify") {
		t.Fatalf("metadata-only prompt must forbid re-classification")
	}
}

// Fixed-category retry phải kiểm tra ảnh, nếu không một ảnh toàn thân đã lọt qua
// lần phân tích đầu sẽ được AI bịa metadata cho danh mục user đã chọn.
func TestVisionMetadataPromptStillRejectsUnusableImages(t *testing.T) {
	prompt := getVisionMetadataPrompt("Đầm")

	for _, expected := range []string{
		`"is_single_item": false`,
		`"review_reason": "multiple_items_detected"`,
		`"review_reason": "full_body_outfit_detected"`,
		`"review_reason": "no_fashion_item_detected"`,
		"Only GARMENTS count towards the item count",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected metadata prompt to contain %q", expected)
		}
	}
}

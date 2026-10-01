package worker

import (
	"encoding/json"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/analysisreason"
)

// TestInvalidImageReason bảo vệ quyết định ở tầng 1 của cổng phân tích: ảnh nào
// bị đánh dấu không dùng được và vì lý do gì.
func TestInvalidImageReason(t *testing.T) {
	tests := []struct {
		name string
		meta *dto.FashionMetadataResult
		want string
	}{
		{
			name: "no garment beats single-item flag",
			meta: &dto.FashionMetadataResult{
				IsSingleItem: true,
				ReviewReason: analysisreason.ErrorNoFashionItem,
			},
			want: analysisreason.ErrorNoFashionItem,
		},
		{
			name: "single garment is valid",
			meta: &dto.FashionMetadataResult{
				IsSingleItem: true,
			},
			want: "",
		},
		{
			name: "multiple garments keeps its reason",
			meta: &dto.FashionMetadataResult{
				ReviewReason: analysisreason.ErrorMultipleItems,
			},
			want: analysisreason.ErrorMultipleItems,
		},
		{
			name: "full body keeps its reason",
			meta: &dto.FashionMetadataResult{
				ReviewReason: analysisreason.ErrorFullBodyOutfit,
			},
			want: analysisreason.ErrorFullBodyOutfit,
		},
		{
			name: "empty reason falls back to multiple items",
			meta: &dto.FashionMetadataResult{},
			want: analysisreason.ErrorMultipleItems,
		},
		{
			name: "contradictory reason falls back to multiple items",
			meta: &dto.FashionMetadataResult{
				ReviewReason: analysisreason.ReviewUncertainCategory,
			},
			want: analysisreason.ErrorMultipleItems,
		},
		{
			name: "review reason on a valid single garment is ignored",
			meta: &dto.FashionMetadataResult{
				IsSingleItem: true,
				ReviewReason: "some free text from the model",
			},
			want: "",
		},
		{
			name: "nil result is treated as unusable",
			meta: nil,
			want: analysisreason.ErrorMultipleItems,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := invalidImageReason(tt.meta); got != tt.want {
				t.Fatalf("invalidImageReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Mã "không có trang phục" phải chặn retry được, nếu không người dùng thử lại
// một ảnh vĩnh viễn không dùng được mãi (FR-033).
func TestNoFashionItemBlocksRetry(t *testing.T) {
	if !analysisreason.IsInvalidImageReason(analysisreason.ErrorNoFashionItem) {
		t.Fatalf("%s must be an invalid-image reason so retry stays blocked", analysisreason.ErrorNoFashionItem)
	}
	if analysisreason.IsInvalidImageReason(analysisreason.ErrorAnalysisTemporary) {
		t.Fatalf("%s must stay retryable", analysisreason.ErrorAnalysisTemporary)
	}
}

// Payload cũ dùng key "error" thay vì review_reason. Sau khi chuẩn hoá, nó phải ra
// mã chặn retry chứ không phải lỗi tạm thời cho phép retry vô hạn.
func TestLegacyErrorPayloadMapsToNoFashionItem(t *testing.T) {
	aiMeta, err := parseAnalyzedJSON(`{"error":"No fashion item detected"}`)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	reason := invalidImageReason(aiMeta)
	if reason != analysisreason.ErrorNoFashionItem {
		t.Fatalf("legacy error payload mapped to %q, want %q", reason, analysisreason.ErrorNoFashionItem)
	}
	if !analysisreason.IsInvalidImageReason(reason) {
		t.Fatalf("legacy error payload must block retry")
	}
}

// Hồi quy: dto.FashionMetadataResult tự implement json.Unmarshaler, nên embed nó
// vào anonymous struct kèm field "error" sẽ khiến field đó không bao giờ được gán.
// Đó chính là lý do ảnh không phải trang phục từng bị báo nhầm là nhiều món.
func TestLegacyErrorFieldIsActuallyRead(t *testing.T) {
	raw := `{"error":"No fashion item detected","is_single_item":false}`

	var embedded struct {
		dto.FashionMetadataResult
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &embedded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if embedded.Error != "" {
		t.Fatalf("precondition failed: embedded form is expected to swallow the field")
	}

	aiMeta, err := parseAnalyzedJSON(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if aiMeta.ReviewReason != analysisreason.ErrorNoFashionItem {
		t.Fatalf("parseAnalyzedJSON must read the legacy error field, got reason %q", aiMeta.ReviewReason)
	}
}

// Payload hợp lệ, có is_single_item=true, không bị đụng tới bởi safety net.
func TestValidResponseIsLeftIntact(t *testing.T) {
	aiMeta, err := parseAnalyzedJSON(`{"is_single_item":true,"category_confident":true,"color":"Đen mun","description":"Áo thun đen cổ tròn."}`)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if got := invalidImageReason(aiMeta); got != "" {
		t.Fatalf("valid single garment must stay valid, got %q", got)
	}
	if aiMeta.Color != "Đen mun" {
		t.Fatalf("valid response metadata must be preserved, got %q", aiMeta.Color)
	}
}

// Hành vi có chủ đích (giữ nguyên từ trước): thiếu is_single_item bị coi là
// không hợp lệ theo safety net FR-027, vì prompt bắt buộc phải có key này.
func TestMissingSingleItemFlagIsRejectedConservatively(t *testing.T) {
	aiMeta, err := parseAnalyzedJSON(`{"color":"Đen mun","description":"Áo thun đen cổ tròn."}`)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if aiMeta.IsSingleItem {
		t.Fatalf("precondition failed: missing is_single_item must default to false")
	}
	if got := invalidImageReason(aiMeta); got != analysisreason.ErrorMultipleItems {
		t.Fatalf("missing is_single_item must fall back to multiple_items_detected, got %q", got)
	}
}

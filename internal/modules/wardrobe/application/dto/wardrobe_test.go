package dto

import (
	"encoding/json"
	"testing"
)

func TestFashionMetadataResult_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectSuccess bool
		expectedBool  bool
	}{
		{
			name:          "is_single_item is true boolean",
			jsonInput:     `{"is_single_item": true, "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  true,
		},
		{
			name:          "is_single_item is false boolean",
			jsonInput:     `{"is_single_item": false, "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  false,
		},
		{
			name:          "is_single_item is true string uppercase",
			jsonInput:     `{"is_single_item": "TRUE", "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  true,
		},
		{
			name:          "is_single_item is false string lowercase",
			jsonInput:     `{"is_single_item": "false", "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  false,
		},
		{
			name:          "is_single_item is 1 string",
			jsonInput:     `{"is_single_item": "1", "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  true,
		},
		{
			name:          "is_single_item is empty string",
			jsonInput:     `{"is_single_item": "", "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  false,
		},
		{
			name:          "is_single_item is null",
			jsonInput:     `{"is_single_item": null, "category_slug": "ao"}`,
			expectSuccess: true,
			expectedBool:  false,
		},
		{
			name:          "is_single_item is invalid string",
			jsonInput:     `{"is_single_item": "maybe", "category_slug": "ao"}`,
			expectSuccess: false,
		},
		{
			name:          "is_single_item is numeric",
			jsonInput:     `{"is_single_item": 123, "category_slug": "ao"}`,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result FashionMetadataResult
			err := json.Unmarshal([]byte(tt.jsonInput), &result)
			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if result.IsSingleItem != tt.expectedBool {
					t.Errorf("expected IsSingleItem to be %v, got %v", tt.expectedBool, result.IsSingleItem)
				}
				if result.CategorySlug != "ao" {
					t.Errorf("expected CategorySlug to be 'ao', got %q", result.CategorySlug)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			}
		})
	}
}

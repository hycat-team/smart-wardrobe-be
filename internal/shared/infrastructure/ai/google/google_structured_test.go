package google

import (
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"testing"
)

func TestGoogleGenerationConfigIncludesStructuredOutput(t *testing.T) {
	schema := map[string]any{"type": "OBJECT"}
	cfg := GoogleGenerationConfig(app_ai.TextGenerationOptions{MaxOutputTokens: 400, Temperature: 0.1, ResponseMIMEType: "application/json", ResponseSchema: schema})
	if cfg["maxOutputTokens"] != 400 {
		t.Fatal("missing output cap")
	}
	if cfg["responseMimeType"] != "application/json" {
		t.Fatal("missing JSON MIME type")
	}
	if cfg["responseSchema"] == nil {
		t.Fatal("missing response schema")
	}
}

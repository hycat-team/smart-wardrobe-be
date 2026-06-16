// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"encoding/json"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

// ParseOutfitRecommendationJSON parses the model response, strips markdown code blocks, and handles surrounding prose.
func ParseOutfitRecommendationJSON(responseText string) (types.LlmOutfitResponse, string, error) {
	cleaned := stringutils.CleanJSONMarkdown(responseText)

	var result types.LlmOutfitResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		return result, cleaned, ValidateOutfitRecommendationPayload(result)
	}

	extracted := ExtractFirstJSONObject(cleaned)
	if extracted == "" {
		return types.LlmOutfitResponse{}, cleaned, fmt.Errorf("could not extract JSON object from response")
	}

	err := json.Unmarshal([]byte(extracted), &result)
	if err != nil {
		return result, extracted, err
	}

	return result, extracted, ValidateOutfitRecommendationPayload(result)
}

// ValidateOutfitRecommendationPayload rejects placeholder-filled responses before mapping candidate items.
func ValidateOutfitRecommendationPayload(payload types.LlmOutfitResponse) error {
	if isPlaceholderValue(payload.Title) || isPlaceholderValue(payload.Explanation) {
		return fmt.Errorf("placeholder values detected in recommendation payload")
	}

	for _, item := range payload.Items {
		if isPlaceholderValue(item.Role) || isPlaceholderValue(item.PrimaryID) {
			return fmt.Errorf("placeholder values detected in recommendation items")
		}
		for _, altID := range item.AlternativeIDs {
			if isPlaceholderValue(altID) {
				return fmt.Errorf("placeholder values detected in recommendation alternatives")
			}
		}
	}

	return nil
}

func isPlaceholderValue(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "string", "uuid", "role", "primary_id", "alternative_ids", "title", "explanation":
		return normalized != ""
	}

	return false
}

// ExtractFirstJSONObject returns the first balanced top-level JSON object in the text.
func ExtractFirstJSONObject(value string) string {
	start := strings.IndexByte(value, '{')
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(value); i++ {
		ch := value[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(value[start : i+1])
			}
		}
	}

	return ""
}

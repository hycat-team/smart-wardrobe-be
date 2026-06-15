package recommendation

import (
	"fmt"
	"encoding/json"
	"strings"

	"smart-wardrobe-be/pkg/utils/stringutils"
)

// parseOutfitRecommendationJSON parses the model response and tolerates non-JSON wrappers
// by extracting the first complete JSON object when the provider returns extra prose.
func parseOutfitRecommendationJSON(responseText string) (llmOutfitResponse, string, error) {
	cleaned := stringutils.CleanJSONMarkdown(responseText)

	var result llmOutfitResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		return result, cleaned, validateOutfitRecommendationPayload(result)
	} else {
		extracted := extractFirstJSONObject(cleaned)
		if extracted == "" {
			return llmOutfitResponse{}, cleaned, err
		}

		err = json.Unmarshal([]byte(extracted), &result)
		if err != nil {
			return result, extracted, err
		}

		return result, extracted, validateOutfitRecommendationPayload(result)
	}
}

// validateOutfitRecommendationPayload rejects placeholder-filled responses before mapping.
func validateOutfitRecommendationPayload(payload llmOutfitResponse) error {
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

// extractFirstJSONObject returns the first balanced top-level JSON object in the text.
func extractFirstJSONObject(value string) string {
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

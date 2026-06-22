package chat

import (
	"encoding/json"
	"errors"
	"strings"

	"golang.org/x/text/unicode/norm"
)

const summarySystemPrompt = "Summarize fashion conversation facts. Return JSON only. Never include reasoning or infer facts not present in the conversation."

type structuredSummary struct {
	Preferences      []string `json:"preferences"`
	Dislikes         []string `json:"dislikes"`
	Constraints      []string `json:"constraints"`
	ImportantContext []string `json:"important_context"`
}

func summaryResponseSchema() any {
	return map[string]any{"type": "OBJECT", "required": []string{"preferences", "dislikes", "constraints", "important_context"}, "properties": map[string]any{"preferences": arrayStringSchema(), "dislikes": arrayStringSchema(), "constraints": arrayStringSchema(), "important_context": arrayStringSchema()}}
}
func arrayStringSchema() any {
	return map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}}
}

func parseAndValidateSummary(raw string) (string, error) {
	start := strings.Index(raw, "{")
	if start < 0 {
		return "", errors.New("summary JSON object missing")
	}
	depth := 0
	end := -1
	quoted := false
	escaped := false
	for i, r := range raw[start:] {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quoted {
			escaped = true
			continue
		}
		if r == '"' {
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		if r == '{' {
			depth++
		}
		if r == '}' {
			depth--
			if depth == 0 {
				end = start + i + 1
				break
			}
		}
	}
	if end < 0 {
		return "", errors.New("summary JSON incomplete")
	}
	var value structuredSummary
	if err := json.Unmarshal([]byte(raw[start:end]), &value); err != nil {
		return "", err
	}
	value.Preferences = cleanSummaryItems(value.Preferences)
	value.Dislikes = cleanSummaryItems(value.Dislikes)
	value.Constraints = cleanSummaryItems(value.Constraints)
	value.ImportantContext = cleanSummaryItems(value.ImportantContext)
	bytes, err := json.Marshal(value)
	return string(bytes), err
}
func cleanSummaryItems(items []string) []string {
	if len(items) > 8 {
		items = items[:8]
	}
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, v := range items {
		v = strings.TrimSpace(norm.NFC.String(v))
		v = truncateRunes(v, 150)
		key := strings.ToLower(v)
		if v != "" && !seen[key] {
			seen[key] = true
			out = append(out, v)
		}
	}
	return out
}

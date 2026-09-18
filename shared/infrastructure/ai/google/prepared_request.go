package google

import (
	"encoding/json"
	"strings"
)

// PreparedGeminiRequest represents a pre-built request payload for Gemini.
// Shared by token estimation, preflight countTokens, and content generation.
type PreparedGeminiRequest struct {
	Model string
	Body  GeminiGenerateContentBody
}

type GeminiGenerateContentBody struct {
	Contents          []GeminiContent `json:"contents"`
	SystemInstruction *GeminiContent  `json:"systemInstruction,omitempty"`
	GenerationConfig  map[string]any  `json:"generationConfig,omitempty"`
	Tools             any             `json:"tools,omitempty"`
	ToolConfig        any             `json:"toolConfig,omitempty"`
	CachedContent     string          `json:"cachedContent,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

// AllInputText returns concatenated text of all input fields in the body for local estimation.
func (r PreparedGeminiRequest) AllInputText() string {
	var sb strings.Builder

	// 1. System instructions
	if r.Body.SystemInstruction != nil {
		for _, part := range r.Body.SystemInstruction.Parts {
			if part.Text != "" {
				sb.WriteString(part.Text)
				sb.WriteString(" ")
			}
		}
	}

	// 2. Chat history and current user turn
	for _, content := range r.Body.Contents {
		for _, part := range content.Parts {
			if part.Text != "" {
				sb.WriteString(part.Text)
				sb.WriteString(" ")
			}
		}
	}

	// 3. Serialize Tools declarations if present
	if r.Body.Tools != nil {
		if toolsJSON, err := json.Marshal(r.Body.Tools); err == nil {
			sb.Write(toolsJSON)
			sb.WriteString(" ")
		}
	}

	// 4. Serialize ToolConfig if present
	if r.Body.ToolConfig != nil {
		if configJSON, err := json.Marshal(r.Body.ToolConfig); err == nil {
			sb.Write(configJSON)
			sb.WriteString(" ")
		}
	}

	// 5. Serialize ResponseSchema from GenerationConfig if present
	if r.Body.GenerationConfig != nil {
		if schema, ok := r.Body.GenerationConfig["responseSchema"]; ok && schema != nil {
			if schemaJSON, err := json.Marshal(schema); err == nil {
				sb.Write(schemaJSON)
				sb.WriteString(" ")
			}
		}
	}

	return strings.TrimSpace(sb.String())
}

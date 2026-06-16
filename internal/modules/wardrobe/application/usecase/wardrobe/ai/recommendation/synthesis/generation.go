// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/application/ai"
)

// GenerateOutfitRecommendation coordinates the prompting, LLM invocation, parsing, and candidate mapping.
func GenerateOutfitRecommendation(
	ctx context.Context,
	aiService ai.IAIService,
	candidates []types.CandidateForPrompt,
	input dto.RecommendOutfitReq,
) (*dto.RecommendedOutfitRes, error) {
	userPrompt := BuildRecommendationPrompt(candidates, input)
	responseText, err := aiService.GenerateRecommendationText(
		ctx,
		"You are a senior fashion stylist and wardrobe editor. Recommend the most suitable outfit from the available items, stay faithful to the actual item attributes, and respond with exactly one valid minified JSON object. The fields title and explanation must be written in natural Vietnamese with proper diacritics.",
		userPrompt,
	)
	if err != nil {
		return nil, NewFallbackTraceError(
			"provider_error",
			err,
			userPrompt,
			"",
		)
	}

	if responseText == "" {
		return nil, NewFallbackTraceError(
			"empty_response",
			fmt.Errorf("empty response from LLM"),
			userPrompt,
			"",
		)
	}

	llmRes, cleanedResponse, err := ParseOutfitRecommendationJSON(responseText)
	if err != nil {
		return nil, NewFallbackTraceError(
			"invalid_json",
			err,
			userPrompt,
			responseText,
		)
	}

	validGroups := MapLLMResponseToGroups(candidates, llmRes)
	if len(validGroups) == 0 {
		return nil, NewFallbackTraceError(
			"invalid_outfit_structure",
			fmt.Errorf("AI returned an invalid outfit structure"),
			userPrompt,
			cleanedResponse,
		)
	}

	return &dto.RecommendedOutfitRes{
		Title:       llmRes.Title,
		Explanation: llmRes.Explanation,
		Items:       validGroups,
	}, nil
}

type fallbackTraceError struct {
	kind            string
	cause           error
	prompt          string
	responsePreview string
}

func (e *fallbackTraceError) Error() string {
	if e == nil || e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

func (e *fallbackTraceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// NewFallbackTraceError builds an error containing debug payload trace info for LLM fallback handling.
func NewFallbackTraceError(kind string, cause error, prompt, response string) error {
	return &fallbackTraceError{
		kind:            kind,
		cause:           cause,
		prompt:          prompt,
		responsePreview: truncateLogText(response, 600),
	}
}

// ClassifyFallbackTrace inspects trace errors to extract debugging stats and API provider hints.
func ClassifyFallbackTrace(err error) (kind string, providerHint string, promptLen int, responsePreview string) {
	kind = "unknown"
	providerHint = "unknown"

	var traceErr *fallbackTraceError
	if errors.As(err, &traceErr) {
		if traceErr.kind != "" {
			kind = traceErr.kind
		}
		promptLen = len(traceErr.prompt)
		responsePreview = traceErr.responsePreview
	}

	errText := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errText, "openai"):
		providerHint = "openai"
	case strings.Contains(errText, "google"):
		providerHint = "gemini"
	case errors.Is(err, context.Canceled):
		providerHint = "request_context"
	case errors.Is(err, context.DeadlineExceeded):
		providerHint = "timeout"
	}

	return kind, providerHint, promptLen, responsePreview
}

func truncateLogText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "...(truncated)"
}

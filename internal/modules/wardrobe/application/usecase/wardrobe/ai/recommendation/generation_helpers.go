package recommendation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

func (uc *OutfitRecommendationUseCase) generateOutfitRecommendation(
	ctx context.Context,
	candidates []CandidateForPrompt,
	input dto.RecommendOutfitReq,
) (*dto.RecommendedOutfitRes, error) {
	userPrompt := buildRecommendationPrompt(candidates, input)
	responseText, err := uc.aiService.GenerateRecommendationText(
		ctx,
		"You are a senior fashion stylist and wardrobe editor. Recommend the most suitable outfit from the available items, stay faithful to the actual item attributes, and respond with exactly one valid minified JSON object. The fields title and explanation must be written in natural Vietnamese with proper diacritics.",
		userPrompt,
	)
	if err != nil {
		return nil, newFallbackTraceError(
			"provider_error",
			err,
			userPrompt,
			"",
		)
	}

	if responseText == "" {
		return nil, newFallbackTraceError(
			"empty_response",
			fmt.Errorf("empty response from LLM"),
			userPrompt,
			"",
		)
	}

	llmRes, cleanedResponse, err := parseOutfitRecommendationJSON(responseText)
	if err != nil {
		return nil, newFallbackTraceError(
			"invalid_json",
			err,
			userPrompt,
			responseText,
		)
	}

	validGroups := uc.mapLLMResponseToGroups(candidates, llmRes)
	if len(validGroups) == 0 {
		return nil, newFallbackTraceError(
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

func (uc *OutfitRecommendationUseCase) updateQuotaAndConstructResponse(
	ctx context.Context,
	userID uuid.UUID,
	finalRes *dto.RecommendedOutfitRes,
	quotaDTO *contract.UserSubscriptionDTO,
) error {
	if err := uc.userQuotaCtr.UpdateOutfitQuota(ctx, userID, 1); err != nil {
		return err
	}

	updatedQuota, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err == nil {
		finalRes.RemainingQuota = updatedQuota.AiOutfitDailyQuota - updatedQuota.OutfitRecommendCount
	} else {
		finalRes.RemainingQuota = quotaDTO.AiOutfitDailyQuota - (quotaDTO.OutfitRecommendCount + 1)
	}

	if finalRes.RemainingQuota < 0 {
		finalRes.RemainingQuota = 0
	}

	return nil
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

func newFallbackTraceError(kind string, cause error, prompt, response string) error {
	return &fallbackTraceError{
		kind:            kind,
		cause:           cause,
		prompt:          prompt,
		responsePreview: truncateLogText(response, 600),
	}
}

func classifyFallbackTrace(err error) (kind string, providerHint string, promptLen int, responsePreview string) {
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

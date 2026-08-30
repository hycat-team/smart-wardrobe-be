package google

import (
	"errors"
)

var (
	ErrCountTokensUnavailable     = errors.New("provider count tokens service unavailable")
	ErrCountTokensInvalidRequest  = errors.New("provider count tokens invalid request")
	ErrCountTokensAuthFailed      = errors.New("provider count tokens auth failed")
	ErrCountTokensModelNotFound   = errors.New("provider count tokens model not found")
	ErrCountTokensInvalidResponse = errors.New("provider count tokens invalid response")
)

type geminiCountTokensRequest struct {
	GenerateContentRequest geminiCountGenerateContentRequest `json:"generateContentRequest"`
}

type geminiCountGenerateContentRequest struct {
	Model string `json:"model"`
	GeminiGenerateContentBody
}

type geminiCountTokensResponse struct {
	TotalTokens int64 `json:"totalTokens"`
}

type googleUsageMetadata struct {
	PromptTokenCount     int64 `json:"promptTokenCount"`
	CandidatesTokenCount int64 `json:"candidatesTokenCount"`
	ThoughtsTokenCount   int64 `json:"thoughtsTokenCount"`
	TotalTokenCount      int64 `json:"totalTokenCount"`
}

func isRetryableCountTokensError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrCountTokensUnavailable)
}

// CountGoogleTokensByRequest calls Google API to count tokens.

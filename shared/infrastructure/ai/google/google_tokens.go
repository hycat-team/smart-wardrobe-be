package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
)

func CountGoogleTokensByRequest(
	ctx context.Context,
	client *http.Client,
	cfg *config.Config,
	provider config.APIProviderConfig,
	req PreparedGeminiRequest,
) (int64, time.Duration, error) {
	parentCtx := ctx

	timeout := 1500 * time.Millisecond
	if cfg != nil {
		if cfg.AI.TokenEstimation.CountTokensTimeout > 0 {
			timeout = cfg.AI.TokenEstimation.CountTokensTimeout
		}
	}

	countCtx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	model := req.Model
	body := geminiCountTokensRequest{
		GenerateContentRequest: geminiCountGenerateContentRequest{
			Model:                     model,
			GeminiGenerateContentBody: req.Body,
		},
	}

	var count int64
	var totalElapsed time.Duration
	var err error

	for attempt := 0; attempt <= 1; attempt++ {
		var elapsed time.Duration
		count, elapsed, err = doCountTokens(countCtx, client, provider, model, body)
		totalElapsed += elapsed

		if err == nil {
			return count, totalElapsed, nil
		}

		if !isRetryableCountTokensError(err) {
			break
		}

		if parentCtx.Err() != nil {
			return 0, totalElapsed, parentCtx.Err()
		}

		if countCtx.Err() != nil {
			return 0, totalElapsed, ErrCountTokensUnavailable
		}
	}

	return count, totalElapsed, err
}

func doCountTokens(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	model string,
	body geminiCountTokensRequest,
) (int64, time.Duration, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, 0, ErrCountTokensInvalidRequest
	}

	url := fmt.Sprintf("%s/%s:countTokens?key=%s", provider.Endpoint, model, provider.ApiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, elapsed, err
		}
		return 0, elapsed, ErrCountTokensUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return 0, elapsed, ErrCountTokensAuthFailed
		case http.StatusNotFound:
			return 0, elapsed, ErrCountTokensModelNotFound
		case http.StatusBadRequest:
			return 0, elapsed, ErrCountTokensInvalidRequest
		case http.StatusRequestTimeout, http.StatusTooManyRequests:
			return 0, elapsed, ErrCountTokensUnavailable
		default:
			if resp.StatusCode >= 500 {
				return 0, elapsed, ErrCountTokensUnavailable
			}
			return 0, elapsed, ErrCountTokensInvalidResponse
		}
	}

	var result geminiCountTokensResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, elapsed, ErrCountTokensInvalidResponse
	}
	if result.TotalTokens < 0 {
		return 0, elapsed, ErrCountTokensInvalidResponse
	}

	return result.TotalTokens, elapsed, nil
}

func GoogleGenerationConfig(options app_ai.TextGenerationOptions) map[string]any {
	cfg := map[string]any{"maxOutputTokens": options.MaxOutputTokens}
	if options.Temperature > 0 {
		cfg["temperature"] = options.Temperature
	}
	if options.ResponseMIMEType != "" {
		cfg["responseMimeType"] = options.ResponseMIMEType
	}
	if options.ResponseSchema != nil {
		cfg["responseSchema"] = options.ResponseSchema
	}
	return cfg
}

// CallTextStream generates chat text response streaming.

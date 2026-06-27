package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/utils/httputils"
	"smart-wardrobe-be/pkg/utils/sliceutils"
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

// NormalizeGeminiModel ensures model name has "models/" prefix.
func NormalizeGeminiModel(model string) string {
	if strings.HasPrefix(model, "models/") {
		return model
	}
	return "models/" + model
}

func isRetryableCountTokensError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrCountTokensUnavailable)
}

// CountGoogleTokensByRequest calls Google API to count tokens.
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

	model := NormalizeGeminiModel(req.Model)
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
func CallTextStream(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
	costPolicy contract.IAICostPolicyContract,
	options app_ai.TextGenerationOptions,
) (<-chan string, <-chan error) {
	textChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		req := PreparedGeminiRequest{
			Model: provider.Model,
			Body: GeminiGenerateContentBody{
				Contents: []GeminiContent{
					{
						Role: "user",
						Parts: []GeminiPart{
							{Text: userPrompt},
						},
					},
				},
				GenerationConfig: GoogleGenerationConfig(options),
			},
		}
		if strings.TrimSpace(systemPrompt) != "" {
			req.Body.SystemInstruction = &GeminiContent{
				Parts: []GeminiPart{
					{Text: systemPrompt},
				},
			}
		}

		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			errChan <- err
			return
		}

		url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", provider.Endpoint, NormalizeGeminiModel(req.Model), provider.ApiKey)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			errChan <- err
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("Google Text Stream API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
			return
		}

		var lastMetadata googleUsageMetadata
		var lastFinishReason string
		hasMetadata := false

		err = parseSSEStream(resp.Body, func(data string) error {
			var googleResp struct {
				Candidates []struct {
					FinishReason string `json:"finishReason"`
					Content      struct {
						Parts []struct {
							Text    string `json:"text"`
							Thought bool   `json:"thought"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
				UsageMetadata googleUsageMetadata `json:"usageMetadata"`
			}

			if err := json.Unmarshal([]byte(data), &googleResp); err != nil {
				return err
			}

			if len(googleResp.Candidates) > 0 && len(googleResp.Candidates[0].Content.Parts) > 0 {
				part := googleResp.Candidates[0].Content.Parts[0]
				if !part.Thought && part.Text != "" {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case textChan <- part.Text:
					}
				}
			}
			if googleResp.UsageMetadata.TotalTokenCount > 0 {
				lastMetadata = googleResp.UsageMetadata
				hasMetadata = true
				if len(googleResp.Candidates) > 0 {
					lastFinishReason = googleResp.Candidates[0].FinishReason
				}
			}
			if len(googleResp.Candidates) > 0 && googleResp.Candidates[0].FinishReason == "MAX_TOKENS" {
				return fmt.Errorf("Google text generation reached MAX_TOKENS")
			}
			return nil
		})
		if hasMetadata && costPolicy != nil {
			_ = confirmGoogleUsage(context.WithoutCancel(ctx), costPolicy, options, provider, lastMetadata, lastFinishReason)
		}
		if err != nil && err != context.Canceled {
			errChan <- err
		}
	}()

	return textChan, errChan
}

// CallText generates chat text response.
func CallText(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
	costPolicy contract.IAICostPolicyContract,
	options app_ai.TextGenerationOptions,
) (string, error) {
	req := PreparedGeminiRequest{
		Model: provider.Model,
		Body: GeminiGenerateContentBody{
			Contents: []GeminiContent{
				{
					Role: "user",
					Parts: []GeminiPart{
						{Text: userPrompt},
					},
				},
			},
			GenerationConfig: GoogleGenerationConfig(options),
		},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		req.Body.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{
				{Text: systemPrompt},
			},
		}
	}

	bodyBytes, err := json.Marshal(req.Body)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", provider.Endpoint, NormalizeGeminiModel(req.Model), provider.ApiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Google Text API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var googleResp struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata googleUsageMetadata `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return "", err
	}

	if len(googleResp.Candidates) == 0 || len(googleResp.Candidates[0].Content.Parts) == 0 {
		return "", apperror.NewInternalError("Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này.")
	}

	var nonThoughtParts []string
	for _, part := range googleResp.Candidates[0].Content.Parts {
		if !part.Thought && part.Text != "" {
			nonThoughtParts = append(nonThoughtParts, part.Text)
		}
	}

	finishReason := googleResp.Candidates[0].FinishReason
	if costPolicy != nil {
		_ = confirmGoogleUsage(context.WithoutCancel(ctx), costPolicy, options, provider, googleResp.UsageMetadata, finishReason)
	}
	if finishReason == "MAX_TOKENS" {
		return "", fmt.Errorf("Google text generation reached MAX_TOKENS")
	}

	if len(nonThoughtParts) == 0 {
		return "", apperror.NewInternalError("Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này.")
	}

	return strings.Join(nonThoughtParts, ""), nil
}

// CallVision analyzes image content.
func CallVision(ctx context.Context, client *http.Client, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	imgBytes, mimeType, err := httputils.DownloadImage(client, ctx, imageUrl)
	if err != nil {
		return "", fmt.Errorf("failed to download image for Google Vision: %w", err)
	}

	base64Data := base64.StdEncoding.EncodeToString(imgBytes)

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"text": prompt,
					},
					{
						"inlineData": map[string]string{
							"mimeType": mimeType,
							"data":     base64Data,
						},
					},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", provider.Endpoint, provider.Model, provider.ApiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Google Vision API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var googleResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return "", err
	}

	if len(googleResp.Candidates) == 0 || len(googleResp.Candidates[0].Content.Parts) == 0 {
		return "", apperror.NewInternalError("Không thể nhận kết quả phân tích hình ảnh từ hệ thống trí tuệ nhân tạo.")
	}

	return googleResp.Candidates[0].Content.Parts[0].Text, nil
}

// CallEmbeddingBatch generates batch text embeddings.
func CallEmbeddingBatch(ctx context.Context, client *http.Client, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
	var allEmbeddings [][]float32
	maxBatchSize := 100

	modelName := provider.Model
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}

	for i := 0; i < len(chunks); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		subSlice := chunks[i:end]

		requestsList := make([]map[string]any, len(subSlice))
		for idx, text := range subSlice {
			requestsList[idx] = map[string]any{
				"model": modelName,
				"content": map[string]any{
					"parts": []map[string]any{
						{
							"text": text,
						},
					},
				},
			}
		}

		payload := map[string]any{
			"requests": requestsList,
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("%s/%s:batchEmbedContents?key=%s", provider.Endpoint, provider.Model, provider.ApiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Google Embedding Batch API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var googleResp struct {
			Embeddings []struct {
				Values []float32 `json:"values"`
			} `json:"embeddings"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
			return nil, err
		}

		if len(googleResp.Embeddings) == 0 {
			return nil, apperror.NewInternalError("Không thể xử lý dữ liệu đặc trưng của nội dung lúc này.")
		}

		for _, emb := range googleResp.Embeddings {
			allEmbeddings = append(allEmbeddings, sliceutils.AdjustVectorLength(emb.Values, 768))
		}
	}

	return allEmbeddings, nil
}

func confirmGoogleUsage(
	ctx context.Context,
	costPolicy contract.IAICostPolicyContract,
	options app_ai.TextGenerationOptions,
	provider config.APIProviderConfig,
	u googleUsageMetadata,
	finish string,
) error {
	return costPolicy.Confirm(ctx, options.RequestID, contract.AIUsage{
		PromptTokens:   u.PromptTokenCount,
		OutputTokens:   u.CandidatesTokenCount,
		ThinkingTokens: u.ThoughtsTokenCount,
		FinishReason:   finish,
		Provider:       provider.Provider,
		Model:          provider.Model,
	})
}

func parseSSEStream(r io.Reader, callback func(data string) error) error {
	var buffer []byte
	temp := make([]byte, 4096)

	for {
		n, err := r.Read(temp)
		if n > 0 {
			buffer = append(buffer, temp[:n]...)
			for {
				lineEnd := bytes.IndexByte(buffer, '\n')
				if lineEnd < 0 {
					break
				}
				line := string(buffer[:lineEnd])
				buffer = buffer[lineEnd+1:]

				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if err := callback(data); err != nil {
						return err
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

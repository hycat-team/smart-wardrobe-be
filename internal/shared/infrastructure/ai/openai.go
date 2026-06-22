package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/utils/sliceutils"
)

func (s *AIService) callOpenAITextStream(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
) (<-chan string, <-chan error) {
	textChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		payload := map[string]any{
			"model": provider.Model,
			"messages": []map[string]string{
				{
					"role":    "system",
					"content": systemPrompt,
				},
				{
					"role":    "user",
					"content": userPrompt,
				},
			},
			"stream":         true,
			"stream_options": map[string]any{"include_usage": true},
		}
		if options.MaxOutputTokens > 0 {
			payload["max_tokens"] = options.MaxOutputTokens
		}
		applyOpenAIStructuredOutput(payload, options)

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			errChan <- err
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

		resp, err := client.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("OpenAI Text Stream API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
			return
		}

		err = parseSSEStream(resp.Body, func(data string) error {
			if data == "[DONE]" {
				return io.EOF
			}

			var openAIResp struct {
				Choices []struct {
					FinishReason string `json:"finish_reason"`
					Delta        struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage struct {
					PromptTokens     int64 `json:"prompt_tokens"`
					CompletionTokens int64 `json:"completion_tokens"`
					TotalTokens      int64 `json:"total_tokens"`
				} `json:"usage"`
			}

			if err := json.Unmarshal([]byte(data), &openAIResp); err != nil {
				return err
			}

			if len(openAIResp.Choices) > 0 {
				text := openAIResp.Choices[0].Delta.Content
				if text != "" {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case textChan <- text:
					}
				}
			}
			if openAIResp.Usage.TotalTokens > 0 {
				_ = s.costPolicy.Confirm(context.WithoutCancel(ctx), options.RequestID, contract.AIUsage{PromptTokens: openAIResp.Usage.PromptTokens, OutputTokens: openAIResp.Usage.CompletionTokens, Provider: provider.Provider, Model: provider.Model})
			}
			if len(openAIResp.Choices) > 0 && openAIResp.Choices[0].FinishReason == "length" {
				return fmt.Errorf("OpenAI text generation reached MAX_TOKENS")
			}
			return nil
		})
		if err != nil && err != io.EOF && err != context.Canceled {
			errChan <- err
		}
	}()

	return textChan, errChan
}

func (s *AIService) callOpenAIText(ctx context.Context, client *http.Client, provider config.APIProviderConfig, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	payload := map[string]any{
		"model": provider.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
	}
	if options.MaxOutputTokens > 0 {
		payload["max_tokens"] = options.MaxOutputTokens
	}
	applyOpenAIStructuredOutput(payload, options)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI Text API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var openAIResp struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", apperror.NewInternalError("Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này.")
	}
	_ = s.costPolicy.Confirm(context.WithoutCancel(ctx), options.RequestID, contract.AIUsage{PromptTokens: openAIResp.Usage.PromptTokens, OutputTokens: openAIResp.Usage.CompletionTokens, Provider: provider.Provider, Model: provider.Model})
	if openAIResp.Choices[0].FinishReason == "length" {
		return "", fmt.Errorf("OpenAI text generation reached MAX_TOKENS")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func applyOpenAIStructuredOutput(payload map[string]any, options app_ai.TextGenerationOptions) {
	if options.ResponseMIMEType != "application/json" {
		return
	}
	if options.ResponseSchema != nil {
		payload["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "structured_response", "strict": true, "schema": options.ResponseSchema}}
		return
	}
	payload["response_format"] = map[string]any{"type": "json_object"}
}

func (s *AIService) callOpenAIVision(ctx context.Context, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	payload := map[string]any{
		"model": provider.Model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": imageUrl,
						},
					},
				},
			},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

	resp, err := s.visionClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI Vision API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", apperror.NewInternalError("Không thể nhận kết quả phân tích hình ảnh từ hệ thống trí tuệ nhân tạo.")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (s *AIService) callOpenAIEmbeddingBatch(ctx context.Context, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
	var allEmbeddings [][]float32
	maxBatchSize := 100

	for i := 0; i < len(chunks); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		subSlice := chunks[i:end]

		payload := map[string]any{
			"input":      subSlice,
			"model":      provider.Model,
			"dimensions": 768,
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

		resp, err := s.embeddingClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("OpenAI Embedding Batch API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var openAIResp struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			return nil, err
		}

		if len(openAIResp.Data) == 0 {
			return nil, apperror.NewInternalError("Không thể xử lý dữ liệu đặc trưng của nội dung lúc này.")
		}

		batchResults := make([][]float32, len(subSlice))
		for _, d := range openAIResp.Data {
			if d.Index >= 0 && d.Index < len(subSlice) {
				batchResults[d.Index] = sliceutils.AdjustVectorLength(d.Embedding, 768)
			}
		}

		allEmbeddings = append(allEmbeddings, batchResults...)
	}

	return allEmbeddings, nil
}

package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"smart-wardrobe-be/internal/config"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func CallTextStream(
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

		url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", provider.Endpoint, req.Model, provider.ApiKey)
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
		if hasMetadata && options.OnUsage != nil {
			options.OnUsage(app_ai.TokenUsage{
				PromptTokens:   lastMetadata.PromptTokenCount,
				OutputTokens:   lastMetadata.CandidatesTokenCount,
				ThinkingTokens: lastMetadata.ThoughtsTokenCount,
				FinishReason:   lastFinishReason,
			})
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

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", provider.Endpoint, req.Model, provider.ApiKey)
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
	if options.OnUsage != nil {
		options.OnUsage(app_ai.TokenUsage{
			PromptTokens:   googleResp.UsageMetadata.PromptTokenCount,
			OutputTokens:   googleResp.UsageMetadata.CandidatesTokenCount,
			ThinkingTokens: googleResp.UsageMetadata.ThoughtsTokenCount,
			FinishReason:   finishReason,
		})
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

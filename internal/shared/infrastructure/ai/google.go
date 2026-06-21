package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/utils/httputils"
	"smart-wardrobe-be/pkg/utils/sliceutils"
)

func (s *AIService) callGoogleTextStream(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
) (<-chan string, <-chan error) {
	textChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		payload := map[string]any{
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{
							"text": userPrompt,
						},
					},
				},
			},
		}
		if strings.TrimSpace(systemPrompt) != "" {
			payload["systemInstruction"] = map[string]any{
				"parts": []map[string]string{
					{
						"text": systemPrompt,
					},
				},
			}
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			errChan <- err
			return
		}

		url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", provider.Endpoint, provider.Model, provider.ApiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
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

		err = parseSSEStream(resp.Body, func(data string) error {
			var googleResp struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}

			if err := json.Unmarshal([]byte(data), &googleResp); err != nil {
				return err
			}

			if len(googleResp.Candidates) > 0 && len(googleResp.Candidates[0].Content.Parts) > 0 {
				text := googleResp.Candidates[0].Content.Parts[0].Text
				if text != "" {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case textChan <- text:
					}
				}
			}
			return nil
		})
		if err != nil && err != context.Canceled {
			errChan <- err
		}
	}()

	return textChan, errChan
}

func (s *AIService) callGoogleText(ctx context.Context, client *http.Client, provider config.APIProviderConfig, systemPrompt string, userPrompt string) (string, error) {
	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{
						"text": userPrompt,
					},
				},
			},
		},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{
					"text": systemPrompt,
				},
			},
		}
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
		return "", fmt.Errorf("Google Text API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
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
		return "", apperror.NewInternalError("Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này.")
	}

	return googleResp.Candidates[0].Content.Parts[0].Text, nil
}

func (s *AIService) callGoogleVision(ctx context.Context, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	imgBytes, mimeType, err := httputils.DownloadImage(s.visionClient, ctx, imageUrl)
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

	resp, err := s.visionClient.Do(req)
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

func (s *AIService) callGoogleEmbeddingBatch(ctx context.Context, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
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

		resp, err := s.embeddingClient.Do(req)
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

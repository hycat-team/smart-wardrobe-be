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
	"smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/pkg/utils/httputils"
	"smart-wardrobe-be/pkg/utils/sliceutils"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

func (s *AIService) callGoogleText(ctx context.Context, provider config.APIProviderConfig, systemPrompt string, userPrompt string) (string, error) {
	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{
						"text": systemPrompt + "\n\n" + userPrompt,
					},
				},
			},
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

	resp, err := s.cli.Do(req)
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
		return "", apperror.NewInternalError("Không nhận được phản hồi văn bản từ Google AI.")
	}

	return googleResp.Candidates[0].Content.Parts[0].Text, nil
}

func (s *AIService) callGoogleVision(ctx context.Context, provider config.APIProviderConfig, imageUrl string, categories []dto.AICategoryRef) (*dto.FashionMetadataResult, error) {
	imgBytes, mimeType, err := httputils.DownloadImage(s.cli, ctx, imageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to download image for Google Vision: %w", err)
	}

	base64Data := base64.StdEncoding.EncodeToString(imgBytes)
	prompt := getVisionSystemPrompt(categories)

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
		return nil, err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", provider.Endpoint, provider.Model, provider.ApiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google Vision API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
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
		return nil, err
	}

	if len(googleResp.Candidates) == 0 || len(googleResp.Candidates[0].Content.Parts) == 0 {
		return nil, apperror.NewInternalError("Không nhận được phản hồi phân tích hình ảnh từ Google AI.")
	}

	var result struct {
		dto.FashionMetadataResult
		Error string `json:"error"`
	}
	cleanContent := stringutils.CleanJSONMarkdown(googleResp.Candidates[0].Content.Parts[0].Text)
	if err := json.Unmarshal([]byte(cleanContent), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from Google AI: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("Google AI Error: %s", result.Error)
	}

	return &result.FashionMetadataResult, nil
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

		resp, err := s.cli.Do(req)
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
			return nil, apperror.NewInternalError("Không nhận được phản hồi mã hóa từ Google AI.")
		}

		for _, emb := range googleResp.Embeddings {
			allEmbeddings = append(allEmbeddings, sliceutils.AdjustVectorLength(emb.Values, 768))
		}
	}

	return allEmbeddings, nil
}


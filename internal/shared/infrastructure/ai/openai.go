package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/pkg/utils/sliceutils"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

func (s *AIService) callOpenAIVision(ctx context.Context, provider config.APIProviderConfig, imageUrl string) (*dto.FashionMetadataResult, error) {
	prompt := getVisionSystemPrompt()

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
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

	resp, err := s.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI Vision API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, err
	}

	if len(openAIResp.Choices) == 0 {
		return nil, errorcode.NewInternalError("Không nhận được phản hồi phân tích hình ảnh.")
	}

	var result dto.FashionMetadataResult
	cleanContent := stringutils.CleanJSONMarkdown(openAIResp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(cleanContent), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from AI: %w", err)
	}

	return &result, nil
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

		resp, err := s.cli.Do(req)
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
			return nil, errorcode.NewInternalError("Không nhận được phản hồi mã hóa từ OpenAI.")
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

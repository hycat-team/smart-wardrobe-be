package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/httputils"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"
)

func CallVision(ctx context.Context, client *http.Client, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	text, _, err := CallVisionWithUsage(ctx, client, provider, imageUrl, prompt)
	return text, err
}

// CallVisionWithUsage analyzes image content and reports the provider usage.
// The text result is identical to CallVision; the usage comes from the
// generateContent usageMetadata. When the response carries no usageMetadata
// the returned usage is zero and the call still succeeds.
func CallVisionWithUsage(ctx context.Context, client *http.Client, provider config.APIProviderConfig, imageUrl string, prompt string) (string, app_ai.TokenUsage, error) {
	imgBytes, mimeType, err := httputils.DownloadImage(client, ctx, imageUrl)
	if err != nil {
		return "", app_ai.TokenUsage{}, fmt.Errorf("failed to download image for Google Vision: %w", err)
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
		return "", app_ai.TokenUsage{}, err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", provider.Endpoint, provider.Model, provider.ApiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", app_ai.TokenUsage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", app_ai.TokenUsage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", app_ai.TokenUsage{}, apperror.NewUpstreamUnavailable(fmt.Errorf("Google Vision API error (HTTP %d): %s", resp.StatusCode, string(respBytes)))
	}

	var googleResp struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata googleUsageMetadata `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return "", app_ai.TokenUsage{}, err
	}

	if len(googleResp.Candidates) == 0 || len(googleResp.Candidates[0].Content.Parts) == 0 {
		return "", app_ai.TokenUsage{}, apperror.NewInternalError("Không thể nhận kết quả phân tích hình ảnh từ hệ thống trí tuệ nhân tạo.")
	}

	usage := app_ai.TokenUsage{
		PromptTokens:   googleResp.UsageMetadata.PromptTokenCount,
		OutputTokens:   googleResp.UsageMetadata.CandidatesTokenCount,
		ThinkingTokens: googleResp.UsageMetadata.ThoughtsTokenCount,
	}
	if len(googleResp.Candidates) > 0 {
		usage.FinishReason = googleResp.Candidates[0].FinishReason
	}

	return googleResp.Candidates[0].Content.Parts[0].Text, usage, nil
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
			return nil, apperror.NewUpstreamUnavailable(fmt.Errorf("Google Embedding Batch API error (HTTP %d): %s", resp.StatusCode, string(respBytes)))
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

package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *AIService) downloadImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := s.cli.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return data, mimeType, nil
}

func (s *AIService) adjustVectorLength(vec []float32, targetLength int) []float32 {
	if len(vec) == targetLength {
		return vec
	}
	if len(vec) > targetLength {
		return vec[:targetLength]
	}
	res := make([]float32, targetLength)
	copy(res, vec)
	return res
}

func cleanJSONMarkdown(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}
	return strings.TrimSpace(content)
}

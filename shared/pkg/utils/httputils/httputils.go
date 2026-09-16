package httputils

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const maxImageSize = 15 * 1024 * 1024

func DownloadImage(cli *http.Client, ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request failed: %w", err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("execute request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	limitReader := io.LimitReader(resp.Body, maxImageSize)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, "", fmt.Errorf("read body failed: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")

	if mimeType == "" && len(data) > 0 {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}

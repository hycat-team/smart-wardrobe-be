package httputils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const maxImageSize = 15 * 1024 * 1024

// maxImageDownloadAttempts and imageDownloadRetryDelay bound how long a caller
// waits for an asset that the CDN says is still being generated.
//
// Cloudinary answers 423 Locked until a derived version exists (the wardrobe
// flow requests a background-removal transformation right after upload, and
// production measured 10-24s of generation). Four attempts with a five second
// gap covers the common case while staying inside the vision client's 20s
// timeout budget; anything slower is handled by the analyze retry queue, which
// waits a minute.
const (
	maxImageDownloadAttempts = 4
	imageDownloadRetryDelay  = 5 * time.Second
	// maxRetryAfter bounds a provider-supplied Retry-After so one slow asset
	// cannot stall a worker for minutes.
	maxRetryAfter = 15 * time.Second
)

// DownloadImage fetches an image, waiting out a "still being generated" answer.
//
// 423 Locked is not a failure: Cloudinary returns it while the requested derived
// version is still being produced. Treating it as fatal (the previous behaviour)
// turned every freshly uploaded wardrobe photo into a failed analysis.
func DownloadImage(cli *http.Client, ctx context.Context, url string) ([]byte, string, error) {
	return downloadImage(cli, ctx, url, maxImageDownloadAttempts, imageDownloadRetryDelay)
}

// downloadImage carries the retry policy so a test can shrink the delay without
// touching package state.
func downloadImage(cli *http.Client, ctx context.Context, url string, attempts int, retryDelay time.Duration) ([]byte, string, error) {
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		data, mimeType, retryAfter, retryable, err := tryDownloadImage(cli, ctx, url)
		if err == nil {
			return data, mimeType, nil
		}
		lastErr = err

		if !retryable || attempt == attempts {
			break
		}

		wait := retryDelay
		if retryAfter > 0 {
			wait = retryAfter
		}
		if waitErr := sleepWithContext(ctx, wait); waitErr != nil {
			return nil, "", waitErr
		}
	}

	return nil, "", lastErr
}

// tryDownloadImage performs one request and reports whether the response is a
// "try again shortly" answer rather than a definitive failure, plus the delay
// the provider asked for when it sent one.
func tryDownloadImage(cli *http.Client, ctx context.Context, url string) ([]byte, string, time.Duration, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", 0, false, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, "", 0, false, fmt.Errorf("execute request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain a bounded amount so the connection can be reused by the retry.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4*1024))
		return nil, "", retryAfterFromHeader(resp), isImageNotReady(resp),
			fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	limitReader := io.LimitReader(resp.Body, maxImageSize)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, "", 0, false, fmt.Errorf("read body failed: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")

	if mimeType == "" && len(data) > 0 {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, 0, false, nil
}

// isImageNotReady reports whether the status means "the derived asset is still
// being generated". 423 is Cloudinary's documented answer for images, 420 for an
// incoming transformation that is still pending.
func isImageNotReady(resp *http.Response) bool {
	return resp.StatusCode == http.StatusLocked || resp.StatusCode == 420
}

// sleepWithContext waits for d unless the caller gives up first.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryAfterFromHeader reads the seconds form of Retry-After, capped by
// maxRetryAfter. Cloudinary does not send it today, but a provider that does
// should not be ignored.
func retryAfterFromHeader(resp *http.Response) time.Duration {
	seconds, err := strconv.Atoi(resp.Header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return 0
	}
	d := time.Duration(seconds) * time.Second
	if d > maxRetryAfter {
		return maxRetryAfter
	}
	return d
}

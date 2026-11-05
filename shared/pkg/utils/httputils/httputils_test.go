package httputils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// 423 Locked means the derived asset is still being generated. Treating it as a
// final error is what turned every freshly uploaded wardrobe photo into
// "analysis_temporary_error" (production measured 10-24s of Cloudinary
// generation while the old code gave up in milliseconds).
func TestDownloadImage_WaitsOutLockedThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusLocked)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer srv.Close()

	data, mimeType, err := downloadImage(http.DefaultClient, context.Background(), srv.URL, 4, time.Millisecond)
	if err != nil {
		t.Fatalf("downloadImage returned error after a retryable 423: %v", err)
	}
	if string(data) != "png-bytes" {
		t.Errorf("data = %q, want png-bytes", string(data))
	}
	if mimeType != "image/png" {
		t.Errorf("mimeType = %q, want image/png", mimeType)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2 (one 423 then success)", got)
	}
}

func TestDownloadImage_GivesUpAfterTheAttemptBudget(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusLocked)
	}))
	defer srv.Close()

	if _, _, err := downloadImage(http.DefaultClient, context.Background(), srv.URL, 3, time.Millisecond); err == nil {
		t.Fatal("expected an error once the attempt budget is spent")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestDownloadImage_NonRetryableStatusFailsImmediately(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, _, err := downloadImage(http.DefaultClient, context.Background(), srv.URL, 4, time.Millisecond); err == nil {
		t.Fatal("expected an error for 404")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1: a 404 is permanent", got)
	}
}

func TestDownloadImage_StopsWhenContextIsCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusLocked)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, err := downloadImage(http.DefaultClient, ctx, srv.URL, 4, time.Millisecond); err == nil {
		t.Fatal("expected an error from a cancelled context")
	}
}

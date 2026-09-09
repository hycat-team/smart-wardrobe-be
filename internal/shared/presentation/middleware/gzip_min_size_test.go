package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/internal/config"

	"github.com/gin-gonic/gin"
)

func gzipTestRouter(t *testing.T, gzipEnabled bool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{}
	cfg.Server.GzipEnabled = gzipEnabled
	r := gin.New()
	r.Use(GzipMinSizeMiddleware(&cfg))
	large := strings.Repeat("x", 4<<10) // 4KB > 1KB threshold
	r.GET("/api/v1/dashboard/overview", func(c *gin.Context) {
		c.String(http.StatusOK, large)
	})
	r.GET("/api/v1/ai/chat/sessions/abc/messages/stream", func(c *gin.Context) {
		c.String(http.StatusOK, large)
	})
	r.GET("/api/v1/small", func(c *gin.Context) {
		c.String(http.StatusOK, "tiny")
	})
	return r
}

func doGzipRequest(r *gin.Engine, path, encoding string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if encoding != "" {
		req.Header.Set("Accept-Encoding", encoding)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGzipMinSizeCompressesLargeResponse(t *testing.T) {
	r := gzipTestRouter(t, true)

	w := doGzipRequest(r, "/api/v1/dashboard/overview", "gzip")
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content encoding, got headers %v", w.Header())
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("expected valid gzip body: %v", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}
	if len(raw) != 4<<10 || !bytes.Contains(raw, []byte("xxxx")) {
		t.Fatalf("unexpected decompressed payload length %d", len(raw))
	}
}

func TestGzipMinSizeSkipsSmallResponses(t *testing.T) {
	r := gzipTestRouter(t, true)

	w := doGzipRequest(r, "/api/v1/small", "gzip")
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("expected small response to stay uncompressed")
	}
	if w.Body.String() != "tiny" {
		t.Fatalf("expected raw payload, got %q", w.Body.String())
	}
}

func TestGzipMinSizeSkipsExcludedStreamPath(t *testing.T) {
	r := gzipTestRouter(t, true)

	w := doGzipRequest(r, "/api/v1/ai/chat/sessions/abc/messages/stream", "gzip")
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("expected SSE stream path to stay uncompressed")
	}
}

func TestGzipMinSizeSkipsSSEAcceptHeader(t *testing.T) {
	r := gzipTestRouter(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/overview", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("expected text/event-stream request to bypass gzip compression")
	}
}

func TestGzipMinSizeSkipsWhenDisabledOrClientUnsupported(t *testing.T) {
	r := gzipTestRouter(t, false)
	w := doGzipRequest(r, "/api/v1/dashboard/overview", "gzip")
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("expected no compression when disabled")
	}

	r = gzipTestRouter(t, true)
	w = doGzipRequest(r, "/api/v1/dashboard/overview", "")
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("expected no compression when client does not accept gzip")
	}
}

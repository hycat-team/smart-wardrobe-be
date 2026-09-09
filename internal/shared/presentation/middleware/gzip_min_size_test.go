package middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"

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

// gzipErrorChainRouter reproduces the production middleware order in
// internal/api/routes/router.go: GlobalErrorHandler (outer) wraps
// GzipMinSizeMiddleware, which wraps a handler that fails via c.Error
// without writing any body — exactly what AuthMiddleware.Handle() does on
// a missing/invalid access token.
func gzipErrorChainRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{}
	cfg.Server.GzipEnabled = true
	r := gin.New()
	r.Use(GlobalErrorHandler(testLogger{}, "test"))
	r.Use(GzipMinSizeMiddleware(&cfg))
	r.GET("/api/v1/me", func(c *gin.Context) {
		c.Error(apperror.ErrUnauthorized())
		c.Abort()
	})
	return r
}

func TestGzipDoesNotSwallowAuthError(t *testing.T) {
	r := gzipErrorChainRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body %q)", w.Code, w.Body.String())
	}
	var body struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body, got %q: %v", w.Body.String(), err)
	}
	if body.Status != http.StatusUnauthorized {
		t.Fatalf("expected error status 401 in body, got %+v", body)
	}
}

func TestGzipErrorWithoutClientGzipStill401(t *testing.T) {
	r := gzipErrorChainRouter(t)

	w := doGzipRequest(r, "/api/v1/me", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body %q)", w.Code, w.Body.String())
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

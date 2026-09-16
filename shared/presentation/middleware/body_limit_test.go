package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func newBodyLimitTestRouter(t *testing.T, cfg config.Config) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewBodyLimitMiddleware(&cfg, &noopTestLogger{}))
	r.POST("/api/v1/wardrobe/items", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/ai/chat/sessions", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/custom/import", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func bodyLimitTestConfig() config.Config {
	cfg := config.Config{}
	cfg.Server.Env = "development"
	cfg.Server.BodyLimit.DefaultMB = 2
	cfg.Server.BodyLimit.ElevatedMB = 10
	cfg.Server.BodyLimit.FreeRoutes = []string{"POST /api/v1/custom"}
	return cfg
}

func performBodyLimitRequest(r *gin.Engine, method, path string, body string, setRealLength bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if setRealLength {
		req.ContentLength = int64(len(body))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBodyLimitRejectsOversizedDefaultTier(t *testing.T) {
	r := newBodyLimitTestRouter(t, bodyLimitTestConfig())
	oversized := string(bytes.Repeat([]byte("x"), 3<<20))

	w := performBodyLimitRequest(r, http.MethodPost, "/api/v1/wardrobe/items", oversized, true)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Dữ liệu gửi lên quá lớn") {
		t.Fatalf("expected standard error envelope, got %q", w.Body.String())
	}
}

func TestBodyLimitAllowsElevatedTierAboveDefault(t *testing.T) {
	r := newBodyLimitTestRouter(t, bodyLimitTestConfig())
	oversized := string(bytes.Repeat([]byte("x"), 3<<20))

	w := performBodyLimitRequest(r, http.MethodPost, "/api/v1/ai/chat/sessions", oversized, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected elevated tier to pass 3MB body, got %d", w.Code)
	}
}

func TestBodyLimitRejectsElevatedTierAboveElevatedLimit(t *testing.T) {
	r := newBodyLimitTestRouter(t, bodyLimitTestConfig())
	oversized := string(bytes.Repeat([]byte("x"), 11<<20))

	w := performBodyLimitRequest(r, http.MethodPost, "/api/v1/ai/chat/sessions", oversized, true)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 above elevated limit, got %d", w.Code)
	}
}

func TestBodyLimitFreeRouteBypassesLimit(t *testing.T) {
	r := newBodyLimitTestRouter(t, bodyLimitTestConfig())
	oversized := string(bytes.Repeat([]byte("x"), 11<<20))

	w := performBodyLimitRequest(r, http.MethodPost, "/api/v1/custom/import", oversized, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected free route to bypass limit, got %d", w.Code)
	}
}

func TestBodyLimitMethodMismatchFallsToDefault(t *testing.T) {
	r := newBodyLimitTestRouter(t, bodyLimitTestConfig())
	// PUT is not in the AI elevated rule (POST only) — 3MB must be rejected.
	w := performBodyLimitRequest(r, http.MethodPut, "/api/v1/ai/chat/sessions", string(bytes.Repeat([]byte("x"), 3<<20)), true)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for method outside elevated rule, got %d", w.Code)
	}
}

type noopTestLogger struct{}

func (l *noopTestLogger) Debug(message string, fields ...zap.Field) {}
func (l *noopTestLogger) Info(message string, fields ...zap.Field)  {}
func (l *noopTestLogger) Warn(message string, fields ...zap.Field)  {}
func (l *noopTestLogger) Error(message string, fields ...zap.Field) {}
func (l *noopTestLogger) Fatal(message string, fields ...zap.Field) {}

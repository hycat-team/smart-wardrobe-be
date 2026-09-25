package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

var corsTestOrigins = []string{
	"https://closy.hycat.online",
	"https://smart-wardrobe-demo.vercel.app/",
}

func newCORSTestRouter(allowOrigins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware(allowOrigins))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func performCORSRequest(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/ping", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORSEchoesFirstOrigin(t *testing.T) {
	r := newCORSTestRouter(corsTestOrigins)
	w := performCORSRequest(r, http.MethodGet, "https://closy.hycat.online")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://closy.hycat.online" {
		t.Fatalf("expected first origin echoed, got %q", got)
	}
}

func TestCORSEchoesSecondOriginDespiteTrailingSlashInConfig(t *testing.T) {
	r := newCORSTestRouter(corsTestOrigins)
	w := performCORSRequest(r, http.MethodGet, "https://smart-wardrobe-demo.vercel.app")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://smart-wardrobe-demo.vercel.app" {
		t.Fatalf("expected second origin echoed without fallback, got %q", got)
	}
}

func TestCORSRejectsUnknownOriginWithoutFallback(t *testing.T) {
	r := newCORSTestRouter(corsTestOrigins)
	w := performCORSRequest(r, http.MethodGet, "https://evil.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin for unknown origin, got %q", got)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected non-OPTIONS unknown origin to pass through, got %d", w.Code)
	}
}

func TestCORSNoOriginHeaderSetsNothing(t *testing.T) {
	r := newCORSTestRouter(corsTestOrigins)
	w := performCORSRequest(r, http.MethodGet, "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin when Origin header missing, got %q", got)
	}
}

func TestCORSPreflightAllowed(t *testing.T) {
	r := newCORSTestRouter(corsTestOrigins)
	w := performCORSRequest(r, http.MethodOptions, "https://smart-wardrobe-demo.vercel.app")
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for allowed preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://smart-wardrobe-demo.vercel.app" {
		t.Fatalf("expected preflight echo, got %q", got)
	}
}

func TestCORSPreflightDenied(t *testing.T) {
	r := newCORSTestRouter(corsTestOrigins)
	w := performCORSRequest(r, http.MethodOptions, "https://evil.example.com")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for denied preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin for denied preflight, got %q", got)
	}
}

func TestCORSToleratesSpacesAndEmptyEntries(t *testing.T) {
	r := newCORSTestRouter([]string{"https://a.example.com", " https://b.example.com ", "", " "})
	w := performCORSRequest(r, http.MethodGet, "https://b.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://b.example.com" {
		t.Fatalf("expected trimmed origin echoed, got %q", got)
	}
}

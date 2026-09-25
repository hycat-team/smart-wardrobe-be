package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/config"

	"github.com/gin-gonic/gin"
)

type recordingLimiter struct {
	keys []string
}

func (r *recordingLimiter) Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	r.keys = append(r.keys, key)
	return true, nil
}

func newRateLimitRouter(cfg *config.Config, limiter *recordingLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewLoginRateLimitMiddleware(cfg, limiter).Handle())
	router.GET("/auth/google", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/auth/google/callback", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func rateLimitTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.ClaimRateLimit.UserLimit = 10
	cfg.RateLimit.ReplenishmentSeconds = 60
	return cfg
}

// The Google redirect endpoints live under /auth and must inherit the login
// shield, keyed by IP when no identifier is present (FR-022).
func TestLoginRateLimit_GoogleGETKeyedByIP(t *testing.T) {
	limiter := &recordingLimiter{}
	router := newRateLimitRouter(rateLimitTestConfig(), limiter)

	req := httptest.NewRequest(http.MethodGet, "/auth/google?redirectUrl=https://closy.hycat.online/cb", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	router.ServeHTTP(httptest.NewRecorder(), req)

	if len(limiter.keys) != 1 {
		t.Fatalf("expected 1 limiter call, got %d", len(limiter.keys))
	}
	if !strings.HasPrefix(limiter.keys[0], "rl:login:1.2.3.4:") {
		t.Fatalf("expected key keyed by IP, got %q", limiter.keys[0])
	}
	if !strings.HasSuffix(limiter.keys[0], ":anonymous") {
		t.Fatalf("expected anonymous identifier for redirect start, got %q", limiter.keys[0])
	}
}

func TestLoginRateLimit_IdentifierFromQuery(t *testing.T) {
	limiter := &recordingLimiter{}
	router := newRateLimitRouter(rateLimitTestConfig(), limiter)

	req := httptest.NewRequest(http.MethodGet, "/auth/google?identifier=duck", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	router.ServeHTTP(httptest.NewRecorder(), req)

	if len(limiter.keys) != 1 {
		t.Fatalf("expected 1 limiter call, got %d", len(limiter.keys))
	}
	if !strings.HasSuffix(limiter.keys[0], ":duck") {
		t.Fatalf("expected identifier from query, got %q", limiter.keys[0])
	}
}

package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// sharedFakeLimiter simulates the Redis backend shared by all instances: one
// global counter per key, so limits hold across replicas (T049a, SC-028).
type sharedFakeLimiter struct {
	mu     sync.Mutex
	counts map[string]int
}

func newSharedFakeLimiter() *sharedFakeLimiter {
	return &sharedFakeLimiter{counts: map[string]int{}}
}

func (f *sharedFakeLimiter) Allow(_ context.Context, key string, limit int, _ int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.counts[key] >= limit {
		return false, nil
	}
	f.counts[key]++
	return true, nil
}

func rateTestConfig() config.Config {
	var cfg config.Config
	cfg.RateLimit.TokenLimit = 3
	cfg.RateLimit.TokensPerPeriod = 3
	cfg.RateLimit.ReplenishmentSeconds = 60
	cfg.ClaimRateLimit.UserLimit = 2
	cfg.ClaimRateLimit.IPLimit = 2
	return cfg
}

// renderAppErrors mimics the global error handler: it turns a recorded
// apperror into its HTTP status so tests observe the real 429.
func renderAppErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		if ae := apperror.From(c.Errors.Last().Err); ae != nil {
			c.AbortWithStatusJSON(ae.Status, gin.H{"message": ae.Message})
		}
	}
}

func serveThrough(t *testing.T, h gin.HandlerFunc) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(renderAppErrors())
	r.Use(h)
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "10.0.0.9:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// TestGeneralRuleSharedAcrossInstances proves the general API rule is enforced
// globally: two middleware instances backed by the same limiter allow 3
// requests in total, not 3 each (T049a, SC-028).
func TestGeneralRuleSharedAcrossInstances(t *testing.T) {
	cfg := rateTestConfig()
	backend := newSharedFakeLimiter()
	instanceA := shared_middleware.NewRateLimitMiddleware(&cfg, backend)
	instanceB := shared_middleware.NewRateLimitMiddleware(&cfg, backend)

	if code := serveThrough(t, instanceA.Handle()); code != http.StatusOK {
		t.Fatalf("request 1 via A: got %d", code)
	}
	if code := serveThrough(t, instanceA.Handle()); code != http.StatusOK {
		t.Fatalf("request 2 via A: got %d", code)
	}
	if code := serveThrough(t, instanceB.Handle()); code != http.StatusOK {
		t.Fatalf("request 3 via B: got %d", code)
	}
	if code := serveThrough(t, instanceB.Handle()); code != http.StatusTooManyRequests {
		t.Fatalf("request 4 via B: expected 429, got %d", code)
	}
}

// TestLoginRuleSharedAcrossInstances proves the login/OTP rule (IP+identifier)
// is enforced globally across instances (T049a, SC-028).
func TestLoginRuleSharedAcrossInstances(t *testing.T) {
	cfg := rateTestConfig()
	backend := newSharedFakeLimiter()
	instanceA := shared_middleware.NewLoginRateLimitMiddleware(&cfg, backend)
	instanceB := shared_middleware.NewLoginRateLimitMiddleware(&cfg, backend)

	serveLogin := func(t *testing.T, h gin.HandlerFunc, user string) int {
		t.Helper()
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(renderAppErrors())
		r.Use(h)
		r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
		req := httptest.NewRequest(http.MethodGet, "/probe?username="+user, nil)
		req.RemoteAddr = "10.0.0.9:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := serveLogin(t, instanceA.Handle(), "alice"); code != http.StatusOK {
		t.Fatalf("login 1 via A: got %d", code)
	}
	if code := serveLogin(t, instanceB.Handle(), "alice"); code != http.StatusOK {
		t.Fatalf("login 2 via B: got %d", code)
	}
	if code := serveLogin(t, instanceA.Handle(), "alice"); code != http.StatusTooManyRequests {
		t.Fatalf("login 3 for same user: expected 429, got %d", code)
	}
	if code := serveLogin(t, instanceB.Handle(), "bob"); code != http.StatusOK {
		t.Fatalf("login for another user: expected 200, got %d", code)
	}
}

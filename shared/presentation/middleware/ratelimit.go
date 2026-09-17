package middleware

import (
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	"smart-wardrobe-be/shared/config"

	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware is a Redis-backed fixed-window rate limiter (FR-054).
// It is accurate across instances because all replicas share the same
// IRateLimiter backend. A backend error fails open to preserve availability.
type RateLimitMiddleware struct {
	limiter       shared_caching.IRateLimiter
	limit         int
	windowSeconds int
	keyPrefix     string
}

// NewRateLimitMiddleware builds the general API rule keyed by client IP.
func NewRateLimitMiddleware(cfg *config.Config, limiter shared_caching.IRateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter:       limiter,
		limit:         cfg.RateLimit.TokenLimit,
		windowSeconds: cfg.RateLimit.ReplenishmentSeconds,
		keyPrefix:     "rl:api:",
	}
}

func (m *RateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.check(c, m.keyPrefix+c.ClientIP())
	}
}

// LoginRateLimitMiddleware is the stricter login/OTP shield keyed by client
// IP plus user/token identifier. Thresholds come from claim_rate_limit
// (user_limit, fallback ip_limit); the window reuses
// rate_limit.replenishment_seconds — nothing is hardcoded.
type LoginRateLimitMiddleware struct {
	limiter       shared_caching.IRateLimiter
	limit         int
	windowSeconds int
}

// NewLoginRateLimitMiddleware builds the login/OTP rule.
func NewLoginRateLimitMiddleware(cfg *config.Config, limiter shared_caching.IRateLimiter) *LoginRateLimitMiddleware {
	limit := cfg.ClaimRateLimit.UserLimit
	if limit <= 0 {
		limit = cfg.ClaimRateLimit.IPLimit
	}
	if limit <= 0 {
		limit = cfg.RateLimit.TokenLimit
	}
	return &LoginRateLimitMiddleware{
		limiter:       limiter,
		limit:         limit,
		windowSeconds: cfg.RateLimit.ReplenishmentSeconds,
	}
}

func (m *LoginRateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:login:" + c.ClientIP() + ":" + loginIdentifier(c)
		allowed, err := m.limiter.Allow(c.Request.Context(), key, m.limit, m.windowSeconds)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			c.Error(apperror.NewTooManyRequest("Bạn đang thao tác quá nhanh. Vui lòng thử lại sau ít phút."))
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *RateLimitMiddleware) check(c *gin.Context, key string) {
	allowed, err := m.limiter.Allow(c.Request.Context(), key, m.limit, m.windowSeconds)
	if err != nil {
		c.Next()
		return
	}
	if !allowed {
		c.Error(apperror.NewTooManyRequest("Bạn đang thao tác quá nhanh. Vui lòng thử lại sau ít phút."))
		c.Abort()
		return
	}
	c.Next()
}

// loginIdentifier binds the rule to the attempting account as well as the IP:
// explicit username/email/phone/identifier, else a bearer token suffix, else
// the IP alone.
func loginIdentifier(c *gin.Context) string {
	for _, field := range []string{"username", "email", "phone", "identifier", "userId"} {
		if v := c.Query(field); v != "" {
			return v
		}
		if v := c.PostForm(field); v != "" {
			return v
		}
	}
	if auth := c.GetHeader("Authorization"); len(auth) > 20 {
		return auth[len(auth)-12:]
	}
	return "anonymous"
}

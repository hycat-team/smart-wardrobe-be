package middleware

import (
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimitMiddleware struct {
	tokenLimit           int
	tokensPerPeriod      int
	replenishmentSeconds int
	visitors             sync.Map
}

type visitor struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
	mu       sync.Mutex
}

func NewRateLimitMiddleware(cfg *config.Config) *RateLimitMiddleware {
	m := &RateLimitMiddleware{
		tokenLimit:           cfg.RateLimit.TokenLimit,
		tokensPerPeriod:      cfg.RateLimit.TokensPerPeriod,
		replenishmentSeconds: cfg.RateLimit.ReplenishmentSeconds,
	}

	go m.startCleanupJob(5*time.Minute, 1*time.Minute)

	return m
}

func (m *RateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		limiter := m.getLimiter(key)
		if !limiter.Allow() {
			c.Error(apperror.NewTooManyRequest("Bạn đang thao tác quá nhanh. Vui lòng thử lại sau ít phút."))
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *RateLimitMiddleware) getLimiter(key string) *rate.Limiter {
	v, exists := m.visitors.Load(key)

	if !exists {
		r := rate.Limit(float64(m.tokensPerPeriod) / float64(m.replenishmentSeconds))
		limiter := rate.NewLimiter(r, m.tokenLimit)
		newVisitor := &visitor{Limiter: limiter, LastSeen: time.Now()}

		actual, loaded := m.visitors.LoadOrStore(key, newVisitor)
		if loaded {
			v = actual
		} else {
			return limiter
		}
	}

	vis := v.(*visitor)
	vis.mu.Lock()
	vis.LastSeen = time.Now()
	vis.mu.Unlock()

	return vis.Limiter
}

func (m *RateLimitMiddleware) startCleanupJob(expireAfter time.Duration, interval time.Duration) {
	for {
		time.Sleep(interval)
		m.visitors.Range(func(key, value any) bool {
			vis := value.(*visitor)

			vis.mu.Lock()
			isExpired := time.Since(vis.LastSeen) > expireAfter
			vis.mu.Unlock()

			if isExpired {
				m.visitors.Delete(key)
			}
			return true
		})
	}
}

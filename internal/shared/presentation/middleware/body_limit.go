package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Body size tiers (spec 019 FR-002): every request is capped at DefaultMB,
// routes matched by the elevated rules at ElevatedMB, and routes listed in
// the configured free registry bypass the limit entirely.
type bodyTier string

const (
	bodyTierDefault  bodyTier = "default"
	bodyTierElevated bodyTier = "elevated"
	bodyTierFree     bodyTier = "free"
)

// bodyLimitRule matches `METHOD prefix` with an optional substring condition.
type bodyLimitRule struct {
	methods  []string
	prefix   string
	contains string
}

// elevatedBodyLimitRules enumerate the routes whose payloads legitimately
// exceed the default limit (research.md D2): AI chat/recommendation, brand
// conversations and brand/admin mutation routes with description fields.
var elevatedBodyLimitRules = []bodyLimitRule{
	{methods: []string{http.MethodPost}, prefix: "/api/v1/ai/"},
	{methods: []string{http.MethodPost}, prefix: "/api/v1/brands/", contains: "conversation"},
	{methods: []string{http.MethodPost, http.MethodPut, http.MethodPatch}, prefix: "/api/v1/brand-portal/"},
	{methods: []string{http.MethodPost, http.MethodPut, http.MethodPatch}, prefix: "/api/v1/admin/brands/"},
	{methods: []string{http.MethodPost}, prefix: "/api/v1/brands/claim"},
	{methods: []string{http.MethodPost}, prefix: "/api/v1/brands/", contains: "join-loyalty"},
	{methods: []string{http.MethodPost}, prefix: "/api/v1/brand-items/", contains: "feedbacks"},
	{methods: []string{http.MethodPost}, prefix: "/api/v1/brand-benefits/", contains: "redeem"},
}

type bodyLimitFreeRule struct {
	method string
	prefix string
}

// NewBodyLimitMiddleware enforces the tiered request body size limit.
// Oversized bodies with a known Content-Length are rejected immediately with
// a 413 apperror; chunked bodies are capped by http.MaxBytesReader and the
// resulting *http.MaxBytesError is translated to 413 in
// validation.TranslateValidationError.
func NewBodyLimitMiddleware(cfg *config.Config, log logger.Interface) gin.HandlerFunc {
	free := make([]bodyLimitFreeRule, 0, len(cfg.Server.BodyLimit.FreeRoutes))
	for _, route := range cfg.Server.BodyLimit.FreeRoutes {
		parts := strings.Fields(route)
		if len(parts) == 2 {
			free = append(free, bodyLimitFreeRule{method: strings.ToUpper(parts[0]), prefix: parts[1]})
		}
	}
	defaultLimit := int64(cfg.Server.BodyLimit.DefaultMB) << 20
	elevatedLimit := int64(cfg.Server.BodyLimit.ElevatedMB) << 20

	return func(c *gin.Context) {
		limit, tier := resolveBodyLimit(c.Request.Method, c.Request.URL.Path, free, defaultLimit, elevatedLimit)
		if tier == bodyTierFree {
			c.Next()
			return
		}

		if c.Request.ContentLength > limit {
			log.Warn("Request body limit exceeded",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("tier", string(tier)),
				zap.Int64("content_length", c.Request.ContentLength),
				zap.Int64("limit", limit),
			)
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, apperror.NewRequestEntityTooLarge(bodyLimitDetail(limit)))
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func resolveBodyLimit(method, path string, free []bodyLimitFreeRule, defaultLimit, elevatedLimit int64) (int64, bodyTier) {
	for _, rule := range free {
		if method == rule.method && strings.HasPrefix(path, rule.prefix) {
			return 0, bodyTierFree
		}
	}
	for _, rule := range elevatedBodyLimitRules {
		if !bodyLimitMethodMatch(rule.methods, method) || !strings.HasPrefix(path, rule.prefix) {
			continue
		}
		if rule.contains != "" && !strings.Contains(path, rule.contains) {
			continue
		}
		return elevatedLimit, bodyTierElevated
	}
	return defaultLimit, bodyTierDefault
}

func bodyLimitMethodMatch(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

func bodyLimitDetail(limit int64) string {
	return fmt.Sprintf("Dữ liệu gửi lên vượt quá giới hạn %dMB.", limit>>20)
}

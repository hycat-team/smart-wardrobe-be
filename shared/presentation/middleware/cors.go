package middleware

import (
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"
)

func normalizeOrigin(o string) string {
	o = strings.TrimSpace(o)
	o = strings.TrimRight(o, "/")
	return o
}

func CORSMiddleware(allowOrigins string) gin.HandlerFunc {
	allowedMap := make(map[string]bool)
	for _, o := range strings.Split(allowOrigins, ",") {
		if normalized := normalizeOrigin(o); normalized != "" {
			allowedMap[normalized] = true
		}
	}
	allowAll := allowedMap["*"]

	return func(c *gin.Context) {
		origin := normalizeOrigin(c.Request.Header.Get("Origin"))
		allowed := origin != "" && (allowAll || allowedMap[origin])

		if allowed {
			// Echo exact request origin (never the first configured origin,
			// never "*" when credentials are enabled).
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH, HEAD")
		}

		if c.Request.Method == "OPTIONS" {
			if !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

package middleware

import (
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowOrigins string) gin.HandlerFunc {
	originsList := strings.Split(allowOrigins, ",")
	allowedMap := make(map[string]bool)
	for _, o := range originsList {
		allowedMap[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedMap[origin] || allowedMap["*"] {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(originsList) > 0 && originsList[0] != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", originsList[0])
			}
		} else if len(originsList) > 0 && originsList[0] != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", originsList[0])
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH, HEAD")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

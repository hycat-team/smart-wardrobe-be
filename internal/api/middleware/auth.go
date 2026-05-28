package middleware

import (
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/jwttype"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/jwtutils"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	cfg *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		cfg: cfg,
	}
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		// Try to read from cookie first
		if cookieToken, err := c.Cookie(contextutils.CookieAccessToken); err == nil && cookieToken != "" {
			tokenStr = cookieToken
		} else {
			// Fallback to Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			c.Error(errorcode.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := jwtutils.ValidateToken([]byte(m.cfg.Jwt.Secret), tokenStr, jwttype.AccessToken)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		c.Set(contextutils.CtxUserId, claims.Subject)
		c.Set(contextutils.CtxEmail, claims.Email)
		c.Set(contextutils.CtxRoleSlug, claims.RoleSlug)
		c.Set(contextutils.CtxAccessToken, tokenStr)

		c.Next()
	}
}

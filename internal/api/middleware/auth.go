package middleware

import (
	"smart-wardrobe-be/config"
	identity_security "smart-wardrobe-be/internal/modules/identity/application/interface/security"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/application/constants/jwttype"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/jwtutils"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthMiddleware struct {
	cfg                   *config.Config
	tokenBlacklistService identity_security.ITokenBlacklistService
	userRepo              identity_repos.IUserRepository
}

func NewAuthMiddleware(
	cfg *config.Config,
	tokenBlacklistService identity_security.ITokenBlacklistService,
	userRepo identity_repos.IUserRepository,
) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:                   cfg,
		tokenBlacklistService: tokenBlacklistService,
		userRepo:              userRepo,
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

		isBlacklisted, err := m.tokenBlacklistService.IsTokenBlacklisted(c.Request.Context(), tokenStr)
		if err != nil {
			c.Error(errorcode.NewInternalError("Không thể kiểm tra trạng thái phiên đăng nhập."))
			c.Abort()
			return
		}
		if isBlacklisted {
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

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			c.Error(errorcode.ErrInvalidAccessToken)
			c.Abort()
			return
		}

		user, err := m.userRepo.GetByID(c.Request.Context(), userID)
		if err != nil {
			c.Error(errorcode.NewInternalError("Không thể tải thông tin người dùng."))
			c.Abort()
			return
		}
		if user == nil || user.IsDeleted || user.Status != userstatus.Active {
			c.Error(errorcode.ErrUnauthorized)
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

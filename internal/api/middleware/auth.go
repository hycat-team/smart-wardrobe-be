package middleware

import (
	"strings"

	"smart-wardrobe-be/config"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	identity_security "smart-wardrobe-be/internal/modules/identity/application/interface/security"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/application/constants/jwttype"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/jwtutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthMiddleware struct {
	cfg                   *config.Config
	tokenBlacklistService identity_security.ITokenBlacklistService
}

func NewAuthMiddleware(
	cfg *config.Config,
	tokenBlacklistService identity_security.ITokenBlacklistService,
) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:                   cfg,
		tokenBlacklistService: tokenBlacklistService,
	}
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := m.authenticate(c, true); err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) OptionalHandle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := m.authenticate(c, false); err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) authenticate(c *gin.Context, required bool) error {
	tokenStr := m.extractToken(c)
	if tokenStr == "" {
		if required {
			return apperror.ErrUnauthorized()
		}
		return nil
	}

	claims, err := jwtutils.ValidateToken([]byte(m.cfg.Jwt.Secret), tokenStr, jwttype.AccessToken)
	if err != nil {
		if required {
			return err
		}
		return nil
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		if required {
			return apperror.ErrInvalidAccessToken()
		}
		return nil
	}

	tokenBlacklisted, userBlacklisted, err := m.tokenBlacklistService.CheckBlacklist(c.Request.Context(), tokenStr, userID)
	if err != nil {
		if required {
			return apperror.NewInternalError("Không thể xác thực trạng thái đăng nhập của bạn.")
		}
		return nil
	}

	if tokenBlacklisted {
		if required {
			return apperror.NewUnauthorized("Phiên làm việc không hợp lệ hoặc đã đăng xuất. Vui lòng đăng nhập lại.")
		}
		return nil
	}

	if userBlacklisted {
		if required {
			return identityerrors.ErrAccountDisabledAuth()
		}
		return nil
	}

	c.Set(contextutils.CtxUserId, claims.Subject)
	c.Set(contextutils.CtxEmail, claims.Email)
	c.Set(contextutils.CtxRoleSlug, claims.RoleSlug)
	c.Set(contextutils.CtxAccessToken, tokenStr)

	return nil
}

func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	if cookieToken, err := c.Cookie(contextutils.CookieAccessToken); err == nil && cookieToken != "" {
		return cookieToken
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}

package middleware

import (
	"smart-wardrobe-be/config"
	identity_security "smart-wardrobe-be/internal/modules/identity/application/interface/security"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
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

	isBlacklisted, err := m.tokenBlacklistService.IsTokenBlacklisted(c.Request.Context(), tokenStr)
	if err != nil {
		if required {
			return apperror.NewInternalError("Không thể kiểm tra trạng thái phiên đăng nhập.")
		}
		return nil
	}
	if isBlacklisted {
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

	user, err := m.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if required {
			return apperror.NewInternalError("Không thể tải thông tin người dùng.")
		}
		return nil
	}
	if user == nil || user.IsDeleted || user.Status != userstatus.Active {
		if required {
			return apperror.ErrUnauthorized()
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

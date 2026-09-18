package middleware

import (
	"crypto/ecdsa"
	"errors"
	"strings"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/application/constants/jwttype"
	sharedsecurity "smart-wardrobe-be/shared/application/interface/security"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/jwtutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware verifies the Closy JWT (Authorization bearer or access-token
// cookie) and consults the shared revocation store. It is service-agnostic:
// every service constructs it from shared config + the shared Redis blacklist
// (FR-022/FR-023, T047).
type AuthMiddleware struct {
	cfg                   *config.Config
	tokenBlacklistService sharedsecurity.ITokenBlacklistService
	useES256              bool
	es256Pub              *ecdsa.PublicKey
	keyErr                error
}

// NewAuthMiddleware builds the shared auth middleware. ES256 public-key
// verification with HS256 legacy fallback mirrors the account behaviour; a
// broken ES256 setup fails closed at request time.
func NewAuthMiddleware(
	cfg *config.Config,
	tokenBlacklistService sharedsecurity.ITokenBlacklistService,
) *AuthMiddleware {
	m := &AuthMiddleware{
		cfg:                   cfg,
		tokenBlacklistService: tokenBlacklistService,
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Jwt.Algorithm), "ES256") {
		m.useES256 = true
		if cfg.Jwt.PublicKey == "" {
			m.keyErr = errors.New("ES256 requires jwt.public_key")
		} else if pub, err := jwtutils.LoadECDSAPublicKey(cfg.Jwt.PublicKey); err != nil {
			m.keyErr = err
		} else {
			m.es256Pub = pub
		}
	}
	return m
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

	if m.useES256 && m.keyErr != nil {
		if required {
			return apperror.NewInternalError("Không thể xác thực phiên làm việc của bạn.")
		}
		return nil
	}

	claims, err := jwtutils.ValidateTokenFlexible(tokenStr, jwttype.AccessToken, jwtutils.VerifyOptions{
		Algorithm: m.cfg.Jwt.Algorithm,
		PublicKey: m.es256Pub,
		Secret:    []byte(m.cfg.Jwt.Secret),
	})
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
			return apperror.NewUnauthorized("Tài khoản đã bị khóa hoặc vô hiệu hóa. Vui lòng liên hệ CSKH.")
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

	if queryToken := c.Query("token"); queryToken != "" {
		return queryToken
	}

	return ""
}

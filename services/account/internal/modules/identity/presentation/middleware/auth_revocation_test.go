package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// flagBlacklistService simulates Redis-backed session revocation (FR-023)
// with controllable token/user blacklist flags.
type flagBlacklistService struct {
	tokenBlacklisted bool
	userBlacklisted  bool
}

func (f *flagBlacklistService) CheckBlacklist(ctx context.Context, token string, userID uuid.UUID) (bool, bool, error) {
	return f.tokenBlacklisted, f.userBlacklisted, nil
}
func (f *flagBlacklistService) BlacklistToken(ctx context.Context, token string, expiry time.Duration) error {
	return nil
}
func (f *flagBlacklistService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return f.tokenBlacklisted, nil
}
func (f *flagBlacklistService) BlacklistTokenWithPrefix(ctx context.Context, token string, prefix string, expiry time.Duration) error {
	return nil
}
func (f *flagBlacklistService) IsTokenBlacklistedWithPrefix(ctx context.Context, token string, prefix string) (bool, error) {
	return f.tokenBlacklisted, nil
}
func (f *flagBlacklistService) BlacklistUser(ctx context.Context, userID uuid.UUID, expiry time.Duration) error {
	return nil
}
func (f *flagBlacklistService) IsUserBlacklisted(ctx context.Context, userID uuid.UUID) (bool, error) {
	return f.userBlacklisted, nil
}
func (f *flagBlacklistService) UnblacklistUser(ctx context.Context, userID uuid.UUID) error {
	return nil
}

// TestRevokedTokenIsBlocked verifies that a token flagged in the revocation
// store is rejected even though its signature is valid (T048, FR-023).
func TestRevokedTokenIsBlocked(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.User))

	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	auth := shared_middleware.NewAuthMiddleware(&cfg, &flagBlacklistService{tokenBlacklisted: true})
	r.Use(auth.Handle())
	r.Use(shared_middleware.RolesAuthorize(roleslug.Admin, roleslug.User))
	r.GET("/probe", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if reached {
		t.Fatal("expected revoked token to be blocked before handler")
	}
}

// TestRevokedUserIsBlocked verifies that all sessions of a revoked user are
// rejected (T048, FR-023).
func TestRevokedUserIsBlocked(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.User))

	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	auth := shared_middleware.NewAuthMiddleware(&cfg, &flagBlacklistService{userBlacklisted: true})
	r.Use(auth.Handle())
	r.Use(shared_middleware.RolesAuthorize(roleslug.Admin, roleslug.User))
	r.GET("/probe", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if reached {
		t.Fatal("expected revoked user session to be blocked before handler")
	}
}

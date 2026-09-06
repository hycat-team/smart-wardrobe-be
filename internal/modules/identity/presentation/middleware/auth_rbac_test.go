package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/identity/presentation/middleware"
	jwtutils "smart-wardrobe-be/internal/pkg/utils/jwtutils"
	"smart-wardrobe-be/internal/shared/application/constants/jwttype"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeTokenBlacklistService struct{}

func (f *fakeTokenBlacklistService) CheckBlacklist(ctx context.Context, token string, userID uuid.UUID) (bool, bool, error) {
	return false, false, nil
}
func (f *fakeTokenBlacklistService) BlacklistToken(ctx context.Context, token string, expiry time.Duration) error {
	return nil
}
func (f *fakeTokenBlacklistService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return false, nil
}
func (f *fakeTokenBlacklistService) BlacklistTokenWithPrefix(ctx context.Context, token string, prefix string, expiry time.Duration) error {
	return nil
}
func (f *fakeTokenBlacklistService) IsTokenBlacklistedWithPrefix(ctx context.Context, token string, prefix string) (bool, error) {
	return false, nil
}
func (f *fakeTokenBlacklistService) BlacklistUser(ctx context.Context, userID uuid.UUID, expiry time.Duration) error {
	return nil
}
func (f *fakeTokenBlacklistService) IsUserBlacklisted(ctx context.Context, userID uuid.UUID) (bool, error) {
	return false, nil
}

func authTestConfig() config.Config {
	cfg := config.Config{}
	cfg.Jwt.Secret = "test-secret-key-for-rbac-matrix"
	cfg.Jwt.Issuer = "smart-wardrobe-test"
	cfg.Jwt.Audience = "smart-wardrobe-test"
	return cfg
}

func signedAccessToken(t *testing.T, cfg config.Config, userID uuid.UUID, roleSlug string) string {
	t.Helper()
	token, err := jwtutils.GenerateToken(userID, "user@example.com", roleSlug, jwttype.AccessToken,
		cfg.Jwt.Secret, cfg.Jwt.Issuer, cfg.Jwt.Audience, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

// rbacRouter wires AuthMiddleware.Handle + RolesAuthorize with a terminal probe
// that records whether the request reached the protected handler.
func rbacRouter(t *testing.T, cfg config.Config, allowed ...roleslug.RoleSlug) (*gin.Engine, *bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	auth := middleware.NewAuthMiddleware(&cfg, &fakeTokenBlacklistService{})
	r.Use(auth.Handle())
	r.Use(shared_middleware.RolesAuthorize(allowed...))
	r.GET("/probe", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	return r, &reached
}

func doAuthRequest(r *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthHandleRejectsMissingToken(t *testing.T) {
	cfg := authTestConfig()
	r, reached := rbacRouter(t, cfg, roleslug.Admin, roleslug.User)

	doAuthRequest(r, "")
	if *reached {
		t.Fatal("expected unauthenticated request to be blocked before handler")
	}
}

func TestAuthHandleRejectsGarbageToken(t *testing.T) {
	cfg := authTestConfig()
	r, reached := rbacRouter(t, cfg, roleslug.Admin, roleslug.User)

	doAuthRequest(r, "not-a-jwt")
	if *reached {
		t.Fatal("expected garbage token to be blocked")
	}
}

func TestRolesAuthorizeAllowsUserIntoUserRoutes(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.User))
	r, reached := rbacRouter(t, cfg, roleslug.Admin, roleslug.User)

	w := doAuthRequest(r, token)
	if !*reached {
		t.Fatalf("expected user token to reach user routes, got status %d", w.Code)
	}
}

func TestRolesAuthorizeBlocksUserFromAdminOnlyRoutes(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.User))
	r, reached := rbacRouter(t, cfg, roleslug.Admin)

	doAuthRequest(r, token)
	if *reached {
		t.Fatal("expected user token to be blocked from admin-only routes")
	}
}

func TestRolesAuthorizeAllowsAdminIntoAdminOnlyRoutes(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.Admin))
	r, reached := rbacRouter(t, cfg, roleslug.Admin)

	w := doAuthRequest(r, token)
	if !*reached {
		t.Fatalf("expected admin token to reach admin routes, got status %d", w.Code)
	}
}

func (f *fakeTokenBlacklistService) UnblacklistUser(ctx context.Context, userID uuid.UUID) error {
	return nil
}

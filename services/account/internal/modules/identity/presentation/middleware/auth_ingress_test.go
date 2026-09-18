package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestIngressDoesNotRequireGatewaySecret verifies that application auth relies
// solely on the Closy JWT: a request carrying a valid Authorization bearer and
// no gateway header must reach the handler (T045, FR-033).
func TestIngressDoesNotRequireGatewaySecret(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.User))
	r, reached := rbacRouter(t, cfg, roleslug.Admin, roleslug.User)

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !*reached {
		t.Fatalf("expected request without gateway secret to reach handler, got status %d", w.Code)
	}
}

// TestGatewaySecretAloneIsNotAccepted verifies that presenting only a gateway
// header (without a Closy JWT) is rejected: the service never treats the
// gateway header as application auth.
func TestGatewaySecretAloneIsNotAccepted(t *testing.T) {
	cfg := authTestConfig()
	r, reached := rbacRouter(t, cfg, roleslug.Admin, roleslug.User)

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-Gateway-Secret", "opaque-gateway-value")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if *reached {
		t.Fatal("expected gateway-secret-only request to be blocked before handler")
	}
}

// TestAuthorizationPreservedThroughMiddleware verifies that the original Closy
// JWT is handed downstream intact (T045, FR-033).
func TestAuthorizationPreservedThroughMiddleware(t *testing.T) {
	cfg := authTestConfig()
	token := signedAccessToken(t, cfg, uuid.New(), string(roleslug.User))

	gin.SetMode(gin.TestMode)
	var seen string
	r := gin.New()
	auth := shared_middleware.NewAuthMiddleware(&cfg, &fakeTokenBlacklistService{})
	r.Use(auth.Handle())
	r.GET("/probe", func(c *gin.Context) {
		seen = c.GetString(contextutils.CtxAccessToken)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if seen != token {
		t.Fatal("expected downstream to see the original bearer token")
	}
}

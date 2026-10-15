package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dto "smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	identity_handler "smart-wardrobe-be/services/account/internal/modules/identity/presentation/handler"
	identity_router "smart-wardrobe-be/services/account/internal/modules/identity/presentation/router"
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 029 T009: the HTTP wiring of GET /api/v1/admin/activity-events (FR-001,
// FR-002, FR-005).
//
// These go through the REAL router — `router.NewRouter(...).Init(...)` — not a
// test-local group. A test that registered the route itself would pass against
// a router that never registered it, and the whole point of this file is the
// wiring: the route carries auth + admin role, it answers only GET, and the
// query reaches the use case as typed fields.
//
// WHAT IT DOES NOT TEST: the rules (UUID, activity type, period). Those live in
// the use case and are pinned in activity_uc_test.go; only their passthrough is
// asserted here.

// stubActivityUseCase satisfies the whole user use case interface by embedding
// it, so only the method under test needs behaviour. Anything else panics
// through the embedded nil interface, which fails the test rather than
// silently answering zero values.
type stubActivityUseCase struct {
	uc_interfaces.IUserUseCase

	calls     int
	lastQuery dto.GetActivityEventsQueryReq
	result    *dto.ActivityEventListRes
	err       error
}

func (s *stubActivityUseCase) ListActivityEvents(
	_ context.Context, query dto.GetActivityEventsQueryReq,
) (*dto.ActivityEventListRes, error) {
	s.calls++
	s.lastQuery = query
	return s.result, s.err
}

// stubAuthMiddleware stands in for the identity auth middleware. role == nil
// models a request with no valid JWT: no context keys are written and
// RolesAuthorize must refuse it.
type stubAuthMiddleware struct {
	role *roleslug.RoleSlug
}

func (m stubAuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.role != nil {
			c.Set("userId", uuid.New())
			c.Set("roleSlug", string(*m.role))
		}
		c.Next()
	}
}

func (m stubAuthMiddleware) OptionalHandle() gin.HandlerFunc { return m.Handle() }

// allowAllRateLimiter keeps the login shield out of the way; the endpoint under
// test is not rate limited.
type allowAllRateLimiter struct{}

func (allowAllRateLimiter) Allow(context.Context, string, int, int) (bool, error) { return true, nil }

var _ shared_caching.IRateLimiter = allowAllRateLimiter{}

func activityEngine(t *testing.T, stub *stubActivityUseCase, role *roleslug.RoleSlug) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(logger.New("test", "", "info", false), "test"))

	loginLimit := shared_middleware.NewLoginRateLimitMiddleware(&sharedconfig.Config{
		ClaimRateLimit: sharedconfig.ClaimRateLimit{UserLimit: 60},
		RateLimit:      sharedconfig.RateLimit{ReplenishmentSeconds: 60},
	}, allowAllRateLimiter{})

	adminHandler := identity_handler.NewAdminHandler(stub, nil)
	// The auth and me handlers are unused by these routes; a method value on a
	// nil receiver is never invoked at registration.
	r := identity_router.NewRouter(nil, nil, adminHandler, stubAuthMiddleware{role: role}, loginLimit)
	r.Init(engine.Group("/api/v1"))
	return engine
}

func activityRequest(t *testing.T, engine *gin.Engine, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func activityResponse(t *testing.T, rec *httptest.ResponseRecorder) struct {
	Items    []map[string]any `json:"items"`
	Metadata map[string]any   `json:"metadata"`
} {
	t.Helper()
	var parsed struct {
		Data struct {
			Items    []map[string]any `json:"items"`
			Metadata map[string]any   `json:"metadata"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parsed))
	return parsed.Data
}

const activityPath = "/api/v1/admin/activity-events"

func TestActivityEventsRouteRequiresAdmin(t *testing.T) {
	userRole := roleslug.User
	adminRole := roleslug.Admin

	t.Run("no valid token is 401 and never reaches the use case", func(t *testing.T) {
		stub := &stubActivityUseCase{}
		engine := activityEngine(t, stub, nil)

		rec := activityRequest(t, engine, http.MethodGet, activityPath)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		assert.Zero(t, stub.calls)
	})

	t.Run("a normal account is 403 and never reaches the use case", func(t *testing.T) {
		stub := &stubActivityUseCase{}
		engine := activityEngine(t, stub, &userRole)

		rec := activityRequest(t, engine, http.MethodGet, activityPath)

		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		assert.Zero(t, stub.calls)
	})

	t.Run("an admin reaches the use case", func(t *testing.T) {
		stub := &stubActivityUseCase{result: &dto.ActivityEventListRes{}}
		engine := activityEngine(t, stub, &adminRole)

		rec := activityRequest(t, engine, http.MethodGet, activityPath)

		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, 1, stub.calls)
	})
}

func TestActivityEventsResponseShape(t *testing.T) {
	adminRole := roleslug.Admin
	occurred := time.Date(2026, 10, 15, 8, 31, 2, 0, timeutils.VnLoc())
	first := "An"
	last := "Nguyễn"

	stub := &stubActivityUseCase{result: &dto.ActivityEventListRes{
		Items: []*dto.ActivityEventRes{{
			ID:            uuid.New(),
			UserID:        uuid.New(),
			ActivityType:  "login",
			OccurredAt:    occurred,
			CreatedAt:     occurred,
			UserEmail:     "user@example.com",
			UserFirstName: &first,
			UserLastName:  &last,
		}},
		Metadata: shared_dto.PaginationMetadata{Page: 1, Limit: 20, TotalItems: 1, TotalPages: 1},
	}}
	engine := activityEngine(t, stub, &adminRole)

	rec := activityRequest(t, engine, http.MethodGet, activityPath)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := activityResponse(t, rec)
	require.Len(t, data.Items, 1)

	item := data.Items[0]
	expectedKeys := []string{
		"id", "userId", "activityType", "occurredAt", "createdAt",
		"userEmail", "userFirstName", "userLastName",
	}
	assert.Len(t, item, len(expectedKeys), "the item must carry exactly the contract fields, got %v", item)
	for _, key := range expectedKeys {
		assert.Contains(t, item, key, "the item must carry %q", key)
	}
	// camelCase only: the Go names must never leak into the contract.
	for _, leaked := range []string{"ID", "UserId", "UserID", "ActivityType", "OccurredAt", "CreatedAt", "UserEmail"} {
		assert.NotContains(t, item, leaked)
	}
	assert.Equal(t, "login", item["activityType"])
	assert.Equal(t, "user@example.com", item["userEmail"])
	assert.Equal(t, "An", item["userFirstName"])
	assert.Equal(t, "Nguyễn", item["userLastName"])

	occurredAt, ok := item["occurredAt"].(string)
	require.True(t, ok, "occurredAt must be a string, got %T", item["occurredAt"])
	assert.Contains(t, occurredAt, "+07:00", "activity instants are shown in Vietnam time")

	require.Contains(t, data.Metadata, "page")
	require.Contains(t, data.Metadata, "limit")
	require.Contains(t, data.Metadata, "totalItems")
	require.Contains(t, data.Metadata, "totalPages")
}

func TestActivityEventsQueryReachesTheUseCase(t *testing.T) {
	adminRole := roleslug.Admin
	stub := &stubActivityUseCase{result: &dto.ActivityEventListRes{}}
	engine := activityEngine(t, stub, &adminRole)

	userID := uuid.New()
	rec := activityRequest(t, engine, http.MethodGet,
		activityPath+"?userId="+userID.String()+"&activityType=refresh"+
			"&fromDate=2026-10-15&toDate=2026-10-15&page=2&limit=50")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, 1, stub.calls)
	assert.Equal(t, userID.String(), stub.lastQuery.UserID)
	assert.Equal(t, "refresh", stub.lastQuery.ActivityType)
	assert.Equal(t, "2026-10-15", stub.lastQuery.FromDate)
	assert.Equal(t, "2026-10-15", stub.lastQuery.ToDate)
	assert.Equal(t, 2, stub.lastQuery.Page)
	assert.Equal(t, 50, stub.lastQuery.Limit)
}

func TestActivityEventsValidationFailureIsSurfaced(t *testing.T) {
	adminRole := roleslug.Admin
	stub := &stubActivityUseCase{err: apperror.NewBadRequest("Loại hoạt động (activityType) không hợp lệ.")}
	engine := activityEngine(t, stub, &adminRole)

	rec := activityRequest(t, engine, http.MethodGet, activityPath+"?activityType=logout")

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var parsed struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parsed))
	assert.Equal(t, "Loại hoạt động (activityType) không hợp lệ.", parsed.Message)
}

// TestActivityEventsRouteIsReadOnly is FR-005 over HTTP: the group registers
// GET and nothing else, so every write verb is a 404 — not a route that exists
// and refuses, but no route at all. The table stays append-only through the
// activity write paths (session issuance), never through this API.
func TestActivityEventsRouteIsReadOnly(t *testing.T) {
	adminRole := roleslug.Admin
	stub := &stubActivityUseCase{}
	engine := activityEngine(t, stub, &adminRole)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rec := activityRequest(t, engine, method, activityPath)
		assert.Equal(t, http.StatusNotFound, rec.Code,
			"%s must be unregistered (read-only endpoint), got %d", method, rec.Code)
	}
	assert.Zero(t, stub.calls, "no write verb may reach the use case")
}

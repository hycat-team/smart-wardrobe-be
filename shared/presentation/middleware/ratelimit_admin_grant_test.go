package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// countingLimiter behaves like a fixed-window limiter: it allows up to `limit`
// uses of a key per window, then denies. Keys are tracked separately, which is
// what lets a test prove the rule is counted PER ADMINISTRATOR rather than per
// address.
type countingLimiter struct {
	used     map[string]int
	calls    []string
	failWith error
}

func (l *countingLimiter) Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	l.calls = append(l.calls, key)
	if l.failWith != nil {
		return false, l.failWith
	}
	if l.used == nil {
		l.used = map[string]int{}
	}
	l.used[key]++
	return l.used[key] <= limit, nil
}

func adminGrantTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.AdminGrantRateLimit.AllPerMinute = 60
	cfg.AdminGrantRateLimit.MultiPerMinute = 6
	cfg.AdminGrantRateLimit.WindowSeconds = 60
	return cfg
}

// newAdminGrantRouter mounts the limiter in the position it occupies in
// production: after authentication, with the administrator already in the context.
//
// GlobalErrorHandler is included because that is what turns a limiter's c.Error
// into a 429 response. Without it the middleware would abort correctly but the
// recorder would still report 200, and the test would be asserting on the test
// harness rather than on the behaviour.
func newAdminGrantRouter(cfg *config.Config, limiter *countingLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GlobalErrorHandler(&testLogger{}, "test"))
	router.Use(func(c *gin.Context) {
		if id := c.GetHeader("X-Test-Admin"); id != "" {
			if parsed, err := uuid.Parse(id); err == nil {
				c.Set(contextutils.CtxUserId, parsed)
			}
		}
		c.Next()
	})
	router.Use(NewAdminGrantRateLimitMiddleware(cfg, limiter).Handle())
	router.POST("/admin/subscription-grants", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func adminGrantRequest(t *testing.T, router *gin.Engine, adminID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/subscription-grants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Admin", adminID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func bulkBody(n int) string {
	items := make([]string, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, `{"userIdentifier":"u@example.com","planSlug":"premium-monthly"}`)
	}
	return `{"items":[` + strings.Join(items, ",") + `]}`
}

// FR-078: 60 requests per minute for everything, counted per administrator.
func TestAdminGrantRateLimit_AllTierAllowsSixtyPerMinute(t *testing.T) {
	limiter := &countingLimiter{}
	router := newAdminGrantRouter(adminGrantTestConfig(), limiter)
	admin := uuid.New().String()

	for i := 1; i <= 60; i++ {
		rec := adminGrantRequest(t, router, admin, bulkBody(1))
		if rec.Code != http.StatusOK {
			t.Fatalf("yêu cầu %d phải được cho qua, có = %d", i, rec.Code)
		}
	}
	rec := adminGrantRequest(t, router, admin, bulkBody(1))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("yêu cầu 61 phải bị chặn, có = %d", rec.Code)
	}
}

// FR-078: the bulk tier allows 6 multi-item requests per minute, on top of the
// general tier. A stricter single number could not express both rules.
func TestAdminGrantRateLimit_MultiTierAllowsSixBulkRequestsPerMinute(t *testing.T) {
	limiter := &countingLimiter{}
	cfg := adminGrantTestConfig()
	cfg.AdminGrantRateLimit.AllPerMinute = 500 // keep the general tier out of the way
	router := newAdminGrantRouter(cfg, limiter)
	admin := uuid.New().String()

	for i := 1; i <= 6; i++ {
		rec := adminGrantRequest(t, router, admin, bulkBody(2))
		if rec.Code != http.StatusOK {
			t.Fatalf("lượt hàng loạt %d phải được cho qua, có = %d", i, rec.Code)
		}
	}
	rec := adminGrantRequest(t, router, admin, bulkBody(2))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("lượt hàng loạt thứ 7 phải bị chặn, có = %d", rec.Code)
	}
}

// FR-078/SC-020: counted PER ADMINISTRATOR, not per IP. Two administrators
// behind one office address each get their own budget.
func TestAdminGrantRateLimit_CountedPerAdministratorNotPerIP(t *testing.T) {
	limiter := &countingLimiter{}
	cfg := adminGrantTestConfig()
	cfg.AdminGrantRateLimit.AllPerMinute = 2
	router := newAdminGrantRouter(cfg, limiter)
	first := uuid.New().String()
	second := uuid.New().String()

	for i := 0; i < 2; i++ {
		adminGrantRequest(t, router, first, bulkBody(1))
	}
	if rec := adminGrantRequest(t, router, first, bulkBody(1)); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("quản trị viên đầu phải bị chặn sau khi hết hạn mức, có = %d", rec.Code)
	}
	// Same test host, same RemoteAddr, different administrator: must still pass.
	for i := 0; i < 2; i++ {
		if rec := adminGrantRequest(t, router, second, bulkBody(1)); rec.Code != http.StatusOK {
			t.Fatalf("quản trị viên thứ hai phải có hạn mức riêng, yêu cầu %d có = %d", i+1, rec.Code)
		}
	}

	allKey := "rl:admin-grant:" + first + ":all"
	multiKey := "rl:admin-grant:" + first + ":multi"
	// No key may reference the network address: a per-IP rule would let one
	// administrator exhaust everyone else's budget from a shared office
	// connection.
	for _, key := range limiter.calls {
		if !strings.HasPrefix(key, "rl:admin-grant:") {
			t.Fatalf("tiền tố khoá phải là rl:admin-grant:, có = %q", key)
		}
		if strings.Contains(key, "127.0.0.1") || strings.Contains(key, "192.0.2") {
			t.Fatalf("khoá đếm không được dùng địa chỉ mạng, có = %q", key)
		}
	}
	if !containsKey(limiter.calls, allKey) {
		t.Errorf("phải dùng khoá mức chung %q", allKey)
	}
	if containsKey(limiter.calls, multiKey) {
		t.Error("mức hàng loạt không được dùng cho yêu cầu một phần tử")
	}
}

// FR-078: the bulk tier must NOT be charged for a single-item request, otherwise
// an administrator doing one-at-a-time work would exhaust the bulk budget.
func TestAdminGrantRateLimit_SingleItemDoesNotConsumeBulkTier(t *testing.T) {
	limiter := &countingLimiter{}
	cfg := adminGrantTestConfig()
	cfg.AdminGrantRateLimit.AllPerMinute = 20
	cfg.AdminGrantRateLimit.MultiPerMinute = 1
	router := newAdminGrantRouter(cfg, limiter)
	admin := uuid.New().String()

	for i := 1; i <= 10; i++ {
		if rec := adminGrantRequest(t, router, admin, bulkBody(1)); rec.Code != http.StatusOK {
			t.Fatalf("yêu cầu đơn %d phải được cho qua, có = %d", i, rec.Code)
		}
	}
	if containsKey(limiter.calls, "rl:admin-grant:"+admin+":multi") {
		t.Error("yêu cầu một phần tử không được tiêu mức hàng loạt")
	}
}

// SC-020: a request rejected by the limiter reaches NO handler, so no item is
// granted. The limit has to be in front of the work, not after it.
func TestAdminGrantRateLimit_RejectedRequestGrantsNothing(t *testing.T) {
	limiter := &countingLimiter{}
	cfg := adminGrantTestConfig()
	cfg.AdminGrantRateLimit.AllPerMinute = 1
	router := newAdminGrantRouter(cfg, limiter)
	admin := uuid.New().String()

	rec := adminGrantRequest(t, router, admin, bulkBody(3))
	if rec.Code != http.StatusOK {
		t.Fatalf("lượt đầu phải được cho qua, có = %d", rec.Code)
	}
	rec = adminGrantRequest(t, router, admin, bulkBody(3))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("lượt thứ hai phải bị chặn trước khi xử lý, có = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "premium") {
		t.Error("phản hồi chặn không được chứa kết quả cấp gói")
	}
}

// FR-079: the 429 tells the administrator how long to wait, in Vietnamese.
func TestAdminGrantRateLimit_TooManyRequestsStatesRetryDelayInVietnamese(t *testing.T) {
	limiter := &countingLimiter{}
	cfg := adminGrantTestConfig()
	cfg.AdminGrantRateLimit.AllPerMinute = 1
	cfg.AdminGrantRateLimit.WindowSeconds = 45
	router := newAdminGrantRouter(cfg, limiter)
	admin := uuid.New().String()

	adminGrantRequest(t, router, admin, bulkBody(1))
	rec := adminGrantRequest(t, router, admin, bulkBody(1))

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("phải trả 429, có = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "45") {
		t.Errorf("thông báo phải nêu số giây cần chờ, có = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "quá nhanh") {
		t.Errorf("thông báo phải viết bằng tiếng Việt, có = %s", rec.Body.String())
	}
}

// The limiter FAILS CLOSED. A grant writes up to 500 customer entitlements, so a
// limiter that cannot answer must not let the request through unbounded.
func TestAdminGrantRateLimit_FailsClosedWhenLimiterErrors(t *testing.T) {
	limiter := &countingLimiter{failWith: errors.New("redis down")}
	router := newAdminGrantRouter(adminGrantTestConfig(), limiter)

	rec := adminGrantRequest(t, router, uuid.New().String(), bulkBody(1))
	if rec.Code == http.StatusOK {
		t.Error("khi bộ đếm không trả lời thì phải chặn, không được cho qua")
	}
}

// An unauthenticated call is refused: without an identity there is no
// per-administrator bucket, and falling back to one shared bucket would let any
// anonymous caller exhaust everyone's budget.
func TestAdminGrantRateLimit_RefusesUnauthenticatedRequest(t *testing.T) {
	limiter := &countingLimiter{}
	router := newAdminGrantRouter(adminGrantTestConfig(), limiter)

	req := httptest.NewRequest(http.MethodPost, "/admin/subscription-grants", strings.NewReader(bulkBody(1)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("yêu cầu chưa xác thực phải bị từ chối")
	}
	if len(limiter.calls) != 0 {
		t.Errorf("không được tiêu hạn mức của ai cả khi chưa xác thực, có = %d lần", len(limiter.calls))
	}
}

// The body is read to decide the tier and then handed on: a handler that binds
// normally must still see the items.
func TestAdminGrantRateLimit_BodyRemainsReadableByHandler(t *testing.T) {
	limiter := &countingLimiter{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GlobalErrorHandler(&testLogger{}, "test"))
	router.Use(func(c *gin.Context) {
		c.Set(contextutils.CtxUserId, uuid.New())
		c.Next()
	})
	router.Use(NewAdminGrantRateLimitMiddleware(adminGrantTestConfig(), limiter).Handle())

	var seen int
	router.POST("/admin/subscription-grants", func(c *gin.Context) {
		var body struct {
			Items []map[string]string `json:"items"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		seen = len(body.Items)
		c.Status(http.StatusOK)
	})

	rec := adminGrantRequest(t, router, uuid.New().String(), bulkBody(4))
	if rec.Code != http.StatusOK {
		t.Fatalf("handler phải đọc được thân yêu cầu, có = %d", rec.Code)
	}
	if seen != 4 {
		t.Errorf("handler phải thấy đủ 4 phần tử, có = %d", seen)
	}
}

func containsKey(keys []string, want string) bool {
	for _, key := range keys {
		if key == want {
			return true
		}
	}
	return false
}

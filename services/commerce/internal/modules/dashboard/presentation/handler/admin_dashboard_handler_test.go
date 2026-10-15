package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 029 T029 / FR-011, FR-014, FR-015 — the admin dashboard period parameters
// over HTTP.
//
// WHY THESE GO THROUGH THE ROUTER. The before→after this task pins is that a
// bad date used to be parsed per-handler with `time.Parse` and, on the three
// older endpoints, a malformed value was refused but an inverted range was
// silently accepted; the four newer endpoints took no date at all. What is
// tested here is the wiring: the shared helper turns every malformed or
// inverted pair into a 400 BEFORE the use case runs, a valid pair reaches the
// use case as the VN half-open [from, to) instants, and the endpoints without
// a period keep their behaviour.
//
// The use case is stubbed: the queries themselves are covered elsewhere (the
// commerce use-case tests for defaults/formulas, the account/AI repo tests for
// the SQL).

const adminDashboardBase = "/api/v1/admin/dashboard"

// periodCall records what one period-carrying handler handed to the use case.
type periodCall struct {
	method string
	from   time.Time
	to     time.Time
}

// stubAdminDashboardUC records forwarded periods and counts the calls of the
// endpoints that take none.
type stubAdminDashboardUC struct {
	periodCalls   []periodCall
	kpiCalls      int
	overviewCalls int
}

func (s *stubAdminDashboardUC) record(method string, from, to time.Time) {
	s.periodCalls = append(s.periodCalls, periodCall{method: method, from: from, to: to})
}

func (s *stubAdminDashboardUC) callFor(t *testing.T, method string) periodCall {
	t.Helper()
	if len(s.periodCalls) != 1 {
		t.Fatalf("expected exactly one use-case call, got %d: %+v", len(s.periodCalls), s.periodCalls)
	}
	if s.periodCalls[0].method != method {
		t.Fatalf("the %s route reached the %s use-case method", method, s.periodCalls[0].method)
	}
	return s.periodCalls[0]
}

func (s *stubAdminDashboardUC) GetAdminKPICards(context.Context) (*dto.AdminKPICardsRes, error) {
	s.kpiCalls++
	return &dto.AdminKPICardsRes{}, nil
}

func (s *stubAdminDashboardUC) GetAIMarginAnalytics(_ context.Context, from, to time.Time) (*dto.AIMarginAnalyticsRes, error) {
	s.record("ai-margin", from, to)
	return &dto.AIMarginAnalyticsRes{}, nil
}

func (s *stubAdminDashboardUC) GetPayOSReconciliation(_ context.Context, from, to time.Time) (*dto.PayOSReconciliationRes, error) {
	s.record("payos-reconciliation", from, to)
	return &dto.PayOSReconciliationRes{}, nil
}

func (s *stubAdminDashboardUC) GetSystemOverview(context.Context) (*dto.AdminOverviewRes, error) {
	s.overviewCalls++
	return &dto.AdminOverviewRes{}, nil
}

func (s *stubAdminDashboardUC) GetSubscriptionDistribution(_ context.Context, from, to time.Time) (*dto.SubscriptionDistributionRes, error) {
	s.record("distribution", from, to)
	return &dto.SubscriptionDistributionRes{}, nil
}

func (s *stubAdminDashboardUC) GetRevenueBreakdown(_ context.Context, from, to time.Time) (*dto.RevenueBreakdownRes, error) {
	s.record("revenue-breakdown", from, to)
	return &dto.RevenueBreakdownRes{}, nil
}

func (s *stubAdminDashboardUC) GetAIUsageByOperation(_ context.Context, from, to time.Time) (*dto.AIUsageByOperationRes, error) {
	s.record("usage-by-operation", from, to)
	return &dto.AIUsageByOperationRes{}, nil
}

func (s *stubAdminDashboardUC) GetAIErrorBreakdown(_ context.Context, from, to time.Time) (*dto.AIErrorBreakdownRes, error) {
	s.record("error-breakdown", from, to)
	return &dto.AIErrorBreakdownRes{}, nil
}

func (s *stubAdminDashboardUC) GetRenewalStats(_ context.Context, from, to time.Time) (*dto.RenewalStatsRes, error) {
	s.record("renewal-stats", from, to)
	return &dto.RenewalStatsRes{}, nil
}

// periodRoutes is the FR-011 set: every metric endpoint that must accept a
// period, with the use-case method it must reach.
var periodRoutes = []struct {
	path   string
	method string
}{
	{"/subscriptions/revenue-breakdown", "revenue-breakdown"},
	{"/subscriptions/distribution", "distribution"},
	{"/subscriptions/renewal-stats", "renewal-stats"},
	{"/ai/error-breakdown", "error-breakdown"},
	{"/payos-reconciliation", "payos-reconciliation"},
	{"/ai-margin", "ai-margin"},
	{"/ai/usage-by-operation", "usage-by-operation"},
}

// authenticatedAsAdmin stands in for the auth middleware having accepted an
// administrator: it writes the two context keys RolesAuthorize reads.
func authenticatedAsAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", uuid.New())
		c.Set("roleSlug", string(roleslug.Admin))
		c.Next()
	}
}

// newAdminDashboardEngine builds the real route surface around the stub, with
// the production error handler and role check.
func newAdminDashboardEngine(t *testing.T, stub *stubAdminDashboardUC) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := logger.New("test", "", "info", false)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))

	h := NewAdminDashboardHandler(stub)
	group := engine.Group(adminDashboardBase)
	group.Use(authenticatedAsAdmin(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		group.GET("/kpi-cards", shared_pres.WrapHandler(h.GetKPICards))
		group.GET("/overview", shared_pres.WrapHandler(h.GetOverview))
		group.GET("/ai-margin", shared_pres.WrapHandler(h.GetAIMargin))
		group.GET("/payos-reconciliation", shared_pres.WrapHandler(h.GetPayOSReconciliation))
		group.GET("/subscriptions/distribution", shared_pres.WrapHandler(h.GetSubscriptionDistribution))
		group.GET("/subscriptions/revenue-breakdown", shared_pres.WrapHandler(h.GetRevenueBreakdown))
		group.GET("/ai/usage-by-operation", shared_pres.WrapHandler(h.GetAIUsageByOperation))
		group.GET("/ai/error-breakdown", shared_pres.WrapHandler(h.GetAIErrorBreakdown))
		group.GET("/subscriptions/renewal-stats", shared_pres.WrapHandler(h.GetRenewalStats))
	}
	return engine
}

// dashboardRequest sends one request through the engine.
func dashboardRequest(engine *gin.Engine, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// TestAdminDashboardPeriodReachesTheUseCase is the FR-014 half: a valid pair
// must reach the use case as [StartOfDayInVN, EndOfDayInVN) — the same
// instants the account/AI repos filter on.
func TestAdminDashboardPeriodReachesTheUseCase(t *testing.T) {
	for _, route := range periodRoutes {
		t.Run(route.path, func(t *testing.T) {
			stub := &stubAdminDashboardUC{}
			engine := newAdminDashboardEngine(t, stub)

			rec := dashboardRequest(engine, adminDashboardBase+route.path+"?fromDate=2026-10-01&toDate=2026-10-15")
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
			}

			call := stub.callFor(t, route.method)
			wantFrom := time.Date(2026, 10, 1, 0, 0, 0, 0, timeutils.VnLoc())
			wantTo := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())
			if !call.from.Equal(wantFrom) {
				t.Errorf("fromDate = %s, want %s (StartOfDayInVN)", call.from, wantFrom)
			}
			if !call.to.Equal(wantTo) {
				t.Errorf("toDate = %s, want %s (EndOfDayInVN — exclusive end of the VN day)", call.to, wantTo)
			}

			// The 23:30 VN boundary from the spec must fall inside [from, to).
			boundary := time.Date(2026, 10, 15, 23, 30, 0, 0, timeutils.VnLoc())
			if boundary.Before(call.from) || !boundary.Before(call.to) {
				t.Errorf("23:30 VN on the last selected day must fall inside [%s, %s)", call.from, call.to)
			}
		})
	}
}

// TestAdminDashboardPeriodRejectsMalformedOrInverted is FR-015: every period
// endpoint answers 400 with a Vietnamese message, before any use-case work.
func TestAdminDashboardPeriodRejectsMalformedOrInverted(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		needle string
	}{
		{"malformed fromDate", "?fromDate=15-10-2026", "fromDate"},
		{"malformed toDate", "?toDate=2026-10-32", "toDate"},
		{"inverted period", "?fromDate=2026-10-16&toDate=2026-10-15", "không được sau"},
	}

	for _, route := range periodRoutes {
		for _, tc := range cases {
			t.Run(route.path+"/"+tc.name, func(t *testing.T) {
				stub := &stubAdminDashboardUC{}
				engine := newAdminDashboardEngine(t, stub)

				rec := dashboardRequest(engine, adminDashboardBase+route.path+tc.query)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("FR-015: a period that cannot be honoured must be refused with 400, got %d (%s)",
						rec.Code, rec.Body.String())
				}
				if !strings.Contains(rec.Body.String(), tc.needle) {
					t.Errorf("the 400 message must mention %q, got %s", tc.needle, rec.Body.String())
				}
				if len(stub.periodCalls) != 0 {
					t.Errorf("the use case ran for a refused period: %+v", stub.periodCalls)
				}
			})
		}
	}
}

// TestAdminDashboardOneSidedPeriodStaysOpen is the data-model §3 single
// boundary: only fromDate sent leaves the upper side open (zero), and vice
// versa, for every period endpoint.
func TestAdminDashboardOneSidedPeriodStaysOpen(t *testing.T) {
	for _, route := range periodRoutes {
		t.Run(route.path, func(t *testing.T) {
			stub := &stubAdminDashboardUC{}
			engine := newAdminDashboardEngine(t, stub)

			rec := dashboardRequest(engine, adminDashboardBase+route.path+"?fromDate=2026-10-01")
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
			}
			call := stub.callFor(t, route.method)
			if !call.from.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, timeutils.VnLoc())) {
				t.Errorf("fromDate = %s, want StartOfDayInVN(2026-10-01)", call.from)
			}
			if !call.to.IsZero() {
				t.Errorf("an omitted toDate must stay zero (open upper side), got %s", call.to)
			}
		})
	}
}

// TestAdminDashboardEndpointsWithoutAPeriodAreUnchanged covers the other half
// of FR-011/SC-005: kpi-cards and overview take no period and must keep
// working with a stray date query present.
func TestAdminDashboardEndpointsWithoutAPeriodAreUnchanged(t *testing.T) {
	stub := &stubAdminDashboardUC{}
	engine := newAdminDashboardEngine(t, stub)

	rec := dashboardRequest(engine, adminDashboardBase+"/kpi-cards")
	if rec.Code != http.StatusOK {
		t.Fatalf("kpi-cards: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	// A date-looking query parameter must not turn the snapshot endpoint into
	// a 400; the period is simply not part of its contract.
	rec = dashboardRequest(engine, adminDashboardBase+"/kpi-cards?fromDate=not-a-date")
	if rec.Code != http.StatusOK {
		t.Fatalf("kpi-cards with a stray fromDate: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	rec = dashboardRequest(engine, adminDashboardBase+"/overview")
	if rec.Code != http.StatusOK {
		t.Fatalf("overview: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	if stub.kpiCalls != 2 {
		t.Errorf("expected kpi-cards to run twice, got %d", stub.kpiCalls)
	}
	if stub.overviewCalls != 1 {
		t.Errorf("expected overview to run once, got %d", stub.overviewCalls)
	}
	if len(stub.periodCalls) != 0 {
		t.Errorf("no period endpoint may run for the snapshot routes: %+v", stub.periodCalls)
	}
}

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 025 T074-T076c: the operator look-up over HTTP (FR-068 to FR-072, SC-011).
//
// WHY THESE GO THROUGH THE ROUTER RATHER THAN THE HANDLER. Half of them are not
// about the handler at all: a normal customer is refused with 403, an exhausted
// budget with 429, and both refusals happen in middleware that runs BEFORE the
// handler. Calling `h.GetAccountEligibility` directly would skip both, and every
// test below would pass against an endpoint that hands campaign data to any
// signed-in customer. The 024 grant tests made the same point the hard way
// (`admin_grant_batch_http_test.go`): the defect was never in the handler, it was
// in what ran before it.
//
// The rules themselves are tested at the use-case level, against the ledger they
// actually read (`campaign_query_eligibility_test.go`). What is tested HERE is the
// wiring: that the route carries the role check, that the limiter is the one the
// grant path uses, that a missing code is a 404 with a Vietnamese message rather
// than an empty list, and that fifty look-ups change nothing.

// campaignPath is the registered base path (contracts/campaign-query-api.md §2).
const campaignPath = "/api/v1/admin/campaigns"

// authenticatedAs stands in for the auth middleware having accepted a caller.
//
// It writes the two context keys the real middleware writes and the limiter reads.
// Modelling the AUTHENTICATED case rather than the middleware itself is
// deliberate: these tests are about what happens after authentication, and the
// authentication contract has its own tests.
func authenticatedAs(role roleslug.RoleSlug) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", uuid.New())
		c.Set("roleSlug", string(role))
		c.Next()
	}
}

// alwaysAllowRateLimiter permits every request, so a test that is not about the
// limit does not have to think about it.
type alwaysAllowRateLimiter struct{}

func (alwaysAllowRateLimiter) Allow(context.Context, string, int, int) (bool, error) {
	return true, nil
}

// neverAllowRateLimiter refuses every request: the moment an administrator has
// exhausted the per-minute budget. A limiter that allowed everything could not
// express that moment, so the 429 case would have nothing to assert.
type neverAllowRateLimiter struct{}

func (neverAllowRateLimiter) Allow(context.Context, string, int, int) (bool, error) {
	return false, nil
}

// campaignEngine builds the real route group around a stubbed use case.
//
// The limiter is the PRODUCTION middleware and its limits come from
// configuration rather than from literals, because FR-071's requirement is that
// the look-up obeys the same budget as the bulk grant — and a test using a
// hard-coded limit would pass without proving the two are the same number.
func campaignEngine(
	t *testing.T,
	stub *stubCampaignQuery,
	as gin.HandlerFunc,
	limiter *shared_middleware.AdminGrantRateLimitMiddleware,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := logger.New("test", "", "info", false)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))

	h := NewCampaignAdminHandler(log, stub)
	// FR-070: the role check lives on the GROUP, exactly as the router wires it.
	// Putting it here rather than in a test-local check is the point — a test that
	// checked the role itself would pass against a route that carries no check.
	group := engine.Group("/api/v1/admin")
	group.Use(as, shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		campaigns := group.Group("/campaigns")
		campaigns.GET("", limiter.Handle(), shared_pres.WrapHandler(h.ListCampaigns))
		campaigns.GET("/:code", limiter.Handle(), shared_pres.WrapHandler(h.GetCampaign))
		campaigns.GET("/:code/claims", limiter.Handle(), shared_pres.WrapHandler(h.ListClaims))
		campaigns.GET("/:code/eligibility/:userId", limiter.Handle(), shared_pres.WrapHandler(h.GetAccountEligibility))
	}
	return engine
}

// newCampaignEngine wires an administrator with a generous budget.
func newCampaignEngine(t *testing.T, stub *stubCampaignQuery) *gin.Engine {
	return campaignEngine(t, stub, authenticatedAs(roleslug.Admin), newAdminLimiter(alwaysAllowRateLimiter{}))
}

// newAdminLimiter builds the production limiter over a rate-limit backend.
func newAdminLimiter(backend shared_caching.IRateLimiter) *shared_middleware.AdminGrantRateLimitMiddleware {
	return shared_middleware.NewAdminGrantRateLimitMiddleware(
		&sharedconfig.Config{AdminGrantRateLimit: sharedconfig.AdminGrantRateLimit{
			AllPerMinute:   60,
			MultiPerMinute: 6,
			WindowSeconds:  60,
		}},
		backend,
	)
}

// ---- stub use case ----

// stubCampaignQuery records its calls and answers from fixed tables.
//
// A STUB rather than the real use case on purpose. FR-069 says the read path
// changes nothing, and the strongest available statement of that is visible in this
// signature: there is no repository to write through, no unit of work, no
// notification contract. So there is nothing here that COULD spend a slot, grant a
// plan or send an email, and the fifty-call test below is asserting a property of
// the SHAPE rather than of a well-behaved collaborator.
//
// The behaviour tests that need the real rules — the three different answers, the
// grant filter, the reversed range — live at the use-case level
// (`campaign_query_eligibility_test.go`), where they run against the ledger they
// actually read.
type stubCampaignQuery struct {
	calls int
	// verdicts answers GetAccountEligibility per account, so three accounts can be
	// given three different answers.
	verdicts map[uuid.UUID]*sub_dto.CampaignAccountEligibilityRes
	// summary answers ListCampaigns and GetCampaign.
	summary *sub_dto.CampaignSummaryRes
	// claims and claimsTotal answer ListClaims. The total must NOT change when the
	// window narrows — that is what "read-only" looks like from outside.
	claims      []*sub_dto.CampaignClaimRes
	claimsTotal int64
	// lastFilter is the filter the handler built, so a test can assert that `from`
	// and `to` arrived as INSTANTS rather than as strings.
	lastFilter repositories.CampaignClaimsFilter
}

func (s *stubCampaignQuery) ListCampaigns(
	ctx context.Context, page, limit int,
) ([]*sub_dto.CampaignSummaryRes, int64, error) {
	s.calls++
	if s.summary == nil {
		return []*sub_dto.CampaignSummaryRes{}, 0, nil
	}
	return []*sub_dto.CampaignSummaryRes{s.summary}, 1, nil
}

func (s *stubCampaignQuery) GetCampaign(
	ctx context.Context, campaignCode string,
) (*sub_dto.CampaignSummaryRes, error) {
	s.calls++
	if campaignCode != "launch-2026-10" {
		return nil, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}
	return s.summary, nil
}

func (s *stubCampaignQuery) ListClaims(
	ctx context.Context,
	campaignCode string,
	filter repositories.CampaignClaimsFilter,
	page, limit int,
) ([]*sub_dto.CampaignClaimRes, int64, error) {
	s.calls++
	if campaignCode != "launch-2026-10" {
		return nil, 0, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}
	// A reversed window is refused by the USE CASE, not by the handler — the rule
	// belongs beside the query it constrains so that a non-HTTP caller gets it
	// too. The stub mirrors that rather than re-deciding it, so an end-to-end
	// assertion here proves the rule is reached, and the rule itself is proved in
	// `TestClaimsFilterRefusesAReversedRange` at the use-case level.
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, 0, subscriptionerrors.ErrCampaignClaimWindowInvalid()
	}
	s.lastFilter = filter
	return s.claims, s.claimsTotal, nil
}

func (s *stubCampaignQuery) GetAccountEligibility(
	ctx context.Context, campaignCode string, userID uuid.UUID,
) (*sub_dto.CampaignAccountEligibilityRes, error) {
	s.calls++
	if campaignCode != "launch-2026-10" {
		return nil, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}
	if verdict, ok := s.verdicts[userID]; ok {
		return verdict, nil
	}
	return nil, subscriptionerrors.ErrCampaignAccountNotFound()
}

// ---- helpers ----

// errorMessage pulls the human-facing message out of an error response.
func campaignErrorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var parsed struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse error response %q: %v", rec.Body.String(), err)
	}
	if parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return parsed.Message
}

// eligibilityResponse pulls one verdict out of a success envelope.
func eligibilityResponse(t *testing.T, rec *httptest.ResponseRecorder) sub_dto.CampaignAccountEligibilityRes {
	t.Helper()
	var parsed struct {
		Data sub_dto.CampaignAccountEligibilityRes `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse response %q: %v", rec.Body.String(), err)
	}
	return parsed.Data
}

// claimsResponse pulls the paginated claims envelope.
func claimsResponse(t *testing.T, rec *httptest.ResponseRecorder) struct {
	Items    []sub_dto.CampaignClaimRes `json:"items"`
	Metadata struct {
		TotalItems int64 `json:"totalItems"`
	} `json:"metadata"`
} {
	t.Helper()
	var parsed struct {
		Data struct {
			Items    []sub_dto.CampaignClaimRes `json:"items"`
			Metadata struct {
				TotalItems int64 `json:"totalItems"`
			} `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse response %q: %v", rec.Body.String(), err)
	}
	return parsed.Data
}

// ---- T074: three accounts, three different answers (FR-068) ----

// TestCampaignEligibilityReturnsThreeDifferentAnswersOverHTTP is T074 / FR-068 /
// SC-011.
//
// FR-068 asks for the three states to be DISTINGUISHABLE, not for three correct
// strings. So the assertion is pairwise: after the three calls, the three verdicts
// must not repeat. A response that returned `not_eligible` for all three would
// satisfy every field-level check and still fail the requirement, and it is the
// shape of the original defect this phase exists to fix.
func TestCampaignEligibilityReturnsThreeDifferentAnswersOverHTTP(t *testing.T) {
	grantedID := uuid.New()
	exhaustedID := uuid.New()
	notEligibleID := uuid.New()

	stub := &stubCampaignQuery{verdicts: map[uuid.UUID]*sub_dto.CampaignAccountEligibilityRes{
		grantedID: {
			CampaignCode: "launch-2026-10",
			UserID:       grantedID,
			Eligibility:  sub_dto.CampaignEligibilityGranted,
			Granted:      true,
		},
		exhaustedID: {
			CampaignCode: "launch-2026-10",
			UserID:       exhaustedID,
			Eligibility:  sub_dto.CampaignEligibilityEligibleButExhausted,
			Granted:      false,
			Reason:       "quota_compensation",
		},
		notEligibleID: {
			CampaignCode: "launch-2026-10",
			UserID:       notEligibleID,
			Eligibility:  sub_dto.CampaignEligibilityNotEligible,
			Granted:      false,
			Reason:       "outside_window",
		},
	}}
	engine := newCampaignEngine(t, stub)

	seen := map[string]uuid.UUID{}
	for name, userID := range map[string]uuid.UUID{
		"granted":                grantedID,
		"eligible_but_exhausted": exhaustedID,
		"not_eligible":           notEligibleID,
	} {
		rec := request(t, engine, http.MethodGet,
			campaignPath+"/launch-2026-10/eligibility/"+userID.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d (%s)", name, rec.Code, rec.Body.String())
		}
		verdict := eligibilityResponse(t, rec)

		// `granted` and the eligibility must agree — they are written by the same
		// branch, and a client switching on one while reading the other must never
		// see them disagree.
		if (verdict.Eligibility == sub_dto.CampaignEligibilityGranted) != verdict.Granted {
			t.Errorf("%s: eligibility %q and granted=%v disagree", name, verdict.Eligibility, verdict.Granted)
		}
		if previous, clash := seen[verdict.Eligibility]; clash {
			t.Fatalf("FR-068: the accounts %s and %s both answered %q; the three states must be "+
				"DISTINGUISHABLE and this phase exists to make that so",
				previous, userID, verdict.Eligibility)
		}
		seen[verdict.Eligibility] = userID
	}

	if len(seen) != 3 {
		t.Fatalf("expected 3 distinct verdicts, got %d", len(seen))
	}
}

// ---- T075: read-only (FR-069) ----

// TestCampaignLookUpIsReadOnlyOverFiftyCalls is T075 / FR-069 / SC-011.
//
// The read-only guarantee has four parts (contracts/campaign-query-api.md §5) and
// three of them are about what does NOT happen. So the assertions are on
// invariants rather than on a return value:
//
//   - the SAME budget numbers every time, fifty calls over — a look-up that spent a
//     slot would move them;
//   - the SAME pagination total — a read that wrote a claim row would change it;
//   - NO notification dependency reachable at all, which is the only honest way to
//     assert "no email outbox row was produced" from a test: the look-up is not
//     wired to anything that could produce one.
func TestCampaignLookUpIsReadOnlyOverFiftyCalls(t *testing.T) {
	const quota, reserve = 50, 5
	stub := &stubCampaignQuery{
		summary: &sub_dto.CampaignSummaryRes{
			Code:                     "launch-2026-10",
			Quota:                    quota,
			Reserve:                  reserve,
			GrantedMainCount:         50,
			GrantedCompensationCount: 3,
			GrantedTotalCount:        53,
			RemainingMain:            0,
			RemainingCompensation:    2,
			HardCap:                  quota + reserve,
			Status:                   sub_dto.CampaignStatusCompensation,
		},
		claims:      []*sub_dto.CampaignClaimRes{{UserID: uuid.New(), PlanSlug: "premium-monthly", Source: sub_dto.CampaignClaimSourceMain}},
		claimsTotal: 53,
	}
	engine := newCampaignEngine(t, stub)

	first := request(t, engine, http.MethodGet, campaignPath)
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", first.Code, first.Body.String())
	}
	baseline := first.Body.String()

	for i := 0; i < 50; i++ {
		rec := request(t, engine, http.MethodGet, campaignPath)
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d (%s)", i, rec.Code, rec.Body.String())
		}
		// BYTE-IDENTICAL is the strongest available statement. A number that drifted
		// by one would change a character, so there is nowhere for a mutation to
		// hide — not in a field the test forgot to read.
		if rec.Body.String() != baseline {
			t.Fatalf("FR-069: call %d returned a different body from call 0.\n first: %s\n  call %d: %s",
				i, baseline, i, rec.Body.String())
		}
	}

	// The claims page too, since it is the other endpoint that reports budget
	// consumption and the one an operator refreshes.
	before := claimsResponse(t, request(t, engine, http.MethodGet, campaignPath+"/launch-2026-10/claims"))
	for i := 0; i < 50; i++ {
		request(t, engine, http.MethodGet, campaignPath+"/launch-2026-10/claims")
	}
	after := claimsResponse(t, request(t, engine, http.MethodGet, campaignPath+"/launch-2026-10/claims"))
	if before.Metadata.TotalItems != after.Metadata.TotalItems {
		t.Fatalf("FR-069: the claim total moved from %d to %d after 50 reads — a read that wrote "+
			"a claim row would do exactly that", before.Metadata.TotalItems, after.Metadata.TotalItems)
	}
	if len(before.Items) != len(after.Items) {
		t.Fatalf("FR-069: the page size changed from %d to %d after 50 reads",
			len(before.Items), len(after.Items))
	}
}

// ---- T076: unknown campaign code (FR-072) ----

// TestUnknownCampaignCodeIsRefusedNotEmpty is T076 / FR-072 / SC-011.
//
// FR-072's requirement has two halves and the second is the one that fails in
// practice: the response must be a FRIENDLY error naming the code, not a technical
// failure and not an empty list. An empty list reads as "this campaign granted
// nobody", which is a completely different fact and the one that sends an operator
// hunting for a budget fault that does not exist.
func TestUnknownCampaignCodeIsRefusedNotEmpty(t *testing.T) {
	const unknownCode = "khong-ton-tai"
	stub := &stubCampaignQuery{summary: &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"}}
	engine := newCampaignEngine(t, stub)

	for _, target := range []string{
		campaignPath + "/" + unknownCode,
		campaignPath + "/" + unknownCode + "/claims",
	} {
		rec := request(t, engine, http.MethodGet, target)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("FR-072: %s must return 404, got %d (%s)", target, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		// Not an empty list: the failure mode FR-072 names by name.
		if containsAny(body, `"items":[]`, `"items": []`) {
			t.Errorf("FR-072: an unknown campaign code must not answer with an empty list: %s", body)
		}
		// A Vietnamese message that NAMES the code, so an operator who typed it
		// wrong knows what to correct.
		message := campaignErrorMessage(t, rec)
		if message == "" {
			t.Fatalf("FR-072: the 404 must carry a message, got %q", body)
		}
		if !containsAny(message, unknownCode) {
			t.Errorf("FR-072: the message must name the code that was asked for, got %q", message)
		}
	}
}

// ---- T076b: the three cases the contract's section 4 leaves to the route ----

// TestNormalUserIsRefusedWith403 is T076b / FR-070 / SC-011.
//
// FR-070's subject is an ordinary account that KNOWS the path. The refusal happens
// in `RolesAuthorize` before the handler runs, which is exactly why this test goes
// through the route: called directly, the handler would answer 200 and the leak
// would be invisible.
func TestNormalUserIsRefusedWith403(t *testing.T) {
	stub := &stubCampaignQuery{
		summary:  &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"},
		verdicts: map[uuid.UUID]*sub_dto.CampaignAccountEligibilityRes{},
	}
	engine := campaignEngine(t, stub, authenticatedAs(roleslug.User), newAdminLimiter(alwaysAllowRateLimiter{}))

	for _, target := range []string{
		campaignPath,
		campaignPath + "/launch-2026-10",
		campaignPath + "/launch-2026-10/claims",
		campaignPath + "/launch-2026-10/eligibility/" + uuid.New().String(),
	} {
		rec := request(t, engine, http.MethodGet, target)

		if rec.Code != http.StatusForbidden {
			t.Errorf("FR-070: a normal account calling %s must be refused with 403, got %d (%s)",
				target, rec.Code, rec.Body.String())
		}
		// The refusal must not leak the campaign's contents. An error body that
		// still carried the numbers would have answered the question the route
		// exists to prevent.
		if containsAny(rec.Body.String(), "launch-2026-10\"", "grantedMainCount") {
			t.Errorf("FR-070: the refusal leaked campaign data to a normal account: %s", rec.Body.String())
		}
	}
	// And the use case was never reached, which is the stronger statement: nothing
	// about the endpoint's contents was even read.
	if stub.calls != 0 {
		t.Errorf("FR-070: the look-up ran %d times for a refused caller; it must not run at all",
			stub.calls)
	}
}

// TestRateLimitExceededReturns429 is T076b / FR-071 / SC-011.
//
// FR-071 asks the look-up to obey the SAME limit as the other administrator
// features, so the test uses the production middleware with a limiter that refuses
// — the moment an administrator has spent the per-minute budget. Asserting the 429
// also pins which limiter is on the route: a second, campaign-only limiter would
// answer identically here, and the requirement is that there is not one.
func TestRateLimitExceededReturns429(t *testing.T) {
	stub := &stubCampaignQuery{summary: &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"}}
	engine := campaignEngine(t, stub, authenticatedAs(roleslug.Admin), newAdminLimiter(neverAllowRateLimiter{}))

	for _, target := range []string{
		campaignPath,
		campaignPath + "/launch-2026-10",
		campaignPath + "/launch-2026-10/claims",
	} {
		rec := request(t, engine, http.MethodGet, target)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("FR-071: %s must return 429 once the administrator's budget is spent, got %d (%s)",
				target, rec.Code, rec.Body.String())
		}
	}
	if stub.calls != 0 {
		t.Errorf("FR-071: the look-up ran %d times behind an exhausted limit; it must not run at all",
			stub.calls)
	}
}

// TestUnknownAccountReturns404WithAVietnameseMessage is T076b / contract §4.
//
// A `:userId` that does not exist is its OWN 404, distinct from the unknown-code
// one: the two are typed by two different people, and merging them leaves an
// operator not knowing which of the two they mistyped. The message says so in
// Vietnamese, as the constitution requires of every client-facing string.
func TestUnknownAccountReturns404WithAVietnameseMessage(t *testing.T) {
	stub := &stubCampaignQuery{
		summary:  &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"},
		verdicts: map[uuid.UUID]*sub_dto.CampaignAccountEligibilityRes{},
	}
	engine := newCampaignEngine(t, stub)

	missing := uuid.New()
	rec := request(t, engine, http.MethodGet,
		campaignPath+"/launch-2026-10/eligibility/"+missing.String())

	if rec.Code != http.StatusNotFound {
		t.Fatalf("contract §4: an account that does not exist must return 404, got %d (%s)",
			rec.Code, rec.Body.String())
	}
	const want = "Không tìm thấy tài khoản."
	if got := campaignErrorMessage(t, rec); got != want {
		t.Errorf("the message must match contracts/campaign-query-api.md §4\n  want = %q\n  got  = %q", want, got)
	}
}

// ---- T076c: the from/to filter (FR-067) ----

// TestClaimsFromToFilterNarrowsTheGrantInstant is T076c / FR-067 / SC-011.
//
// The filter axis is the GRANT instant, half-open: `from` inclusive, `to`
// exclusive. Both halves matter, and the exclusive one is easy to get wrong — an
// inclusive `to` puts the boundary grant in two consecutive exports and the
// operator never notices, because each export looks complete.
func TestClaimsFromToFilterNarrowsTheGrantInstant(t *testing.T) {
	grantedAt := time.Date(2026, 10, 2, 9, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	from := grantedAt.Add(-30 * time.Minute)
	to := grantedAt.Add(30 * time.Minute)

	stub := &stubCampaignQuery{
		summary: &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"},
		claims: []*sub_dto.CampaignClaimRes{{
			UserID:    uuid.New(),
			Username:  "nguyenvana",
			PlanSlug:  "premium-monthly",
			GrantedAt: grantedAt,
			Source:    sub_dto.CampaignClaimSourceMain,
		}},
		claimsTotal: 1,
	}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet,
		campaignPath+"/launch-2026-10/claims?from="+url.QueryEscape(from.Format(time.RFC3339))+
			"&to="+url.QueryEscape(to.Format(time.RFC3339)))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	body := claimsResponse(t, rec)
	if len(body.Items) != 1 {
		t.Fatalf("expected the grant inside the window, got %d items", len(body.Items))
	}
	if !body.Items[0].GrantedAt.Equal(grantedAt) {
		t.Errorf("expected the grant at %s, got %s", grantedAt, body.Items[0].GrantedAt)
	}

	// The bounds must arrive at the use case as INSTANTS. A handler that passed the
	// raw strings would compile, work for one client and break for another whose
	// zone differs — seven hours off in production, with nothing to show it.
	if stub.lastFilter.From == nil || stub.lastFilter.To == nil {
		t.Fatalf("FR-067: the handler must parse from/to into instants, got %#v", stub.lastFilter)
	}
	if !stub.lastFilter.From.Equal(from) || !stub.lastFilter.To.Equal(to) {
		t.Errorf("the filter must carry the parsed instants: got %s..%s, want %s..%s",
			stub.lastFilter.From, stub.lastFilter.To, from, to)
	}
}

// TestClaimsFromAfterToIsRefused is T076c / FR-067.
//
// `from > to` is a BUG in the caller. Answering it with an empty page would hide
// the bug until somebody noticed a grant that "never happened" — and by then it
// has been reported to a customer.
func TestClaimsFromAfterToIsRefused(t *testing.T) {
	from := time.Date(2026, 10, 2, 10, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	to := from.Add(-time.Hour)

	stub := &stubCampaignQuery{
		summary: &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"},
	}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet,
		campaignPath+"/launch-2026-10/claims?from="+url.QueryEscape(from.Format(time.RFC3339))+
			"&to="+url.QueryEscape(to.Format(time.RFC3339)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("FR-067: a reversed from/to must be refused with 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	// The request REACHED the use case — which is where the rule lives — and no
	// filter was built for a repository to run against.
	if stub.calls != 1 {
		t.Errorf("expected the reversed window to reach the use case once, got %d calls", stub.calls)
	}
	if stub.lastFilter.From != nil || stub.lastFilter.To != nil {
		t.Errorf("a refused window must not reach the repository; the filter %#v was handed over", stub.lastFilter)
	}
}

// TestClaimsMalformedFromIsRefused is T076c / FR-067.
//
// A malformed bound is IGNORED by a lenient implementation, which then answers a
// different question from the one that was asked: an operator who typed the wrong
// format gets a plausible-looking page and concludes the campaign granted fewer
// plans than it did.
func TestClaimsMalformedFromIsRefused(t *testing.T) {
	for _, target := range []string{
		campaignPath + "/launch-2026-10/claims?from=02-10-2026",
		campaignPath + "/launch-2026-10/claims?to=yesterday",
		// A zone-less timestamp is not accepted: read in the server's zone it would
		// be seven hours off in production, and the filter would silently drop the
		// grants the operator was looking for.
		campaignPath + "/launch-2026-10/claims?from=2026-10-02T00:00:00",
	} {
		stub := &stubCampaignQuery{summary: &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"}}
		engine := newCampaignEngine(t, stub)

		rec := request(t, engine, http.MethodGet, target)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("FR-067: a malformed bound on %s must be refused with 400, got %d (%s)",
				target, rec.Code, rec.Body.String())
		}
	}
}

// TestClaimsOmittedBoundsAreUnbounded is T076c / FR-067.
//
// An operator exporting a whole month passes only one bound. Treating the missing
// one as the zero time would exclude every grant before 1970 — an empty export
// that looks exactly like a campaign that granted nobody.
func TestClaimsOmittedBoundsAreUnbounded(t *testing.T) {
	stub := &stubCampaignQuery{
		summary: &sub_dto.CampaignSummaryRes{Code: "launch-2026-10"},
	}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet, campaignPath+"/launch-2026-10/claims")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with no bounds, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.lastFilter.From != nil || stub.lastFilter.To != nil {
		t.Errorf("an omitted bound must stay nil, not become the zero time: got %#v", stub.lastFilter)
	}
}

// ---- small helpers ----

// request builds and sends one request through the engine.
func request(t *testing.T, engine *gin.Engine, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

// containsAny reports whether body holds any of the needles.
func containsAny(body string, needles ...string) bool {
	for _, needle := range needles {
		if needle == "" {
			continue
		}
		if strings.Contains(body, needle) {
			return true
		}
	}
	return false
}

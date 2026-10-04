package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 026 T037 (FR-052..FR-055, FR-060, FR-063) — `POST /api/v1/admin/campaigns` over HTTP.
//
// WHY THESE GO THROUGH THE ROUTER AND NOT THE HANDLER. Two of the six codes FR-054
// assigns to this endpoint are produced by middleware that runs BEFORE the handler:
// 403 comes from the group's `RolesAuthorize` and 429 from the limiter. Calling
// `h.Create` directly would skip both, and every test below would pass against an
// endpoint that opens campaigns for any signed-in customer and without any limit —
// which is exactly the defect 024 found the hard way in the grant path
// (`admin_grant_batch_http_test.go`). The wiring is the subject here.
//
// A STUB, NOT THE REAL USE CASE. The rules — 409, the 400/422 split, the audit row —
// are decided in `registry_uc.go` and proved in `registry_uc_create_test.go`, against
// the repository they actually write through. What is proved HERE is the HTTP surface:
// the path, the status codes, the role check, the limiter and the response envelope.

// registryStub answers `Create` and `Close` from a script, and records what reached
// them.
//
// It implements the WHOLE interface so that adding a method in a later phase (T052)
// breaks this file loudly instead of leaving a stub that silently stops covering the
// new route.
type registryStub struct {
	calls int
	// lastReq is what the handler built, so a test can assert on the REQUEST the
	// handler assembled — a handler that passed the raw struct through with a missing
	// field would otherwise look identical from outside.
	lastReq *sub_dto.CampaignCreateReq
	// lastActor and lastIP prove FR-060's "the audit row names who did it" is fed from
	// the request rather than invented.
	lastActor uuid.UUID
	lastIP    *string
	// result is returned when err is nil.
	result *sub_dto.CampaignSummaryRes
	// err is returned verbatim, so a test can script 409, 422 or 500 without the stub
	// knowing FR-054's table.
	err error

	// ---- the close side (026 T048) ----
	//
	// SEPARATE FIELDS RATHER THAN SHARED ONES, because the two operations have
	// different inputs: `Create` carries the code in its body, `Close` in the path. One
	// shared `err` and one shared `lastReq` would force a close test to set the create
	// result to reach its own, and the assertions would then be about which field the
	// test happened to remember.
	closeCalls int
	// lastCloseCode is the `:code` the handler read out of the URL. It is asserted
	// because the body has no `code` field at all (FR-024), so a handler that passed
	// the wrong segment would still reach the use case with a plausible string.
	lastCloseCode string
	lastCloseReq  *sub_dto.CampaignCloseReq
	closeResult   *sub_dto.CampaignSummaryRes
	closeErr      error

	// ---- the update side (026 T053/T058) ----
	//
	// SEPARATE FIELDS AGAIN, for the same reason as the close's: an edit carries the
	// code in the PATH and its own body shape, and sharing `err` with either
	// operation would make a test reach for the wrong field to script its refusal.
	updateCalls int
	// lastUpdateCode is asserted for the same reason as `lastCloseCode`:
	// `CampaignUpdateReq` has NO `code` field (FR-024), so the URL segment is the only
	// way a code can reach this method.
	lastUpdateCode string
	lastUpdateReq  *sub_dto.CampaignUpdateReq
	updateResult   *sub_dto.CampaignSummaryRes
	updateErr      error
}

// Update records the edit and answers from the script.
//
// `version` is echoed into the default result on purpose: FR-055 says the response
// carries the version the write PRODUCED, and a stub that always answered
// `version: 1` would let a handler that echoed the request pass every test here.
func (s *registryStub) Update(
	_ context.Context,
	actorAdminID uuid.UUID,
	campaignCode string,
	req *sub_dto.CampaignUpdateReq,
	requestIP *string,
) (*sub_dto.CampaignSummaryRes, error) {
	s.updateCalls++
	s.lastActor = actorAdminID
	s.lastIP = requestIP
	s.lastUpdateCode = campaignCode
	copied := *req
	s.lastUpdateReq = &copied
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	if s.updateResult != nil {
		return s.updateResult, nil
	}
	return &sub_dto.CampaignSummaryRes{
		Code:         campaignCode,
		Version:      req.Version + 1,
		BudgetLocked: false,
	}, nil
}

func (s *registryStub) Create(
	_ context.Context,
	actorAdminID uuid.UUID,
	req *sub_dto.CampaignCreateReq,
	requestIP *string,
) (*sub_dto.CampaignSummaryRes, error) {
	s.calls++
	s.lastActor = actorAdminID
	s.lastIP = requestIP
	copied := *req
	s.lastReq = &copied
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return &sub_dto.CampaignSummaryRes{Code: req.Code}, nil
	}
	return s.result, nil
}

func (s *registryStub) Close(
	_ context.Context,
	actorAdminID uuid.UUID,
	campaignCode string,
	req *sub_dto.CampaignCloseReq,
	requestIP *string,
) (*sub_dto.CampaignSummaryRes, error) {
	s.closeCalls++
	s.lastActor = actorAdminID
	s.lastIP = requestIP
	s.lastCloseCode = campaignCode
	copied := *req
	s.lastCloseReq = &copied
	if s.closeErr != nil {
		return nil, s.closeErr
	}
	if s.closeResult == nil {
		return &sub_dto.CampaignSummaryRes{Code: campaignCode, Status: sub_dto.CampaignStatusClosed}, nil
	}
	return s.closeResult, nil
}

// ---- engine ----

// registryEngine builds the real route group around a stubbed registry use case.
//
// The group, the role check and the limiter are wired exactly as `router.go` wires
// them, which is the point: a test that installed its own check would pass against a
// route carrying none. The limiter is the PRODUCTION middleware with its limits read
// from configuration, because FR-053 asks this route to obey the SAME budget as the
// grant path and a hard-coded limit would not prove that.
func registryEngine(
	t *testing.T,
	stub *registryStub,
	as gin.HandlerFunc,
	limiter *shared_middleware.AdminGrantRateLimitMiddleware,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := logger.New("test", "", "info", false)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))

	h := NewCampaignRegistryHandler(log, stub)
	group := engine.Group("/api/v1/admin")
	group.Use(as, shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		campaigns := group.Group("/campaigns")
		campaigns.POST("", limiter.Handle(), shared_pres.WrapHandler(h.Create))
		// 026 T048: the close route, registered on the same group with the same
		// middleware, exactly as `router.go` registers it. Both live in this engine
		// because the Gin tree is the subject: a static `POST ""` beside a `POST
		// "/:code/close"` is the shape that panics at startup when it is wrong, and
		// only registering both proves it does not.
		campaigns.POST("/:code/close", limiter.Handle(), shared_pres.WrapHandler(h.Close))
		// 026 T058: the edit route. It is registered in THIS engine, on the same group
		// with the same middleware, because the Gin tree is the subject: `PATCH
		// "/:code"` beside `POST "/:code/close"` is a parameter registered beside a
		// parameter, and only building the real tree proves the R11 conflict (a static
		// segment beside a parameter) is still absent.
		campaigns.PATCH("/:code", limiter.Handle(), shared_pres.WrapHandler(h.Update))
	}
	return engine
}

// newRegistryEngine wires an administrator with a generous budget.
func newRegistryEngine(t *testing.T, stub *registryStub) *gin.Engine {
	t.Helper()
	return registryEngine(t, stub, authenticatedAs(roleslug.Admin), newAdminLimiter(alwaysAllowRateLimiter{}))
}

// ---- body helpers ----

// createBody is a well-formed request as a client would send it.
func createBody(t *testing.T) *bytes.Reader {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"code":     "tet-2027",
		"planSlug": "premium-monthly",
		"quota":    100,
		"reserve":  10,
		"startsAt": "2027-01-20T00:00:00+07:00",
		"endsAt":   "2027-02-05T00:00:00+07:00",
	})
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}
	return bytes.NewReader(raw)
}

// postCampaign sends one create request.
func postCampaign(t *testing.T, engine *gin.Engine, body *bytes.Reader) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, campaignPath, body)
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)
	return rec
}

// createdResponse pulls the summary out of the 201 envelope.
func createdResponse(t *testing.T, rec *httptest.ResponseRecorder) sub_dto.CampaignSummaryRes {
	t.Helper()
	var parsed struct {
		Data sub_dto.CampaignSummaryRes `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse response %q: %v", rec.Body.String(), err)
	}
	return parsed.Data
}

// ---- 201 ----

// TestCreateCampaignReturns201WithTheCreatedSummary is the success path.
//
// 201 AND NOT 200 is the assertion, and it is not cosmetic: the campaign now EXISTS in
// the registry, and 201 is what tells a client that the thing it asked for was created
// rather than replaced or echoed back. The envelope carries the same summary the read
// endpoints return, so an admin screen renders the campaign and then refreshes instead
// of reformatting one shape into another (contract §3.2).
//
// NOTE WHAT IS NOT ASSERTED HERE: `version`, `budgetLocked` and `closedAt`. They are
// FR-051's additive fields and they are added to `CampaignSummaryRes` by T039/T050, in
// the phases that need them. Asserting them now would assert a field that does not
// exist; the version the response will carry is written by the use case and proved
// against the stored row in `registry_uc_create_test.go`.
func TestCreateCampaignReturns201WithTheCreatedSummary(t *testing.T) {
	stub := &registryStub{result: &sub_dto.CampaignSummaryRes{
		Code:     "tet-2027",
		PlanSlug: "premium-monthly",
		Quota:    100,
		Reserve:  10,
		HardCap:  110,
		Status:   sub_dto.CampaignStatusNotStarted,
		StartsAt: time.Date(2027, 1, 20, 0, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
	}}
	engine := newRegistryEngine(t, stub)

	rec := postCampaign(t, engine, createBody(t))

	if rec.Code != http.StatusCreated {
		t.Fatalf("contract §3.2: a successful create must return 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	data := createdResponse(t, rec)
	if data.Code != "tet-2027" || data.Quota != 100 || data.HardCap != 110 {
		t.Errorf("the 201 body must be the created campaign, got %+v", data)
	}
	if data.Status != sub_dto.CampaignStatusNotStarted {
		t.Errorf("status = %q, want %q for a campaign that opens in the future", data.Status, sub_dto.CampaignStatusNotStarted)
	}
}

// TestCreateCampaignReachesTheUseCaseWithTheActorAndAddress is FR-060's input half.
//
// The audit row is what makes "who opened this campaign" answerable, and it can only
// name somebody if the handler passes the actor through. It is asserted here rather
// than only at the use-case level because the handler is the layer that reads the
// identity out of the request — a use-case test that supplies the actor itself proves
// nothing about whether the HTTP path supplies it.
func TestCreateCampaignReachesTheUseCaseWithTheActorAndAddress(t *testing.T) {
	stub := &registryStub{}
	engine := newRegistryEngine(t, stub)

	rec := postCampaign(t, engine, createBody(t))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.calls != 1 {
		t.Fatalf("expected the use case to be called once, got %d", stub.calls)
	}
	if stub.lastActor == uuid.Nil {
		t.Error("FR-060: the handler must pass the authenticated administrator, not a zero id")
	}
	if stub.lastIP == nil || *stub.lastIP == "" {
		t.Error("FR-060: the handler must pass the request address so a disputed change can be traced to a connection")
	}
	// The request the handler BUILT, not the raw JSON: a missing `quota` arriving as a
	// zero value would be a 422 from the validator instead of the binder's 400, and the
	// difference is visible only here.
	if stub.lastReq.Code != "tet-2027" || stub.lastReq.Quota == nil || *stub.lastReq.Quota != 100 {
		t.Errorf("the handler must pass the bound request through, got %+v", stub.lastReq)
	}
	if stub.lastReq.EndsAt == nil || *stub.lastReq.EndsAt != "2027-02-05T00:00:00+07:00" {
		t.Errorf("endsAt must arrive as the string the client sent, not as a parsed instant: %+v", stub.lastReq.EndsAt)
	}
}

// ---- FR-054: 400 ----

// TestCreateCampaignRejectsBrokenShapesWith400 is the 400 row of FR-054 over HTTP.
//
// THE BINDER OWNS HALF OF THIS, which is why it is tested through the route: a missing
// `code`, `planSlug`, `quota` or `startsAt` is refused by `validation.BindJSON` with
// the shared Vietnamese message, and a test that called the use case directly would
// never reach that code. The other half — a malformed `code`, a zone-less timestamp — is
// the validator's, and it is scripted through the stub here because its behaviour is
// proved in `registry_uc_create_test.go`.
func TestCreateCampaignRejectsBrokenShapesWith400(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
		// scripted is the use-case error the handler must propagate unchanged, or nil
		// when the refusal happens in the binder before the use case is reached.
		scripted error
	}{
		{
			name: "a missing code",
			body: map[string]any{
				"planSlug": "premium-monthly", "quota": 100,
				"startsAt": "2027-01-20T00:00:00+07:00",
			},
		},
		{
			name: "a missing planSlug",
			body: map[string]any{
				"code": "tet-2027", "quota": 100,
				"startsAt": "2027-01-20T00:00:00+07:00",
			},
		},
		{
			name: "a missing quota",
			body: map[string]any{
				"code": "tet-2027", "planSlug": "premium-monthly",
				"startsAt": "2027-01-20T00:00:00+07:00",
			},
		},
		{
			name: "a missing startsAt",
			body: map[string]any{
				"code": "tet-2027", "planSlug": "premium-monthly", "quota": 100,
			},
		},
		{
			name: "a quota that is not a number",
			body: map[string]any{
				"code": "tet-2027", "planSlug": "premium-monthly", "quota": "many",
				"startsAt": "2027-01-20T00:00:00+07:00",
			},
		},
		{
			name:     "a malformed code",
			body:     map[string]any{"code": "Launch 2026", "planSlug": "premium-monthly", "quota": 100, "startsAt": "2027-01-20T00:00:00+07:00"},
			scripted: subscriptionerrors.ErrCampaignCodeInvalid(),
		},
		{
			name:     "a zone-less startsAt",
			body:     map[string]any{"code": "tet-2027", "planSlug": "premium-monthly", "quota": 100, "startsAt": "2027-01-20 00:00:00"},
			scripted: subscriptionerrors.ErrCampaignTimestampMalformed("startsAt"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registryStub{err: tc.scripted}
			engine := newRegistryEngine(t, stub)

			raw, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			rec := postCampaign(t, engine, bytes.NewReader(raw))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("FR-054: expected 400, got %d (%s)", rec.Code, rec.Body.String())
			}
			if message := campaignErrorMessage(t, rec); message == "" {
				t.Errorf("a 400 must carry a Vietnamese message, got %q", rec.Body.String())
			}
			// The binder's refusals happen BEFORE the use case; a scripted one reaches it.
			// Both must produce 400 and neither may write anything, but only the binder
			// case may claim the use case was never called.
			if tc.scripted == nil && stub.calls != 0 {
				t.Errorf("a shape the binder refuses must never reach the use case, got %d calls", stub.calls)
			}
		})
	}
}

// ---- FR-054: 409 and 422 ----

// TestCreateCampaignReturns409ForADuplicateCode is US1 acceptance scenario 3 at the HTTP
// surface.
//
// The assertion is on the BODY as well as the code: FR-054 gives 409 its own meaning
// and contract §3.3 names the campaign code in the message, because the operator
// mistyped it from another source. A generic "conflict" would leave them looking for a
// collision they cannot identify.
func TestCreateCampaignReturns409ForADuplicateCode(t *testing.T) {
	stub := &registryStub{err: subscriptionerrors.ErrCampaignDuplicateCode("tet-2027")}
	engine := newRegistryEngine(t, stub)

	rec := postCampaign(t, engine, createBody(t))

	if rec.Code != http.StatusConflict {
		t.Fatalf("US1.3: a duplicate code must return 409, got %d (%s)", rec.Code, rec.Body.String())
	}
	if message := campaignErrorMessage(t, rec); !strings.Contains(message, "tet-2027") {
		t.Errorf("the 409 message must name the code that was taken, got %q", message)
	}
}

// TestCreateCampaignReturns422ForWrongMeaning is FR-054's 422 row.
//
// 422 is NOT ABOUT SYNTAX, and the distinction is the requirement: a `400` here would
// tell the operator their request is unreadable when it is perfectly readable and the
// NUMBERS are wrong. So the cases scripted below are all well-formed JSON with
// plausible field names, and each still has to come back 422.
func TestCreateCampaignReturns422ForWrongMeaning(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"a zero quota", subscriptionerrors.ErrCampaignInvalidField("quota", "phải lớn hơn hoặc bằng 1")},
		{"a negative reserve", subscriptionerrors.ErrCampaignInvalidField("reserve", "không được âm")},
		{"a blank planSlug", subscriptionerrors.ErrCampaignInvalidField("planSlug", "không được để trống")},
		{"an endsAt before startsAt", subscriptionerrors.ErrCampaignInvalidField("endsAt", "phải sau thời điểm bắt đầu")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registryStub{err: tc.err}
			engine := newRegistryEngine(t, stub)

			rec := postCampaign(t, engine, createBody(t))

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("FR-054: expected 422, got %d (%s)", rec.Code, rec.Body.String())
			}
			if message := campaignErrorMessage(t, rec); message == "" {
				t.Errorf("a 422 must carry a Vietnamese message naming the field, got %q", rec.Body.String())
			}
		})
	}
}

// ---- FR-054: 403 and 429 ----

// TestCreateCampaignIsRefusedForANormalAccount is FR-053 / US1 acceptance scenario 4.
//
// The refusal happens in `RolesAuthorize`, before the handler runs, which is exactly
// why this goes through the route: called directly the handler would answer 201 and the
// leak would be invisible. It also asserts that the use case was NEVER reached, which
// is the stronger statement — nothing about the request was even read.
func TestCreateCampaignIsRefusedForANormalAccount(t *testing.T) {
	stub := &registryStub{}
	engine := registryEngine(t, stub, authenticatedAs(roleslug.User), newAdminLimiter(alwaysAllowRateLimiter{}))

	rec := postCampaign(t, engine, createBody(t))

	if rec.Code != http.StatusForbidden {
		t.Errorf("US1.4: a normal account must be refused with 403, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.calls != 0 {
		t.Errorf("FR-053: the registry write ran %d times for a refused caller; it must not run at all", stub.calls)
	}
}

// TestCreateCampaignReturns429OnceTheAdministratorBudgetIsSpent is FR-053's other half.
//
// The route carries the SAME limiter as the bulk grant, so the test uses the production
// middleware over a limiter that refuses — the moment an administrator has spent the
// per-minute budget. Asserting the 429 also pins WHICH limiter is on the route: a
// second, campaign-only limiter would answer identically here, and the requirement is
// that there is not one (research.md R10).
func TestCreateCampaignReturns429OnceTheAdministratorBudgetIsSpent(t *testing.T) {
	stub := &registryStub{}
	engine := registryEngine(t, stub, authenticatedAs(roleslug.Admin), newAdminLimiter(neverAllowRateLimiter{}))

	rec := postCampaign(t, engine, createBody(t))

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("FR-053: an exhausted administrator budget must return 429, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.calls != 0 {
		t.Errorf("FR-053: the registry write ran %d times behind an exhausted limit; it must not run at all", stub.calls)
	}
}

// ---- the codes FR-054 forbids ----

// TestCreateCampaignNeverAnswersWithACodeOutsideTheTable is FR-054's closing rule,
// stated as a test because it is the rule most likely to be broken by a later change.
//
// "Không mã nào ngoài bảng này được trả về từ ba endpoint ghi." A 500 from a duplicate
// code, or a 409 from an unreachable database, would both satisfy the code that produced
// them while leaving the client unable to decide whether to change its input or retry.
func TestCreateCampaignNeverAnswersWithACodeOutsideTheTable(t *testing.T) {
	allowed := map[int]string{
		http.StatusBadRequest:          "400 hỏng hình dạng",
		http.StatusConflict:            "409 trùng mã lúc tạo",
		http.StatusUnprocessableEntity: "422 sai nghĩa",
		http.StatusUnauthorized:        "401 chưa xác thực",
		http.StatusForbidden:           "403 không phải Admin",
		http.StatusTooManyRequests:     "429 vượt giới hạn",
		http.StatusInternalServerError: "500 lỗi hạ tầng",
	}

	for _, scripted := range []struct {
		name string
		err  error
	}{
		{"a duplicate code", subscriptionerrors.ErrCampaignDuplicateCode("tet-2027")},
		{"a malformed code", subscriptionerrors.ErrCampaignCodeInvalid()},
		{"a zone-less timestamp", subscriptionerrors.ErrCampaignTimestampMalformed("startsAt")},
		{"a zero quota", subscriptionerrors.ErrCampaignInvalidField("quota", "phải lớn hơn hoặc bằng 1")},
		{"a database fault", subscriptionerrors.ErrCampaignRegistryWriteFailed("tet-2027", errors.New("connection reset"))},
		{"a failed audit insert", subscriptionerrors.ErrCampaignRegistryAuditFailed("tet-2027", errors.New("outbox down"))},
	} {
		stub := &registryStub{err: scripted.err}
		engine := newRegistryEngine(t, stub)

		rec := postCampaign(t, engine, createBody(t))

		if _, ok := allowed[rec.Code]; !ok {
			t.Errorf("FR-054: %s answered %d, which is not in the table (%v)", scripted.name, rec.Code, rec.Body.String())
		}
		// And the raw database text must never reach the client, or an operator learns
		// the schema instead of what to do (design-pattern "never expose raw database
		// errors").
		if strings.Contains(rec.Body.String(), "connection reset") || strings.Contains(rec.Body.String(), "outbox down") {
			t.Errorf("%s leaked the underlying cause to the client: %s", scripted.name, rec.Body.String())
		}
	}
}

// ---- 026 T048: POST /campaigns/:code/close ----
//
// The seven codes below are contract §5.4's table. Two are produced by middleware that
// never reaches the handler (403 and 429), and one — 400 — by the binder for the two
// REQUIRED fields, which is why it can only be observed through the route.

// closeBody is a well-formed close request as a client would send it.
func closeBody(t *testing.T) *bytes.Reader {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"version": 3, "reason": "Hết ngân sách khuyến mãi tháng 11"})
	if err != nil {
		t.Fatalf("marshal close body: %v", err)
	}
	return bytes.NewReader(raw)
}

// postClose sends one close request for the given code.
func postClose(t *testing.T, engine *gin.Engine, code string, body *bytes.Reader) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, campaignPath+"/"+code+"/close", body)
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)
	return rec
}

// TestCloseCampaignReturns200WithTheClosedSummary is the success path, including the
// idempotent no-op (contract §5.3) which the handler cannot distinguish and must not
// need to.
//
// 200 AND NOT 201: nothing was created. And `status: closed` is asserted because it is
// the two things the operator asked for, in one response, without a refresh.
func TestCloseCampaignReturns200WithTheClosedSummary(t *testing.T) {
	closedAt := time.Date(2026, 11, 1, 9, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	stub := &registryStub{closeResult: &sub_dto.CampaignSummaryRes{
		Code:     "tet-2027",
		PlanSlug: "premium-monthly",
		Quota:    100,
		Reserve:  10,
		HardCap:  110,
		Status:   sub_dto.CampaignStatusClosed,
		StartsAt: time.Date(2027, 1, 20, 0, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
		ClosedAt: &closedAt,
	}}
	engine := newRegistryEngine(t, stub)

	rec := postClose(t, engine, "tet-2027", closeBody(t))

	if rec.Code != http.StatusOK {
		t.Fatalf("contract §5: a successful close must return 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var parsed struct {
		Data sub_dto.CampaignSummaryRes `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse response %q: %v", rec.Body.String(), err)
	}
	if parsed.Data.Status != sub_dto.CampaignStatusClosed {
		t.Errorf("status = %q, want %q", parsed.Data.Status, sub_dto.CampaignStatusClosed)
	}
	if parsed.Data.ClosedAt == nil {
		t.Error("the 200 body must carry closedAt (FR-051)")
	}
}

// TestCloseCampaignReachesTheUseCaseWithThePathCodeAndTheActor is FR-060's input half
// for close, and it is the one assertion the create tests could not make.
//
// `code` is NOT in the body — it is immutable after creation (FR-024) — so the only
// way it reaches the use case is through the URL segment. A handler that read it from
// anywhere else, or from the wrong segment, would still compile and would still answer
// 200 in every other test here.
func TestCloseCampaignReachesTheUseCaseWithThePathCodeAndTheActor(t *testing.T) {
	stub := &registryStub{}
	engine := newRegistryEngine(t, stub)

	rec := postClose(t, engine, "launch-2026-10", closeBody(t))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.closeCalls != 1 {
		t.Fatalf("expected the use case to be called once, got %d", stub.closeCalls)
	}
	if stub.lastCloseCode != "launch-2026-10" {
		t.Errorf("the handler must pass the :code segment through, got %q", stub.lastCloseCode)
	}
	if stub.lastActor == uuid.Nil {
		t.Error("FR-060: the handler must pass the authenticated administrator, not a zero id")
	}
	if stub.lastIP == nil || *stub.lastIP == "" {
		t.Error("FR-060: the handler must pass the request address so a disputed closure can be traced")
	}
	// `version` MUST ARRIVE AS THE INTEGER THE CLIENT SENT — it is the If-Match of
	// FR-055, and a handler that defaulted it to zero would turn every close into a 412.
	if stub.lastCloseReq.Version != 3 {
		t.Errorf("version = %d, want the 3 the client sent", stub.lastCloseReq.Version)
	}
	if stub.lastCloseReq.Reason == nil || *stub.lastCloseReq.Reason == "" {
		t.Errorf("reason must arrive as the string the client sent, got %+v", stub.lastCloseReq.Reason)
	}
}

// TestCloseCampaignRejectsBrokenShapesWith400 is the 400 row of contract §5.4: a
// missing `version` or a missing `reason`. These are the binder's refusals, which is
// why they can only be observed through the route — and the assertion that the use case
// was NEVER reached is the stronger half: nothing about such a request was even read.
func TestCloseCampaignRejectsBrokenShapesWith400(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
	}{
		{"a missing version", map[string]any{"reason": "Hết ngân sách"}},
		{"a missing reason", map[string]any{"version": 3}},
		{"an empty body", map[string]any{}},
		{"a version that is not a number", map[string]any{"version": "three", "reason": "Hết ngân sách"}},
		{"a reason that is not a string", map[string]any{"version": 3, "reason": 42}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registryStub{}
			engine := newRegistryEngine(t, stub)

			raw, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			rec := postClose(t, engine, "tet-2027", bytes.NewReader(raw))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("contract §5.4: expected 400, got %d (%s)", rec.Code, rec.Body.String())
			}
			if message := campaignErrorMessage(t, rec); message == "" {
				t.Errorf("a 400 must carry a Vietnamese message, got %q", rec.Body.String())
			}
			if stub.closeCalls != 0 {
				t.Errorf("a request the binder refuses must never reach the use case, got %d calls",
					stub.closeCalls)
			}
		})
	}
}

// TestCloseCampaignReturns404412And422 is FR-054's remaining rows for close, in the
// order contract §5.4 lists them.
//
// Each is a DIFFERENT mistake with a different remedy — a mistyped code, a page that
// has been open too long, and a confirmation box submitted empty — so returning the
// wrong one would send the operator to do the wrong next thing. 412 in particular is
// not fixable by editing the form: the page they were looking at is not the current
// state.
func TestCloseCampaignReturns404412And422(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
		// mentions is a fragment the message must contain, where the message has to
		// carry something actionable rather than a bare code.
		mentions string
	}{
		{
			name:     "an unknown code",
			err:      subscriptionerrors.ErrCampaignNotFound("khong-ton-tai"),
			want:     http.StatusNotFound,
			mentions: "khong-ton-tai",
		},
		{
			name:     "a stale version",
			err:      subscriptionerrors.ErrCampaignVersionConflict("tet-2027"),
			want:     http.StatusPreconditionFailed,
			mentions: "tet-2027",
		},
		{
			name:     "a blank reason",
			err:      subscriptionerrors.ErrCampaignInvalidField("reason", "không được để trống"),
			want:     http.StatusUnprocessableEntity,
			mentions: "reason",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registryStub{closeErr: tc.err}
			engine := newRegistryEngine(t, stub)

			rec := postClose(t, engine, "tet-2027", closeBody(t))

			if rec.Code != tc.want {
				t.Fatalf("expected %d, got %d (%s)", tc.want, rec.Code, rec.Body.String())
			}
			if message := campaignErrorMessage(t, rec); !strings.Contains(message, tc.mentions) {
				t.Errorf("the message must carry %q, got %q", tc.mentions, message)
			}
			// The use case DID run here, unlike the 400 cases: it is the layer that
			// knows the code does not exist, the version moved, or the reason is blank.
			if stub.closeCalls != 1 {
				t.Errorf("expected the use case to be reached once, got %d", stub.closeCalls)
			}
		})
	}
}

// TestCloseCampaignIsRefusedForANormalAccount is FR-053 applied to close. A close is
// the most consequential write in the registry — one-way, and not undoable (FR-034) —
// so "an ordinary signed-in customer must not reach it" matters more here than anywhere
// else in the group.
func TestCloseCampaignIsRefusedForANormalAccount(t *testing.T) {
	stub := &registryStub{}
	engine := registryEngine(t, stub, authenticatedAs(roleslug.User), newAdminLimiter(alwaysAllowRateLimiter{}))

	rec := postClose(t, engine, "tet-2027", closeBody(t))

	if rec.Code != http.StatusForbidden {
		t.Errorf("FR-053: a normal account must be refused with 403, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.closeCalls != 0 {
		t.Errorf("FR-053: the close ran %d times for a refused caller; it must not run at all", stub.closeCalls)
	}
}

// TestCloseCampaignReturns429OnceTheAdministratorBudgetIsSpent is FR-053's other half,
// over the PRODUCTION limiter, so it also pins WHICH limiter is on the route: a second,
// campaign-only limiter would answer identically here, and the requirement is that there
// is not one (research.md R10).
func TestCloseCampaignReturns429OnceTheAdministratorBudgetIsSpent(t *testing.T) {
	stub := &registryStub{}
	engine := registryEngine(t, stub, authenticatedAs(roleslug.Admin), newAdminLimiter(neverAllowRateLimiter{}))

	rec := postClose(t, engine, "tet-2027", closeBody(t))

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("FR-053: an exhausted administrator budget must return 429, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.closeCalls != 0 {
		t.Errorf("FR-053: the close ran %d times behind an exhausted limit; it must not run at all",
			stub.closeCalls)
	}
}

// TestCloseCampaignNeverAnswersWithACodeOutsideTheTable is FR-054's closing rule for
// close: "không mã nào ngoài bảng này được trả về từ ba endpoint ghi".
//
// A 500 from a database fault and a 409 from a duplicate both satisfy the code that
// produced them while leaving the client unable to decide whether to change its input
// or retry — and 409 has NO meaning here at all: it belongs to create.
func TestCloseCampaignNeverAnswersWithACodeOutsideTheTable(t *testing.T) {
	allowed := map[int]string{
		http.StatusBadRequest:          "400 hỏng hình dạng",
		http.StatusNotFound:            "404 mã không tồn tại",
		http.StatusConflict:            "409 chỉ dành cho tạo",
		http.StatusUnprocessableEntity: "422 sai nghĩa",
		http.StatusPreconditionFailed:  "412 phiên bản lệch",
		http.StatusUnauthorized:        "401 chưa xác thực",
		http.StatusForbidden:           "403 không phải Admin",
		http.StatusTooManyRequests:     "429 vượt giới hạn",
		http.StatusInternalServerError: "500 lỗi hạ tầng",
	}

	for _, scripted := range []struct {
		name string
		err  error
	}{
		{"an unknown code", subscriptionerrors.ErrCampaignNotFound("khong-ton-tai")},
		{"a stale version", subscriptionerrors.ErrCampaignVersionConflict("tet-2027")},
		{"a blank reason", subscriptionerrors.ErrCampaignInvalidField("reason", "không được để trống")},
		{"a database fault", subscriptionerrors.ErrCampaignRegistryWriteFailed("tet-2027", errors.New("connection reset"))},
		{"a failed audit insert", subscriptionerrors.ErrCampaignRegistryAuditFailed("tet-2027", errors.New("outbox down"))},
	} {
		stub := &registryStub{closeErr: scripted.err}
		engine := newRegistryEngine(t, stub)

		rec := postClose(t, engine, "tet-2027", closeBody(t))

		if _, ok := allowed[rec.Code]; !ok {
			t.Errorf("FR-054: %s answered %d, which is not in the table (%s)", scripted.name, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "connection reset") || strings.Contains(rec.Body.String(), "outbox down") {
			t.Errorf("%s leaked the underlying cause to the client: %s", scripted.name, rec.Body.String())
		}
	}
}

// ---- the constants the audit row is built from ----

// TestCampaignAuditConstantsAreTheOnesThePayloadUses is FR-063's coupling, pinned.
//
// `adminaudit.Campaign` and `adminaudit.CampaignCreate` are the shared vocabulary the
// per-campaign audit endpoint filters on. A handler or use case that spelled "campaign"
// and "campaign.create" as literals would produce a row the read endpoint cannot find —
// and it would keep passing every behavioural test, because the row is still written.
func TestCampaignAuditConstantsAreTheOnesThePayloadUses(t *testing.T) {
	if string(adminaudit.Campaign) != "campaign" {
		t.Errorf("FR-063: the campaign target type must be %q, got %q", "campaign", adminaudit.Campaign)
	}
	if string(adminaudit.CampaignCreate) != "campaign.create" {
		t.Errorf("FR-060: the create action must be %q, got %q", "campaign.create", adminaudit.CampaignCreate)
	}
	// And the close action, which the per-campaign audit screen filters on exactly the
	// same way (FR-063, contract §6.3). A misspelling here would produce a record the
	// screen cannot find, while every behavioural test kept passing.
	if string(adminaudit.CampaignClose) != "campaign.close" {
		t.Errorf("FR-060: the close action must be %q, got %q", "campaign.close", adminaudit.CampaignClose)
	}
}

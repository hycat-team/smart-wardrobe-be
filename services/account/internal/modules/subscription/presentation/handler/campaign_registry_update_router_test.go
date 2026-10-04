package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 026 T058 (FR-020..FR-024, FR-052..FR-055) — `PATCH /api/v1/admin/campaigns/{code}`
// over HTTP.
//
// WHY A SEPARATE FILE AND NOT A SECTION OF `campaign_registry_router_test.go`. That
// file already carries the create and close contracts and sits above the test-file
// warning threshold (code-hygiene H1, which relaxes the limit by 50% for `*_test.go`).
// Appending a third operation's contract to it would push a file this feature already
// had to keep an eye on further past it, and the tests below share only the stub and the
// engine — both of which live in the sibling file and are used from here unchanged. The
// subject is the same: the HTTP surface, not the rules.
//
// TWO OF FR-054'S SEVEN CODES CANNOT BE REACHED BY CALLING THE USE CASE DIRECTLY, and
// that is why this goes through the router. 403 comes from the group's
// `RolesAuthorize` and 429 from the limiter, both of which run BEFORE the handler; a
// test that invoked `h.Update` would pass against an endpoint that lets any signed-in
// customer rewrite a campaign's budget. 400 is the third: `version` is
// `binding:"required"`, so a body without it is refused by the shared binder with the
// shared Vietnamese message, and that code is unreachable from a unit test.

// updateBody is a well-formed edit as a client would send it: one field, the version it
// holds, and nothing else.
func updateBody(t *testing.T) *bytes.Reader {
	t.Helper()
	return rawJSONBody(t, map[string]any{"version": 3, "quota": 120})
}

// rawJSONBody marshals `body` into a reader, so every case below declares its payload
// as data rather than as a hand-written string.
func rawJSONBody(t *testing.T, body map[string]any) *bytes.Reader {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return bytes.NewReader(raw)
}

// patchCampaign sends one edit for the given code.
func patchCampaign(t *testing.T, engine *gin.Engine, code string, body *bytes.Reader) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, campaignPath+"/"+code, body)
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)
	return rec
}

// ---- 200 ----

// TestUpdateCampaignReturns200WithTheUpdatedSummary is the success path.
//
// 200 AND NOT 201 — nothing was created, and a 201 would tell an operator's screen that
// a second campaign now exists. The body must be the campaign as the write left it,
// which is the assertion that matters: the response carries `version` 4, the value the
// write PRODUCED, while the request carried 3. A handler echoing the request would
// hand the client a token the row no longer has, and its next edit would be refused
// with 412 for its own successful change (FR-055).
func TestUpdateCampaignReturns200WithTheUpdatedSummary(t *testing.T) {
	stub := &registryStub{updateResult: &sub_dto.CampaignSummaryRes{
		Code:         "tet-2027",
		PlanSlug:     "premium-monthly",
		Quota:        120,
		Reserve:      10,
		HardCap:      130,
		Status:       sub_dto.CampaignStatusRunning,
		Version:      4,
		BudgetLocked: false,
	}}
	engine := newRegistryEngine(t, stub)

	rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

	if rec.Code != http.StatusOK {
		t.Fatalf("contract §4.4: a successful edit must return 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if rec.Code == http.StatusCreated {
		t.Error("an edit creates nothing; 201 would tell the client's screen a campaign was created")
	}
	data := createdResponse(t, rec)
	if data.Code != "tet-2027" || data.Quota != 120 || data.HardCap != 130 {
		t.Errorf("the 200 body must be the edited campaign, got %+v", data)
	}
	if data.Version != 4 {
		t.Errorf("the response must carry the version the write produced (4), got %d", data.Version)
	}
}

// TestUpdateCampaignAnswers200ForTheNoOpToo is FR-055's exception at the HTTP surface.
//
// The handler CANNOT tell a no-op from a real edit and must not need to: both are "the
// campaign is in the state you asked for". What it must NOT do is answer an error for
// the no-op, because the client cannot tell one from a real conflict either — and a
// screen that resubmits an unchanged form would then show the operator a failure for an
// edit nobody made.
func TestUpdateCampaignAnswers200ForTheNoOpToo(t *testing.T) {
	// The stub returns the campaign unchanged at the version it already had, which is
	// the shape of FR-055's no-op answer.
	stub := &registryStub{updateResult: &sub_dto.CampaignSummaryRes{
		Code: "tet-2027", Quota: 120, Version: 4,
	}}
	engine := newRegistryEngine(t, stub)

	rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

	if rec.Code != http.StatusOK {
		t.Errorf("FR-055: a PATCH that changes nothing must still answer 200, got %d (%s)",
			rec.Code, rec.Body.String())
	}
	if createdResponse(t, rec).Version != 4 {
		t.Error("the no-op must report the version the row already has, so the next edit is not stale")
	}
}

// TestUpdateCampaignReachesTheUseCaseWithThePathCodeAndTheActor is FR-060's input half
// for the edit.
//
// `code` is not in the body (FR-024), so the URL segment is the only way a code can
// arrive — and the audit row is worthless if it names the wrong campaign. `version` and
// the bound numbers are asserted too, because a handler that defaulted any of them
// would answer 412 for every request while every other test here stayed green.
func TestUpdateCampaignReachesTheUseCaseWithThePathCodeAndTheActor(t *testing.T) {
	stub := &registryStub{}
	engine := newRegistryEngine(t, stub)

	rec := patchCampaign(t, engine, "launch-2026-10", rawJSONBody(t, map[string]any{
		"version": 7, "quota": 120, "reserve": 10, "planSlug": "premium-yearly",
	}))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.updateCalls != 1 {
		t.Fatalf("expected the use case to be called once, got %d", stub.updateCalls)
	}
	if stub.lastUpdateCode != "launch-2026-10" {
		t.Errorf("the handler must pass the :code segment through, got %q", stub.lastUpdateCode)
	}
	if stub.lastActor == uuid.Nil {
		t.Error("FR-060: the handler must pass the authenticated administrator, not a zero id")
	}
	if stub.lastIP == nil || *stub.lastIP == "" {
		t.Error("FR-060: the handler must pass the request address so a disputed edit can be traced")
	}
	if stub.lastUpdateReq.Version != 7 {
		t.Errorf("version = %d, want the 7 the client sent", stub.lastUpdateReq.Version)
	}
	if stub.lastUpdateReq.Quota == nil || *stub.lastUpdateReq.Quota != 120 {
		t.Errorf("quota must arrive as the number the client sent, got %+v", stub.lastUpdateReq.Quota)
	}
	if stub.lastUpdateReq.Reserve == nil || *stub.lastUpdateReq.Reserve != 10 {
		t.Errorf("reserve must arrive as the number the client sent, got %+v", stub.lastUpdateReq.Reserve)
	}
	if stub.lastUpdateReq.PlanSlug == nil || *stub.lastUpdateReq.PlanSlug != "premium-yearly" {
		t.Errorf("planSlug must arrive as the string the client sent, got %+v", stub.lastUpdateReq.PlanSlug)
	}
}

// TestUpdateCampaignIgnoresACodeInTheBody is FR-024 over HTTP.
//
// A client that believes it can rename sends `code` alongside a real edit. The edit must
// still be applied AND the campaign must keep its code — which is only provable by
// asserting both, because a handler that dropped the whole body on an unknown field
// would pass "the code did not change" while silently ignoring the operator's edit.
func TestUpdateCampaignIgnoresACodeInTheBody(t *testing.T) {
	stub := &registryStub{}
	engine := newRegistryEngine(t, stub)

	rec := patchCampaign(t, engine, "tet-2027", rawJSONBody(t, map[string]any{
		"version": 3, "code": "tet-2028", "quota": 120,
	}))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.lastUpdateCode != "tet-2027" {
		t.Errorf("the path's code must win, got %q", stub.lastUpdateCode)
	}
	if stub.lastUpdateReq.Quota == nil || *stub.lastUpdateReq.Quota != 120 {
		t.Error("the rest of the body must still be applied — a `code` key must not cost the operator their edit")
	}
}

// ---- FR-054: 400 ----

// TestUpdateCampaignRejectsBrokenShapesWith400 is the 400 row of contract §4.4, and
// all of it is the BINDER's.
//
// `version` is `binding:"required"`, and nothing else about the body can be unreadable
// — every other field is optional and typed. The stronger half of each case is that the
// use case was NEVER reached: a request that could not be read must not have consulted
// the database, compared a version or opened a transaction.
func TestUpdateCampaignRejectsBrokenShapesWith400(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
	}{
		{"a missing version", map[string]any{"quota": 120}},
		{"an empty body", map[string]any{}},
		{"a version of zero", map[string]any{"version": 0, "quota": 120}},
		{"a version that is not a number", map[string]any{"version": "three", "quota": 120}},
		{"a quota that is not a number", map[string]any{"version": 3, "quota": "many"}},
		{"a planSlug that is not a string", map[string]any{"version": 3, "planSlug": 42}},
		{"a startsAt that is not a string", map[string]any{"version": 3, "startsAt": 1700000000}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registryStub{}
			engine := newRegistryEngine(t, stub)

			rec := patchCampaign(t, engine, "tet-2027", rawJSONBody(t, tc.body))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("contract §4.4: expected 400, got %d (%s)", rec.Code, rec.Body.String())
			}
			if message := campaignErrorMessage(t, rec); message == "" {
				t.Errorf("a 400 must carry a Vietnamese message, got %q", rec.Body.String())
			}
			if stub.updateCalls != 0 {
				t.Errorf("a body the binder refuses must never reach the use case, got %d calls",
					stub.updateCalls)
			}
		})
	}
}

// ---- FR-054: 404, 412, 422 ----

// TestUpdateCampaignReturns404412And422 is FR-054's remaining rows for the edit, in the
// order contract §4.4 lists them.
//
// Each is a DIFFERENT mistake with a different remedy — a mistyped code, a page that has
// been open too long, and an operator editing the budget of a campaign that already
// granted — so answering the wrong one sends them to do the wrong next thing. 412 in
// particular is not fixable by editing the form: the page they were looking at is not
// the current state.
func TestUpdateCampaignReturns404412And422(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
		// mentions is a fragment the message must carry, so the operator is told what to
		// do rather than only which rule fired.
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
			name:     "a zero quota",
			err:      subscriptionerrors.ErrCampaignInvalidField("quota", "phải lớn hơn hoặc bằng 1"),
			want:     http.StatusUnprocessableEntity,
			mentions: "quota",
		},
		{
			// US3 scenario 2: the refusal has to name the way out, or the operator edits
			// the same form again.
			name:     "a budget change on a campaign that has granted",
			err:      subscriptionerrors.ErrCampaignBudgetLocked("tet-2027"),
			want:     http.StatusUnprocessableEntity,
			mentions: "mở",
		},
		{
			name:     "a zone-less timestamp",
			err:      subscriptionerrors.ErrCampaignTimestampMalformed("startsAt"),
			want:     http.StatusBadRequest,
			mentions: "startsAt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registryStub{updateErr: tc.err}
			engine := newRegistryEngine(t, stub)

			rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

			if rec.Code != tc.want {
				t.Fatalf("expected %d, got %d (%s)", tc.want, rec.Code, rec.Body.String())
			}
			if message := campaignErrorMessage(t, rec); !strings.Contains(message, tc.mentions) {
				t.Errorf("the message must carry %q, got %q", tc.mentions, message)
			}
			// The use case DID run here, unlike the 400 cases: it is the layer that knows
			// the code does not exist, the version moved, or the budget is frozen.
			if stub.updateCalls != 1 {
				t.Errorf("expected the use case to be reached once, got %d", stub.updateCalls)
			}
		})
	}
}

// ---- FR-054: 403 and 429 ----

// TestUpdateCampaignIsRefusedForANormalAccount is FR-053 applied to the edit.
//
// Editing the budget decides how many plans a promotion can still hand out, so "an
// ordinary signed-in customer must not reach it" is the assertion that matters. It
// happens in `RolesAuthorize`, before the handler runs — which is exactly why it can
// only be observed through the route.
func TestUpdateCampaignIsRefusedForANormalAccount(t *testing.T) {
	stub := &registryStub{}
	engine := registryEngine(t, stub, authenticatedAs(roleslug.User), newAdminLimiter(alwaysAllowRateLimiter{}))

	rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

	if rec.Code != http.StatusForbidden {
		t.Errorf("FR-053: a normal account must be refused with 403, got %d (%s)", rec.Code, rec.Body.String())
	}
	if stub.updateCalls != 0 {
		t.Errorf("FR-053: the edit ran %d times for a refused caller; it must not run at all", stub.updateCalls)
	}
}

// TestUpdateCampaignReturns429OnceTheAdministratorBudgetIsSpent is FR-053's other half,
// over the PRODUCTION limiter, so it also pins WHICH limiter is on the route: a second,
// campaign-only limiter would answer identically, and the requirement is that there is
// not one (research.md R10).
func TestUpdateCampaignReturns429OnceTheAdministratorBudgetIsSpent(t *testing.T) {
	stub := &registryStub{}
	engine := registryEngine(t, stub, authenticatedAs(roleslug.Admin), newAdminLimiter(neverAllowRateLimiter{}))

	rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("FR-053: an exhausted administrator budget must return 429, got %d (%s)",
			rec.Code, rec.Body.String())
	}
	if stub.updateCalls != 0 {
		t.Errorf("FR-053: the edit ran %d times behind an exhausted limit; it must not run at all",
			stub.updateCalls)
	}
}

// ---- the codes FR-054 forbids ----

// TestUpdateCampaignNeverAnswersWithACodeOutsideTheTable is FR-054's closing rule for
// the edit: "không mã nào ngoài bảng này được trả về từ ba endpoint ghi".
//
// A 500 from a database fault satisfies the code that produced it while leaving the
// client unable to decide whether to change its input or retry — and the raw cause must
// not travel either, or an operator learns the schema instead of what to do.
func TestUpdateCampaignNeverAnswersWithACodeOutsideTheTable(t *testing.T) {
	allowed := map[int]string{
		http.StatusBadRequest:          "400 hỏng hình dạng",
		http.StatusNotFound:            "404 mã không tồn tại",
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
		{"a frozen budget", subscriptionerrors.ErrCampaignBudgetLocked("tet-2027")},
		{"a database fault", subscriptionerrors.ErrCampaignRegistryWriteFailed("tet-2027", errors.New("connection reset"))},
		{"a failed audit insert", subscriptionerrors.ErrCampaignRegistryAuditFailed("tet-2027", errors.New("outbox down"))},
		{"a registry read fault", subscriptionerrors.ErrCampaignRegistryReadFailed()},
	} {
		stub := &registryStub{updateErr: scripted.err}
		engine := newRegistryEngine(t, stub)

		rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

		if _, ok := allowed[rec.Code]; !ok {
			t.Errorf("FR-054: %s answered %d, which is not in the table (%s)",
				scripted.name, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "connection reset") || strings.Contains(rec.Body.String(), "outbox down") {
			t.Errorf("%s leaked the underlying cause to the client: %s", scripted.name, rec.Body.String())
		}
	}
}

// TestUpdateCampaignNeverAnswers409 pins FR-054's one ASYMMETRY, and it is stated as a
// test because it is the rule most likely to be broken by copying code between the
// endpoints.
//
// `409` belongs to CREATE ONLY — a duplicate code at insert time. An edit names an
// existing campaign by a primary key and inserts nothing, so there is nothing for 409
// to mean here. The handler propagates whatever the use case returns, so the guarantee
// is made of the REQUEST TYPE: `CampaignUpdateReq` carries no code, therefore no
// duplicate can be produced, therefore no duplicate-code error can reach this route.
// If somebody adds a `Code` field back (FR-024) the reflection assertion below fails
// in the same breath that the 409 becomes reachable.
func TestUpdateCampaignNeverAnswers409(t *testing.T) {
	reqType := reflect.TypeOf(sub_dto.CampaignUpdateReq{})
	if _, found := reqType.FieldByName("Code"); found {
		t.Errorf("FR-024/FR-054: `CampaignUpdateReq` carries a Code again. An edit that could name a " +
			"code could produce a duplicate, and 409 is create-only")
	}

	for _, scripted := range []struct {
		name string
		err  error
	}{
		{"a frozen budget", subscriptionerrors.ErrCampaignBudgetLocked("tet-2027")},
		{"an unknown code", subscriptionerrors.ErrCampaignNotFound("khong-ton-tai")},
		{"a stale version", subscriptionerrors.ErrCampaignVersionConflict("tet-2027")},
		{"a zero quota", subscriptionerrors.ErrCampaignInvalidField("quota", "phải lớn hơn hoặc bằng 1")},
	} {
		stub := &registryStub{updateErr: scripted.err}
		engine := newRegistryEngine(t, stub)

		rec := patchCampaign(t, engine, "tet-2027", updateBody(t))

		if rec.Code == http.StatusConflict {
			t.Errorf("FR-054: 409 is create-only, and %s must not produce it on an edit (%s)",
				scripted.name, rec.Body.String())
		}
	}
}

// TestUpdateCampaignAuditConstantsAreTheOnesThePayloadUses is FR-063's coupling, the
// edit's half of it.
//
// `adminaudit.CampaignUpdate` is the shared vocabulary the per-campaign audit endpoint
// filters on. A use case that spelled "campaign.update" as a literal would write a row
// the screen cannot find — and every behavioural test would still pass, because the row
// IS written.
func TestUpdateCampaignAuditConstantsAreTheOnesThePayloadUses(t *testing.T) {
	if string(adminaudit.CampaignUpdate) != "campaign.update" {
		t.Errorf("FR-060: the update action must be %q, got %q", "campaign.update", adminaudit.CampaignUpdate)
	}
	// And it must be DISTINCT from the two actions that already exist: an audit screen
	// that cannot tell an edit from a creation cannot answer "what changed", which is
	// the whole reason the trail exists.
	if adminaudit.CampaignUpdate == adminaudit.CampaignCreate {
		t.Error("FR-060: `campaign.update` must differ from `campaign.create`")
	}
	if adminaudit.CampaignUpdate == adminaudit.CampaignClose {
		t.Error("FR-060: `campaign.update` must differ from `campaign.close`")
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// 026 T071 (FR-053, FR-063, FR-072, contract §6) — `GET /admin/campaigns/{code}/audit`
// over HTTP.
//
// WHY THESE GO THROUGH THE ROUTER, in the same way the four 025 read endpoints do: two of
// the four answers here are produced by middleware that runs BEFORE the handler. A normal
// customer is refused with 403 and an exhausted budget with 429, and neither refusal is
// the handler's to make. Calling `h.GetCampaignAudit` directly would skip both, and every
// test below would pass against an endpoint that hands the audit trail of a live
// promotion to any signed-in customer.
//
// WHAT IS TESTED HERE AND WHAT IS NOT. The wiring: that the route carries the role check
// and the shared limiter, that an unknown code is a 404 with a Vietnamese message rather
// than an empty page, and that `targetId` is EMITTED as `null` rather than dropped — the
// one field-level fact that cannot be asserted anywhere else, because it is decided by a
// struct tag on a response type. The rules behind the trail are tested in
// `registry_uc_close_test.go`, `registry_uc_update_test.go` and
// `campaign_db_audit_read_integration_test.go`.

// auditPath is the route under test, built from the registered base path so a change to
// the prefix cannot leave this file pointing at a path that no longer exists.
func auditPath(code string) string { return campaignPath + "/" + code + "/audit" }

// auditRow is one trail row as the fake use case returns it.
func auditRow(action string, payload map[string]any) *sub_dto.CampaignAuditLogRes {
	return &sub_dto.CampaignAuditLogRes{
		ID:          uuid.New(),
		ActorUserID: uuid.New(),
		Action:      action,
		TargetType:  string(adminaudit.Campaign),
		// Intentionally nil: FR-063. EVERY campaign row has a null `target_id`, and this
		// is the only layer where that can be turned into JSON at all.
		TargetID:  nil,
		Payload:   payload,
		CreatedAt: time.Date(2026, 10, 3, 10, 15, 0, 0, time.UTC),
	}
}

// TestCampaignAuditReturnsTheTrail is contract §6.2 and FR-063's happy path.
//
// THE ENVELOPE IS ASSERTED, not just the status: the page has to be a paginated result
// whose `metadata` describes the page, because a client rendering the screen needs the
// total to know whether there is more — and an endpoint that returned a bare array would
// make "is there a second page" unanswerable.
func TestCampaignAuditReturnsTheTrail(t *testing.T) {
	stub := &stubCampaignQuery{
		audit: map[string][]*sub_dto.CampaignAuditLogRes{
			"launch-2026-10": {
				auditRow(string(adminaudit.CampaignCreate), map[string]any{
					"campaignCode": "launch-2026-10",
				}),
				auditRow(string(adminaudit.CampaignUpdate), map[string]any{
					"campaignCode": "launch-2026-10",
					"before":       map[string]any{"quota": 50},
					"after":        map[string]any{"quota": 60},
				}),
			},
		},
		auditTotal: 2,
	}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Message string `json:"message"`
		Data    struct {
			Items    []sub_dto.CampaignAuditLogRes `json:"items"`
			Metadata struct {
				Page       int   `json:"page"`
				Limit      int   `json:"limit"`
				TotalItems int64 `json:"totalItems"`
				TotalPages int   `json:"totalPages"`
			} `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("parse response %q: %v", rec.Body.String(), err)
	}
	if envelope.Message != successAdminGetCampaignAudit {
		t.Errorf("message = %q, want %q", envelope.Message, successAdminGetCampaignAudit)
	}
	if len(envelope.Data.Items) != 2 {
		t.Fatalf("expected the two rows the stub holds, got %d", len(envelope.Data.Items))
	}
	if envelope.Data.Metadata.TotalItems != 2 || envelope.Data.Metadata.TotalPages != 1 {
		t.Errorf("the page metadata must describe the page: %+v", envelope.Data.Metadata)
	}
	if envelope.Data.Metadata.Page != 1 || envelope.Data.Metadata.Limit != 20 {
		t.Errorf("an unparameterised request must be normalised to page 1 / limit 20, got %d / %d",
			envelope.Data.Metadata.Page, envelope.Data.Metadata.Limit)
	}
	// FR-063's three constants, on the wire rather than in Go: a client switches on these
	// strings, so they are part of the contract and not an implementation detail.
	for _, item := range envelope.Data.Items {
		if item.TargetType != string(adminaudit.Campaign) {
			t.Errorf("FR-063: targetType = %q, want %q", item.TargetType, adminaudit.Campaign)
		}
		if item.Action != string(adminaudit.CampaignCreate) &&
			item.Action != string(adminaudit.CampaignUpdate) {
			t.Errorf("FR-063: unexpected action %q on the campaign trail", item.Action)
		}
		if got, ok := item.Payload["campaignCode"]; !ok || got != "launch-2026-10" {
			t.Errorf("FR-063: payload.campaignCode = %v, want \"launch-2026-10\"", got)
		}
	}
}

// TestCampaignAuditEmitsNullTargetIdRatherThanOmittingIt is the field-level reason
// `CampaignAuditLogRes` exists at all, and it is asserted on the RAW BODY because that is
// the only place the difference is observable.
//
// `dto/admin.AdminAuditLogDTO` declares `json:"targetId,omitempty"`, so reusing it would
// have DROPPED the key entirely — and an operator could no longer tell "every campaign row
// has a null target id, because the code travels in the payload" from "this response
// forgot to include one". The contract's §6.2 example shows `"targetId": null`, and this
// is the assertion that the example is real.
//
// THE STRING COMPARISON IS DELIBERATE. Decoding into the DTO would prove nothing: Go
// fills a missing key and an explicit null into the same nil pointer, which is exactly the
// ambiguity the type exists to avoid.
func TestCampaignAuditEmitsNullTargetIdRatherThanOmittingIt(t *testing.T) {
	stub := &stubCampaignQuery{
		audit: map[string][]*sub_dto.CampaignAuditLogRes{
			"launch-2026-10": {auditRow(string(adminaudit.CampaignClose), map[string]any{
				"campaignCode": "launch-2026-10",
				"reason":       "Hết ngân sách khuyến mãi tháng 11",
			})},
		},
		auditTotal: 1,
	}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !containsAny(body, `"targetId":null`) {
		t.Errorf("FR-063/contract §6.2: the trail must EMIT targetId as null, not omit it: %s", body)
	}
	if containsAny(body, `"targetId":null,"requestIp"`) == false && containsAny(body, `"reason"`) {
		// A row with no request IP omits `requestIp` and nothing else; this branch only
		// fires to keep the intent readable, since the assertion above already covers it.
		t.Log("trail row carries no requestIp, as expected when the caller sent none")
	}
}

// TestCampaignAuditUnknownCodeIs404NotAnEmptyPage is FR-072 and contract §6.5.
//
// AN EMPTY PAGE IS THE FAILURE MODE THIS EXCLUDES, and it is not a stylistic preference.
// The trail lives in the `admin` module, which does not own `subscription.campaigns` and
// must not read it (constitution §II), so IT answers an unknown code with zero rows and a
// total of zero — indistinguishable from "nobody has ever touched this campaign". An
// operator reading that would conclude the audit mechanism is broken rather than that they
// typed a code wrong.
//
// THE ORDERING THAT PREVENTS IT IS NOT ASSERTED HERE, and the reason is the stub: this
// test stands in for the whole USE CASE, so it cannot show whether the use case consulted
// the registry before the trail — it can only report that it was consulted once, which is
// what the stub did to produce the 404. That assertion lives where the real use case runs,
// in `campaign_db_audit_read_integration_test.go`, over the real registry and a recording
// audit contract.
func TestCampaignAuditUnknownCodeIs404NotAnEmptyPage(t *testing.T) {
	const unknownCode = "khong-ton-tai"
	stub := &stubCampaignQuery{audit: map[string][]*sub_dto.CampaignAuditLogRes{}}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet, auditPath(unknownCode))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("FR-072: %s must return 404, got %d (%s)", unknownCode, rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if containsAny(body, `"items":[]`, `"items": []`) {
		t.Errorf("FR-072: an unknown campaign code must not answer with an empty page: %s", body)
	}
	// A Vietnamese message that NAMES the code, so an operator who typed it wrong knows
	// what to correct (constitution: client-facing text is Vietnamese).
	message := campaignErrorMessage(t, rec)
	if message == "" {
		t.Fatalf("the 404 must carry a message, got %q", body)
	}
	if !containsAny(message, unknownCode) {
		t.Errorf("the message must name the code that was asked for, got %q", message)
	}
}

// TestCampaignAuditSurfacesATrailReadFailure is the other half of not-answering-with-an
// empty-page, and it is the case a stub exists for.
//
// A FAILED READ ANSWERED AS AN EMPTY TRAIL would say "nobody changed this campaign" to an
// operator whose next action is to close it. The use case has to surface the failure, and
// the message has to name the campaign they were looking at — the `admin` module cannot
// produce that wording because it does not know what a campaign is.
func TestCampaignAuditSurfacesATrailReadFailure(t *testing.T) {
	stub := &stubCampaignQuery{
		audit:    map[string][]*sub_dto.CampaignAuditLogRes{},
		auditErr: subscriptionerrors.ErrCampaignRegistryReadFailed(),
	}
	engine := newCampaignEngine(t, stub)

	rec := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("a failed trail read must be a 500, got %d (%s)", rec.Code, rec.Body.String())
	}
	if containsAny(rec.Body.String(), `"items":[]`, `"items": []`) {
		t.Errorf("a failed read must not be rendered as an empty page: %s", rec.Body.String())
	}
	if !containsAny(campaignErrorMessage(t, rec), "chiến dịch") {
		t.Errorf("the failure message must name the campaign in Vietnamese, got %q",
			campaignErrorMessage(t, rec))
	}
}

// TestCampaignAuditIsRefusedForANormalUser is FR-053 and contract §6.5, and the audit
// trail is the most sensitive thing on this handler: it says WHICH ADMINISTRATOR changed
// a live promotion.
//
// The refusal happens in `RolesAuthorize` before the handler runs, which is exactly why
// this test goes through the route: called directly, the handler would answer 200 and the
// leak would be invisible.
func TestCampaignAuditIsRefusedForANormalUser(t *testing.T) {
	stub := &stubCampaignQuery{
		audit: map[string][]*sub_dto.CampaignAuditLogRes{
			"launch-2026-10": {auditRow(string(adminaudit.CampaignUpdate), map[string]any{
				"campaignCode": "launch-2026-10",
			})},
		},
		auditTotal: 1,
	}
	engine := campaignEngine(t, stub, authenticatedAs(roleslug.User),
		newAdminLimiter(alwaysAllowRateLimiter{}))

	rec := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))

	if rec.Code != http.StatusForbidden {
		t.Errorf("FR-053: a normal account must be refused with 403, got %d (%s)", rec.Code,
			rec.Body.String())
	}
	// The refusal must not leak the trail: the row names the administrator and the change.
	if containsAny(rec.Body.String(), "campaignCode", "actorUserId", "launch-2026-10\"") {
		t.Errorf("FR-053: the refusal leaked audit data to a normal account: %s", rec.Body.String())
	}
	// And nothing was even read, which is the stronger statement.
	if stub.auditCalls != 0 {
		t.Errorf("FR-053: the trail was read %d times behind a refusal; it must not be read at all",
			stub.auditCalls)
	}
}

// TestCampaignAuditReturns429OnceTheAdministratorBudgetIsSpent is FR-053's second half and
// the reason the route reuses `adminGrantRateLimit()` rather than a limiter of its own.
//
// Asserting the 429 also pins WHICH limiter is on the route: a second, trail-only limiter
// would answer identically here, and FR-053's requirement is that there is not one — a
// second configuration for the same limit is a second answer waiting to be given (R10).
func TestCampaignAuditReturns429OnceTheAdministratorBudgetIsSpent(t *testing.T) {
	stub := &stubCampaignQuery{
		audit: map[string][]*sub_dto.CampaignAuditLogRes{
			"launch-2026-10": {auditRow(string(adminaudit.CampaignCreate), nil)},
		},
		auditTotal: 1,
	}
	engine := campaignEngine(t, stub, authenticatedAs(roleslug.Admin),
		newAdminLimiter(neverAllowRateLimiter{}))

	rec := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("FR-053: %s must return 429 once the administrator's budget is spent, got %d (%s)",
			auditPath("launch-2026-10"), rec.Code, rec.Body.String())
	}
	if stub.auditCalls != 0 {
		t.Errorf("FR-053: the trail was read %d times behind an exhausted limit; it must not be read at all",
			stub.auditCalls)
	}
}

// TestCampaignAuditIsReadOnlyOverFiftyCalls is FR-069 applied to the new endpoint.
//
// The read-only guarantee has to cover the endpoint added in 026 as well as the four 025
// added, and "reading the audit trail" is precisely the operation somebody might expect to
// have a side effect: it is called AUDIT, it is about changes, and a reader could easily
// believe that consulting it settles something. Fifty calls must produce byte-identical
// bodies and leave the trail's length alone.
func TestCampaignAuditIsReadOnlyOverFiftyCalls(t *testing.T) {
	stub := &stubCampaignQuery{
		audit: map[string][]*sub_dto.CampaignAuditLogRes{
			"launch-2026-10": {auditRow(string(adminaudit.CampaignUpdate), map[string]any{
				"campaignCode": "launch-2026-10",
			})},
		},
		auditTotal: 1,
	}
	engine := newCampaignEngine(t, stub)

	first := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", first.Code, first.Body.String())
	}
	baseline := first.Body.String()

	for i := 0; i < 50; i++ {
		rec := request(t, engine, http.MethodGet, auditPath("launch-2026-10"))
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d (%s)", i, rec.Code, rec.Body.String())
		}
		if rec.Body.String() != baseline {
			t.Fatalf("FR-069: call %d returned a different body from call 0.\n first: %s\n  call %d: %s",
				i, baseline, i, rec.Body.String())
		}
	}
	if stub.auditTotal != 1 {
		t.Errorf("FR-069: reading the trail must not change what it reports, got total=%d", stub.auditTotal)
	}
}

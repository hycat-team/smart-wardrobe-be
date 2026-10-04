package campaign

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
)

// 026 T056 (FR-020, FR-021, FR-022, FR-023, FR-055, FR-060, FR-061) —
// `CampaignRegistryUseCase.Update`.
//
// WHAT A UNIT TEST CAN PROVE HERE AND WHAT IT CANNOT. Every assertion below is a
// DECISION: which error the operator is told, in which ORDER, whether the row was
// written, and whether exactly one audit command was issued inside the transaction.
// It cannot prove that the trigger refuses a hand-written `UPDATE`, that
// `version + 1` is computed by PostgreSQL, or that a real conditional UPDATE changes
// zero rows on a stale version — those are the database's, and they belong to
// `campaign_guard_integration_test.go` (T057) and `campaign_repo_integration_test.go`
// (T022).
//
// THE ORDER TESTS ARE THE POINT. `Update` answers 404, 200-no-op, 422 and
// 412-and-200, and three of the four are reachable from the same request shape; a use
// case that consulted them in a different order would still pass a test of each code
// taken alone. So what matters is pinning which ANSWER a request gets.

// ---- fixtures ----

// validUpdateReq is a request that passes every rule and changes exactly one field,
// so a test can break one thing and know the failure it is reading.
func validUpdateReq(version int) *sub_dto.CampaignUpdateReq {
	quota := 120
	return &sub_dto.CampaignUpdateReq{Version: version, Quota: &quota}
}

// seededUpdateFixture returns a fixture whose registry already holds `campaign`, in
// the state every update starts from: a campaign that EXISTS and is open.
func seededUpdateFixture(t *testing.T, campaign *entities.Campaign) *registryFixture {
	t.Helper()
	f := newRegistryFixture()
	f.repo.seedRow(campaign)
	return f
}

// ---- US3 scenario 1: editable before the first grant ----

// TestUpdateChangesQuotaReserveAndPlanSlugBeforeTheFirstGrant is US3 acceptance
// scenario 1, and it is the positive half of FR-022: with no grant recorded, all
// three budget columns are the operator's to change.
//
// THE RESPONSE IS THE STORED ROW, not the request. `version` is asserted as 2 rather
// than as the 1 the client sent, because `version + 1` is the STATEMENT's arithmetic
// and the only honest source for it is the row read back afterwards (FR-055). A use
// case that echoed the request would hand the client a token that is already stale,
// and its next edit would be refused with 412 for its own successful change.
func TestUpdateChangesQuotaReserveAndPlanSlugBeforeTheFirstGrant(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	actor := uuid.New()

	planSlug := "premium-yearly"
	req := validUpdateReq(1)
	req.PlanSlug = &planSlug
	req.Reserve = intPtr(25)

	summary, err := f.uc.Update(context.Background(), actor, "tet-2027", req, nil)
	if err != nil {
		t.Fatalf("FR-022: a campaign nobody was granted from must be editable, got %v", err)
	}

	row := f.repo.row("tet-2027")
	if row.Quota != 120 || row.Reserve != 25 || row.PlanSlug != "premium-yearly" {
		t.Errorf("the stored row did not take the edit: %+v", row)
	}
	// FR-055: exactly one increment, and it is the STATEMENT's, so it appears on the
	// response as the version the next write must carry.
	if row.Version != 2 {
		t.Errorf("version = %d, want it incremented to 2 by the statement", row.Version)
	}
	if summary.Version != 2 {
		t.Errorf("the response must carry the version the WRITE produced, got %d, want 2", summary.Version)
	}
	// US3.1 spells this out: `budgetLocked` is still false, because nothing was
	// granted. On a write response the counts are zero (the writer holds no ledger
	// repository), and FR-020's condition is derived from them — so this is asserting
	// the derivation, not the ledger.
	if summary.BudgetLocked {
		t.Error("US3.1: budgetLocked must still be false before the first grant")
	}
	if summary.Quota != 120 || summary.Reserve != 25 || summary.PlanSlug != "premium-yearly" {
		t.Errorf("the response does not describe the campaign that was edited: %+v", summary)
	}

	// The version the CLIENT sent is the version COMPARED — the same assertion the
	// close tests make, and for the same reason: comparing a version of the use case's
	// own would make the conditional UPDATE unconditional.
	if len(f.repo.updateVersions) != 1 || f.repo.updateVersions[0] != 1 {
		t.Errorf("expected exactly one conditional UPDATE compared against version 1, got %v",
			f.repo.updateVersions)
	}

	// Exactly one transaction (constitution §V).
	if f.uow.entries != 1 {
		t.Errorf("Update entered %d transactions; constitution §V allows exactly one per operation", f.uow.entries)
	}

	// FR-060/FR-063: exactly one audit row, findable by the per-campaign endpoint.
	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("FR-060: expected exactly one audit command, got %d", len(records))
	}
	record := records[0]
	if record.Action != string(adminaudit.CampaignUpdate) {
		t.Errorf("action = %q, want %q", record.Action, adminaudit.CampaignUpdate)
	}
	if record.TargetType != string(adminaudit.Campaign) {
		t.Errorf("FR-063: targetType = %q, want %q", record.TargetType, adminaudit.Campaign)
	}
	if record.TargetID != nil {
		t.Errorf("FR-063: targetId must be nil for a campaign record, got %v", *record.TargetID)
	}
	if got := record.Payload["campaignCode"]; got != "tet-2027" {
		t.Errorf("FR-063: payload.campaignCode = %v, want \"tet-2027\"", got)
	}
	if record.actor != actor {
		t.Errorf("the audit actor = %s, want the administrator %s", record.actor, actor)
	}
}

// TestUpdateAuditCarriesOnlyTheFieldsThatMoved is FR-060's "old → new" half, and the
// "ONLY" is the requirement.
//
// A payload repeating the whole campaign would leave the reader of the audit screen
// computing the diff themselves, which is the work the trail exists to have done —
// and it would put an unchanged value on both sides of a comparison, so a field that
// was left alone would be indistinguishable from one that was rewritten with its own
// value.
func TestUpdateAuditCarriesOnlyTheFieldsThatMoved(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())

	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil); err != nil {
		t.Fatalf("the edit must succeed: %v", err)
	}

	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("precondition: one audit row expected, got %d", len(records))
	}
	before, okBefore := records[0].Payload["before"].(map[string]any)
	after, okAfter := records[0].Payload["after"].(map[string]any)
	if !okBefore || !okAfter {
		t.Fatalf("the update audit payload must carry before and after, got %#v", records[0].Payload)
	}

	if before["quota"] != 100 || after["quota"] != 120 {
		t.Errorf("quota must travel as %d → %d, got %v → %v", 100, 120, before["quota"], after["quota"])
	}
	// `version` is on both sides even though nobody typed it: it moved, and an audit
	// line that could not say which revision of the row it describes would be unusable
	// for the concurrent-edit question FR-055 exists to answer.
	if before["version"] != 1 || after["version"] != 2 {
		t.Errorf("version must travel as 1 → 2, got %v → %v", before["version"], after["version"])
	}
	// The two fields the request did NOT send must be absent, not echoed.
	if _, present := before["reserve"]; present {
		t.Errorf("an unchanged field must be absent from before, got %#v", before)
	}
	if _, present := after["reserve"]; present {
		t.Errorf("an unchanged field must be absent from after, got %#v", after)
	}
	if _, present := after["planSlug"]; present {
		t.Errorf("an unchanged field must be absent from after, got %#v", after)
	}
	// And `reason` belongs to close alone (contract §6.3), never to an edit.
	if _, present := records[0].Payload["reason"]; present {
		t.Errorf("an update record must not carry a closure reason, got %#v", records[0].Payload)
	}
}

// ---- FR-023: an omitted field keeps its stored value ----

// TestUpdateKeepsEveryFieldTheRequestDidNotSend is the merge rule, and it is asserted
// through the ROW rather than through the payload because the row is what the grants
// read.
//
// The reason it needs stating: `CampaignRepository.Update` writes all five mutable
// columns in one statement — deliberately, so `code` can never be rewritten and
// `closed_at` can never be cleared — which means an omitted field has to arrive
// pre-filled. A merge that forgot one column would silently blank it, and a campaign
// that quietly lost its `ends_at` would keep granting forever.
func TestUpdateKeepsEveryFieldTheRequestDidNotSend(t *testing.T) {
	endsAt := baseInstant().AddDate(0, 1, 0)
	campaign := testCampaign("tet-2027", "premium-monthly", 100, 10, timePtr(baseInstant()), &endsAt)
	f := seededUpdateFixture(t, campaign)

	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil); err != nil {
		t.Fatalf("the edit must succeed: %v", err)
	}

	row := f.repo.row("tet-2027")
	if row.PlanSlug != "premium-monthly" {
		t.Errorf("planSlug must keep its stored value when omitted, got %q", row.PlanSlug)
	}
	if row.Reserve != 10 {
		t.Errorf("reserve must keep its stored value when omitted, got %d", row.Reserve)
	}
	if !row.StartsAt.Equal(baseInstant()) {
		t.Errorf("startsAt must keep its stored value when omitted, got %s", row.StartsAt)
	}
	if row.EndsAt == nil || !row.EndsAt.Equal(endsAt) {
		t.Errorf("endsAt must keep its stored value when omitted, got %v", row.EndsAt)
	}
	if row.ClosedAt != nil {
		t.Errorf("an edit must never close a campaign (FR-034), got closedAt=%v", row.ClosedAt)
	}
}

// ---- US3 scenario 4 / FR-055's no-op exception ----

// TestUpdateWithNoChangedFieldIsANoOp is US3 acceptance scenario 4 and FR-055's
// explicit exception, and the ORDERING inside it is the requirement: the no-op answer
// is taken BEFORE the version check and BEFORE the immutability check.
//
// The request below carries fields whose values ALREADY match the row, and arrives
// with the STALE version a retry would bring — because that is exactly what a
// resubmitted form is. It must succeed, move nothing, and record nothing: a history
// line reading "quota changed from 120 to 120" would be worse than no line at all.
func TestUpdateWithNoChangedFieldIsANoOp(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())

	// One real edit first, so the fixture holds a campaign at version 2 — the state a
	// screen is looking at when the operator hits save again without editing anything.
	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil); err != nil {
		t.Fatalf("precondition: the first edit must succeed: %v", err)
	}
	afterFirst := f.audit.all()
	if len(afterFirst) != 1 {
		t.Fatalf("precondition: the first edit must record one audit row, got %d", len(afterFirst))
	}

	// The value the screen is DISPLAYING (120 after the first edit), resubmitted with
	// the version the client still holds (1) — which is what a resubmit looks like.
	unchanged := 120
	stale := &sub_dto.CampaignUpdateReq{Version: 1, Quota: &unchanged}
	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", stale, nil)
	if err != nil {
		t.Fatalf("FR-055: a PATCH that changes nothing must succeed, got %v", err)
	}

	if summary.Version != 2 {
		t.Errorf("version = %d, want it left at 2 — a no-op changes nothing (FR-055)", summary.Version)
	}
	if summary.Quota != 120 {
		t.Errorf("the no-op must report the CURRENT values, got quota=%d, want 120", summary.Quota)
	}
	// No second write: the answer is taken before the conditional statement, so it is
	// not even attempted. Asserted on the CALL rather than on the row alone, because a
	// write that changed nothing would still leave the row looking correct.
	if len(f.repo.updateVersions) != 1 {
		t.Errorf("an edit that changes nothing must not be written, got %d Update calls",
			len(f.repo.updateVersions))
	}
	if records := f.audit.all(); len(records) != 1 {
		t.Errorf("FR-055/FR-060: a no-op must record NO second audit row, got %d", len(records))
	}
	if row := f.repo.row("tet-2027"); row == nil || row.Version != 2 {
		t.Errorf("version = %v, want it left at 2", row)
	}
}

// TestUpdateTreatsTheSameInstantInAnotherOffsetAsNoChange is the same rule with the
// subtlety that makes it worth a test of its own.
//
// A client sending `startsAt` in a different UTC offset for the SAME moment has not
// moved the campaign. Comparing the rendered strings would call that a change, bump
// the version and write an audit row for an edit that changed nothing — which is
// FR-055's no-op exception failing for a reason no operator could explain.
func TestUpdateTreatsTheSameInstantInAnotherOffsetAsNoChange(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())

	sameMomentElsewhere := baseInstant().In(time.FixedZone("UTC+0", 0)).Format(time.RFC3339)
	req := &sub_dto.CampaignUpdateReq{Version: 1, StartsAt: &sameMomentElsewhere}

	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err != nil {
		t.Fatalf("an equivalent instant must not be refused: %v", err)
	}
	if summary.Version != 1 {
		t.Errorf("version = %d, want it left at 1 — the same instant in another offset is not a change",
			summary.Version)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("an equivalent instant must record nothing, got %d rows", len(f.audit.all()))
	}
}

// ---- US3 scenario 2: FR-020/FR-021 refused at the application layer ----

// TestUpdateRefusesBudgetChangesOnceTheCampaignHasGranted is US3 acceptance scenario 2
// and FR-021's application layer.
//
// THE MESSAGE IS ASSERTED, and it is a requirement rather than polish. The operator
// cannot act on "immutable" — they can act on "close this code and open a new one",
// and that is the only thing available to them, because closing is one-way and no
// endpoint can reopen a campaign (FR-034). A 422 that names no way out sends them to
// the same form again.
func TestUpdateRefusesBudgetChangesOnceTheCampaignHasGranted(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*sub_dto.CampaignUpdateReq)
	}{
		{"quota", func(r *sub_dto.CampaignUpdateReq) { r.Quota = intPtr(120) }},
		{"reserve", func(r *sub_dto.CampaignUpdateReq) { r.Reserve = intPtr(25) }},
		{"planSlug", func(r *sub_dto.CampaignUpdateReq) { r.PlanSlug = strPtr("premium-yearly") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := seededUpdateFixture(t, openCampaignRow())
			f.repo.markGranted("tet-2027")
			req := validUpdateReq(1)
			tc.mut(req)

			_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
			if err == nil {
				t.Fatalf("changing %s after a grant must be refused", tc.name)
			}
			if status := httpStatusOf(t, err); status != 422 {
				t.Errorf("FR-021: changing %s after a grant must be 422, got %d (%v)",
					tc.name, status, err)
			}
			// The way out, in Vietnamese, because the reader is an operator and the
			// constitution requires client-facing text in that language.
			for _, fragment := range []string{"tet-2027", "mở"} {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("the 422 must tell the operator to open a new code (missing %q), got %q",
						fragment, err.Error())
				}
			}
			if len(f.repo.updateVersions) != 0 {
				t.Errorf("a refused edit must not attempt the UPDATE, got %v", f.repo.updateVersions)
			}
			if len(f.audit.all()) != 0 {
				t.Errorf("FR-061: a refused edit must record nothing, got %d", len(f.audit.all()))
			}
			row := f.repo.row("tet-2027")
			if row.Quota != 100 || row.Reserve != 10 {
				t.Errorf("the stored budget must be untouched, got %+v", row)
			}
		})
	}
}

// TestUpdateStillEditsTheWindowOfACampaignThatHasGranted is FR-023, and it is the
// negative control for the test above.
//
// `starts_at` and `ends_at` are editable whether or not grants exist, because each
// claim snapshotted its own `registered_at` and its own budget share. A check that
// asked "has this campaign granted?" before ANY change would refuse this edit — and
// moving `ends_at` is how an operator extends a promotion that is already running.
func TestUpdateStillEditsTheWindowOfACampaignThatHasGranted(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	f.repo.markGranted("tet-2027")

	later := baseInstant().AddDate(0, 2, 0)
	req := &sub_dto.CampaignUpdateReq{
		Version:  1,
		StartsAt: strPtr(later.Format(time.RFC3339)),
	}

	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil); err != nil {
		t.Fatalf("FR-023: the window stays editable after a grant, got %v", err)
	}
	row := f.repo.row("tet-2027")
	if !row.StartsAt.Equal(later) {
		t.Errorf("starts_at = %s, want the requested %s", row.StartsAt, later)
	}
	if len(f.audit.all()) != 1 {
		t.Errorf("FR-060: an edit that changes a row must record one audit command, got %d",
			len(f.audit.all()))
	}
}

// TestUpdateDoesNotReadTheLedgerForANonBudgetEdit is the cost half of FR-023.
//
// It matters because the stub answers `ExistsClaims` from memory while the real
// `EXISTS` query costs a round trip, and an implementation that consulted it on every
// edit would be slower for no gain. `touchesBudget()` is what keeps the read off the
// timestamp path; asserting the absence of the call is what keeps it off later.
func TestUpdateDoesNotReadTheLedgerForANonBudgetEdit(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())

	req := &sub_dto.CampaignUpdateReq{Version: 1, Quota: intPtr(101)}
	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil); err != nil {
		t.Fatalf("the edit must succeed: %v", err)
	}
	if len(f.repo.ledgerReads) != 1 {
		t.Fatalf("precondition: a budget edit must ask the ledger exactly once, got %d reads",
			len(f.repo.ledgerReads))
	}

	later := baseInstant().AddDate(0, 2, 0)
	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027",
		&sub_dto.CampaignUpdateReq{Version: 2, StartsAt: strPtr(later.Format(time.RFC3339))}, nil); err != nil {
		t.Fatalf("the window edit must succeed: %v", err)
	}
	if len(f.repo.ledgerReads) != 1 {
		t.Errorf("FR-023: a window-only edit must not read the ledger, got %d reads in total",
			len(f.repo.ledgerReads))
	}
}

// ---- FR-054: 404 before 412 ----

// TestUpdateRefusesAnUnknownCodeWith404BeforeItComparesTheVersion pins contract §4.3's
// ordering.
//
// The version below is one no row could ever hold, so a use case that compared it
// before looking the code up would still produce 412 here and fail the assertions.
func TestUpdateRefusesAnUnknownCodeWith404BeforeItComparesTheVersion(t *testing.T) {
	f := newRegistryFixture()

	_, err := f.uc.Update(context.Background(), uuid.New(), "khong-ton-tai", validUpdateReq(99), nil)
	if err == nil {
		t.Fatal("editing an unknown code must fail")
	}
	if status := httpStatusOf(t, err); status != 404 {
		t.Errorf("FR-054: an unknown code must be 404, got %d (%v)", status, err)
	}
	if len(f.repo.updateVersions) != 0 {
		t.Errorf("the conditional UPDATE must not be attempted for an unknown code, got %v",
			f.repo.updateVersions)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused edit must record nothing, got %d", len(f.audit.all()))
	}
}

// TestUpdateRefusesAStaleVersionWith412AndWritesNothing is the other side of the same
// boundary: the code EXISTS and somebody else wrote to it in between.
func TestUpdateRefusesAStaleVersionWith412AndWritesNothing(t *testing.T) {
	campaign := openCampaignRow()
	f := seededUpdateFixture(t, campaign)

	moved := *campaign
	moved.Quota = 999
	moved.Version = 2
	f.repo.seedRow(&moved)

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil)
	if err == nil {
		t.Fatal("a stale version must be refused")
	}
	if status := httpStatusOf(t, err); status != 412 {
		t.Errorf("FR-055: a stale version must be 412, got %d (%v)", status, err)
	}
	// The other administrator's change is STILL THERE — the half of FR-055 that
	// matters, since a refusal that overwrote anyway would have lost exactly the work
	// the loser was told to reload and redo.
	if row := f.repo.row("tet-2027"); row == nil || row.Quota != 999 || row.Version != 2 {
		t.Errorf("a refused edit must change nothing, got %+v", row)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused edit must record nothing, got %d", len(f.audit.all()))
	}
}

// TestUpdateReportsADatabaseFailureAsAnInfrastructureError is the failure that is easy
// to get wrong, because it arrives from the same statement that produces a 412.
//
// Reporting it as 412 would tell the operator "somebody else changed this, reload and
// retry" when in fact the database refused the statement and the row they would reload
// is unchanged — so they would retry against a broken database for ever.
func TestUpdateReportsADatabaseFailureAsAnInfrastructureError(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	f.repo.updateErr = context.DeadlineExceeded

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil)
	if err == nil {
		t.Fatal("a failing UPDATE must fail the operation")
	}
	if status := httpStatusOf(t, err); status == 412 {
		t.Errorf("a database fault must not be reported as 412 (got %v)", err)
	}
	if status := httpStatusOf(t, err); status != 500 {
		t.Errorf("expected 500 for an infrastructure failure, got %d (%v)", status, err)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a failed edit must record nothing, got %d", len(f.audit.all()))
	}
}

// ---- FR-054: 400 versus 422 ----

// TestUpdateDistinguishesBrokenShapeFromWrongMeaning is FR-054's table for the edit
// endpoint, and the whole reason it exists is that a client has to tell the two apart.
//
// The cases are all well-formed JSON with plausible field names: a 400 here would tell
// the operator their request is unreadable when it is perfectly readable and the NUMBERS
// in it are wrong.
func TestUpdateDistinguishesBrokenShapeFromWrongMeaning(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*sub_dto.CampaignUpdateReq)
		want   int
		// mentions is a fragment the message must contain, so an operator is told WHICH
		// field to fix rather than only that something is wrong.
		mentions string
	}{
		{
			name:     "a zone-less startsAt is a broken shape",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.StartsAt = strPtr("2026-10-02 00:00:00") },
			want:     400,
			mentions: fieldStartsAt,
		},
		{
			name:     "a date without a time is a broken shape",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.StartsAt = strPtr("2026-10-02") },
			want:     400,
			mentions: fieldStartsAt,
		},
		{
			name:     "an empty-string endsAt is a malformed timestamp",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.EndsAt = strPtr("") },
			want:     400,
			mentions: fieldEndsAt,
		},
		{
			name:     "a zero quota is wrong meaning",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.Quota = intPtr(0) },
			want:     422,
			mentions: fieldQuota,
		},
		{
			name:     "a negative quota is wrong meaning",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.Quota = intPtr(-5) },
			want:     422,
			mentions: fieldQuota,
		},
		{
			name:     "a negative reserve is wrong meaning",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.Reserve = intPtr(-1) },
			want:     422,
			mentions: fieldReserve,
		},
		{
			name:     "a blank planSlug is wrong meaning",
			mutate:   func(r *sub_dto.CampaignUpdateReq) { r.PlanSlug = strPtr("   ") },
			want:     422,
			mentions: fieldPlanSlug,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := seededUpdateFixture(t, openCampaignRow())
			req := validUpdateReq(1)
			tc.mutate(req)

			_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
			if err == nil {
				t.Fatalf("expected a refusal with %d, got success", tc.want)
			}
			if status := httpStatusOf(t, err); status != tc.want {
				t.Errorf("FR-054: expected %d, got %d (%v)", tc.want, status, err)
			}
			if !strings.Contains(err.Error(), tc.mentions) {
				t.Errorf("the message must name the offending field (%q), got %q", tc.mentions, err.Error())
			}
			// A rejected edit writes nothing at all, so a typo cannot leave a trace
			// saying somebody tried.
			if len(f.repo.updateVersions) != 0 {
				t.Errorf("a refused edit must not write a row, got %d statements", len(f.repo.updateVersions))
			}
			if len(f.audit.all()) != 0 {
				t.Errorf("a refused edit must not write an audit record, got %d", len(f.audit.all()))
			}
		})
	}
}

// TestUpdateIsRejectedBeforeAnyTransactionWhenTheShapeIsWrong is the ordering FR-054's
// table implies and the transactional guarantee depends on.
//
// A malformed request must not open a transaction: every typo would take a connection
// from the pool and a lock it does not need, and the audit trail would gain outbox
// rows for requests that never described a change.
func TestUpdateIsRejectedBeforeAnyTransactionWhenTheShapeIsWrong(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	req := validUpdateReq(1)
	req.StartsAt = strPtr("2026-10-02 00:00:00")

	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil); err == nil {
		t.Fatal("expected a refusal")
	}
	if f.uow.entries != 0 {
		t.Errorf("a rejected request must not open a transaction, got %d", f.uow.entries)
	}
}

// ---- FR-061: the rollback ----

// TestUpdateRollsBackAndLeavesNoAuditRecordWhenTheAuditFails is FR-061 for the edit,
// and the assertion is on BOTH sides of the transaction.
//
// The audit fake RECORDS the command and THEN fails, so "nothing is left" is the
// rollback's doing rather than the stub's refusal. Checking only that an error came
// back would also pass for a use case that wrote its audit row after the commit.
func TestUpdateRollsBackAndLeavesNoAuditRecordWhenTheAuditFails(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	f.audit.fail = true

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil)
	if err == nil {
		t.Fatal("a failed audit insert must fail the edit")
	}
	if status := httpStatusOf(t, err); status != 500 {
		t.Errorf("expected 500 for an audit failure, got %d (%v)", status, err)
	}
	if f.audit.issuedCount() != 1 {
		t.Fatalf("the fixture must have issued exactly one audit command, got %d - without it this "+
			"test cannot tell a rollback from a stub that never records", f.audit.issuedCount())
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a failed edit must leave NO audit record, got %d", len(f.audit.all()))
	}
	if f.uow.entries != 1 {
		t.Errorf("expected the operation to have entered one transaction, got %d", f.uow.entries)
	}
}

// ---- FR-062: the trigger is the other half ----

// TestUpdateCannotBeBypassedByTheApplicationBecauseItIsNotTheApplicationsJob records
// what T057 proves on a real database, so the gap is a stated one rather than a silent
// omission.
//
// FR-062 says the campaign table MUST NOT be writable by direct SQL in operation, and
// FR-021 names TWO layers: this application check, which answers the operator in
// Vietnamese, and `trg_campaigns_immutable_budget`, which is the only layer a
// hand-written UPDATE cannot walk around. Nothing in this file can prove the trigger
// exists — `campaign_guard_integration_test.go` (T057) is where the assertion lives.
func TestUpdateCannotBeBypassedByTheApplicationBecauseItIsNotTheApplicationsJob(t *testing.T) {
	// Not a tautology: the test fails if somebody removes the application layer's
	// EXISTS read, because the stub counts the ledger reads and a budget edit now
	// makes exactly one.
	f := seededUpdateFixture(t, openCampaignRow())
	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil); err != nil {
		t.Fatalf("precondition: the edit must succeed: %v", err)
	}
	if len(f.repo.ledgerReads) != 1 {
		t.Errorf("FR-021: a budget edit must consult the ledger inside its transaction, got %d reads",
			len(f.repo.ledgerReads))
	}
}

// TestUpdateHandlesAnOpenEndedCampaignWithoutInventingAnEnd is the negative control
// for the merge rule on a nullable column.
//
// `ends_at` nil means "no end" and must stay nil through an edit. A merge that
// dereferenced the stored pointer to compare it would panic here, and one that wrote a
// zero time would give a campaign an expiry the moment its budget was raised.
func TestUpdateHandlesAnOpenEndedCampaignWithoutInventingAnEnd(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())

	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil); err != nil {
		t.Fatalf("an open-ended campaign must be editable: %v", err)
	}
	row := f.repo.row("tet-2027")
	if row.EndsAt != nil {
		t.Errorf("ends_at must stay NULL, got %v", row.EndsAt)
	}
}

// TestUpdateReadsARegistryFailureAsAnInfrastructureError keeps 404 and 500 apart on the
// UPDATE path too.
//
// The stub cannot produce a read fault for `GetByCode` — it answers from a map — so the
// fault is injected on the ledger read, which the update path DOES consult. An outage
// reported as "this campaign does not exist" would send an operator hunting for a code
// that has been running for months.
func TestUpdateReadsARegistryFailureAsAnInfrastructureError(t *testing.T) {
	f := seededUpdateFixture(t, openCampaignRow())
	f.repo.claimsErr = errors.New("connection reset")

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", validUpdateReq(1), nil)
	if err == nil {
		t.Fatal("a failing ledger read must fail the operation")
	}
	if status := httpStatusOf(t, err); status != 500 {
		t.Errorf("a read failure must be 500, got %d (%v)", status, err)
	}
	if strings.Contains(err.Error(), "connection reset") {
		t.Errorf("the underlying cause must not reach the client, got %q", err.Error())
	}
}

package campaign

import (
	"context"
	"strings"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
)

// 026 T045 (FR-030..FR-034, FR-055, FR-060, FR-061, FR-063) — `CampaignRegistryUseCase.Close`.
//
// WHAT A UNIT TEST CAN PROVE HERE AND WHAT IT CANNOT. Every assertion below is a
// DECISION: which error the operator is told, in which ORDER, whether the audit
// command was issued, and how many times. It cannot prove that `closed_at` was
// written by PostgreSQL's clock, that `FOR SHARE` makes a concurrent close wait, or
// that a real `UPDATE ... WHERE version = ?` changes zero rows on a stale version —
// those are the database's, and they are covered by `campaign_repo_integration_test.go`
// (T022) and by `campaign_db_close_integration_test.go` (T049).
//
// THE ORDER TESTS ARE THE POINT, not a detail of them. `Close` answers 404, 200-idempotent,
// 412 and 200-changed, and three of the four are reachable from the same request
// shape; a use case that consulted them in a different order would still pass a test
// of each code taken alone. So the tests that matter here are the ones that pin which
// ANSWER a request gets, not merely that every answer exists.

// ---- fixtures ----

// openCampaignRow is a campaign that exists and has not been closed: the state every
// close starts from.
//
// The version is 1 because that is what the column default installs and what `Create`
// returns, so `validCloseReq(1)` is the request a client would send after opening one.
func openCampaignRow() *entities.Campaign {
	return testCampaign("tet-2027", "premium-monthly", 100, 10, timePtr(baseInstant()), nil)
}

// validCloseReq is a request that passes every rule, so a test can break exactly one
// thing and know the failure it is reading.
func validCloseReq(version int) *sub_dto.CampaignCloseReq {
	reason := "Hết ngân sách khuyến mãi tháng 11"
	return &sub_dto.CampaignCloseReq{Version: version, Reason: &reason}
}

// seededCloseFixture returns a fixture whose registry already holds `campaign`.
func seededCloseFixture(t *testing.T, campaign *entities.Campaign) *registryFixture {
	t.Helper()
	f := newRegistryFixture()
	f.repo.seedRow(campaign)
	return f
}

// ---- the happy path ----

// TestCloseWritesClosedAtBumpsTheVersionAndRecordsExactlyOneAudit is FR-030, FR-032,
// FR-055 and FR-060 in one pass, because a closure that did all three wrongly would
// still look right to an operator.
//
// The AUDIT PAYLOAD is asserted field by field rather than for presence, because the
// per-campaign audit screen (FR-063, T067) filters on `payload.campaignCode` and
// renders `payload.reason`. A record that exists but cannot be found, or that cannot
// say why anybody closed the campaign, answers neither question the screen exists for.
func TestCloseWritesClosedAtBumpsTheVersionAndRecordsExactlyOneAudit(t *testing.T) {
	f := seededCloseFixture(t, openCampaignRow())
	actor := uuid.New()
	reqIP := "203.0.113.7"

	summary, err := f.uc.Close(context.Background(), actor, "tet-2027", validCloseReq(1), &reqIP)
	if err != nil {
		t.Fatalf("closing an open campaign must succeed, got %v", err)
	}

	// `closed_at` IS SET, and the response says `closed` — FR-032's whole content.
	// The instant itself is written by the database, so the assertion is that the
	// operation PRODUCED one rather than that it produced a particular value.
	row := f.repo.row("tet-2027")
	if row == nil || row.ClosedAt == nil {
		t.Fatalf("FR-031: the close must write closed_at, got row %+v", row)
	}
	if summary.Status != sub_dto.CampaignStatusClosed {
		t.Errorf("status = %q, want %q for a campaign an operator just closed",
			summary.Status, sub_dto.CampaignStatusClosed)
	}
	if summary.ClosedAt == nil {
		t.Error("the response must carry closedAt, or the operator cannot see when it was closed")
	}

	// FR-055: the version moves exactly once, so the next write from this client is
	// checked against the state this one produced.
	if row.Version != 2 {
		t.Errorf("version = %d, want it incremented to 2 by the statement", row.Version)
	}

	// The version the CLIENT sent is the version COMPARED. A use case that compared a
	// version of its own would pass every other assertion here and still fail FR-055
	// the moment two administrators acted on the same campaign.
	if len(f.repo.closeVersions) != 1 || f.repo.closeVersions[0] != 1 {
		t.Errorf("expected exactly one Close compared against version 1, got %v", f.repo.closeVersions)
	}

	// Exactly one transaction (constitution §V). A second one would open a second
	// connection, and the row written in the first would be invisible to the audit
	// insert in the second.
	if f.uow.entries != 1 {
		t.Errorf("Close entered %d transactions; constitution §V allows exactly one per operation", f.uow.entries)
	}

	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("FR-060: expected exactly one audit command, got %d", len(records))
	}
	record := records[0]
	if record.Action != string(adminaudit.CampaignClose) {
		t.Errorf("action = %q, want %q", record.Action, adminaudit.CampaignClose)
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
	// The reason is the ONLY record of why a one-way operation was taken, six weeks
	// later, on the one screen that can show it.
	want := "Hết ngân sách khuyến mãi tháng 11"
	if got := record.Payload["reason"]; got != want {
		t.Errorf("payload.reason = %v, want %q", got, want)
	}
	// FR-060 asks for the old → new values, and a closure's change is the closure
	// instant plus the version that moved (contract §6.3).
	before, okBefore := record.Payload["before"].(map[string]any)
	after, okAfter := record.Payload["after"].(map[string]any)
	if !okBefore || !okAfter {
		t.Fatalf("the close audit payload must carry before and after, got %#v", record.Payload)
	}
	if before["closedAt"] != "" || before["version"] != 1 {
		t.Errorf("payload.before must describe an open campaign at version 1, got %#v", before)
	}
	if after["closedAt"] == "" || after["version"] != 2 {
		t.Errorf("payload.after must describe a closed campaign at version 2, got %#v", after)
	}
	if record.actor != actor {
		t.Errorf("the audit actor = %s, want the administrator %s", record.actor, actor)
	}
}

// WHAT THIS FILE DELIBERATELY DOES NOT ASSERT, AND WHY. FR-033 — "closing must not
// change the plan of an account that was already granted" — is absent from this file
// rather than covered by a fake: the registry writer holds no subscription repository,
// so nothing it can do reaches a granted account, and an assertion made through a fake
// about a table this component cannot write to would be theatre. It is proved against
// the database instead, in `campaign_db_close_granted_integration_test.go` (T047).
//
// ---- FR-034: idempotence ----

// TestCloseIsIdempotentAndRecordsNoSecondAuditRow is US2 acceptance scenario 4, and
// it is the one case where the version is deliberately NOT consulted.
//
// The request carries a version the row no longer has, which is exactly what a retry
// looks like: the client read version 1, the first attempt succeeded and moved the row
// to 2, and the retry arrives with 1 again. Answering 412 here would tell the operator
// their own successful closure failed, which is how a one-way operation ends up being
// attempted twice out of sheer distrust — and two audit rows would then describe one
// change (FR-060 permits one).
func TestCloseIsIdempotentAndRecordsNoSecondAuditRow(t *testing.T) {
	f := seededCloseFixture(t, openCampaignRow())
	actor := uuid.New()

	if _, err := f.uc.Close(context.Background(), actor, "tet-2027", validCloseReq(1), nil); err != nil {
		t.Fatalf("the first close must succeed: %v", err)
	}
	afterFirst := f.audit.all()
	if len(afterFirst) != 1 {
		t.Fatalf("precondition: the first close must record one audit row, got %d", len(afterFirst))
	}

	// THE SAME STALE VERSION, which is the whole point: idempotence is checked BEFORE
	// the version comparison (FR-055, contract §5.3).
	summary, err := f.uc.Close(context.Background(), actor, "tet-2027", validCloseReq(1), nil)
	if err != nil {
		t.Fatalf("FR-034: closing an already-closed campaign must succeed, got %v", err)
	}
	if summary.Status != sub_dto.CampaignStatusClosed {
		t.Errorf("status = %q, want %q on the idempotent answer too", summary.Status, sub_dto.CampaignStatusClosed)
	}

	if records := f.audit.all(); len(records) != 1 {
		t.Errorf("FR-060/FR-034: a repeated close must record NO second audit row, got %d", len(records))
	}
	// And no second write: the idempotent answer is taken before the UPDATE, so the
	// conditional statement is not even attempted — otherwise `version` would move a
	// second time for a request that changed nothing.
	if len(f.repo.closeVersions) != 1 {
		t.Errorf("an already-closed campaign must not be written again, got %d Close calls", len(f.repo.closeVersions))
	}
	if row := f.repo.row("tet-2027"); row == nil || row.Version != 2 {
		t.Errorf("version = %v, want it left at 2 — an idempotent close changes nothing", row)
	}
}

// ---- FR-054: 404 before 412 ----

// TestCloseRefusesAnUnknownCodeWith404BeforeItComparesTheVersion pins contract §4.3's
// ordering for close.
//
// `UPDATE ... WHERE code = ? AND version = ?` returns ZERO ROWS for both "no such
// code" and "wrong version", so a use case that looked only at the row count would
// answer 412 to an operator who mistyped a code — telling them to reload a campaign
// that does not exist. The assertion that no `Close` was even ATTEMPTED is what makes
// this about the order rather than about the code.
func TestCloseRefusesAnUnknownCodeWith404BeforeItComparesTheVersion(t *testing.T) {
	f := newRegistryFixture()

	// A version nothing could ever carry, so a use case that compared it before
	// looking the code up would still produce 412 here and fail the assertion below.
	_, err := f.uc.Close(context.Background(), uuid.New(), "khong-ton-tai", validCloseReq(99), nil)
	if err == nil {
		t.Fatal("closing an unknown code must fail")
	}
	if status := httpStatusOf(t, err); status != 404 {
		t.Errorf("FR-054: an unknown code must be 404, got %d (%v)", status, err)
	}
	if len(f.repo.closeVersions) != 0 {
		t.Errorf("the conditional UPDATE must not be attempted for an unknown code, got versions %v",
			f.repo.closeVersions)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused close must record nothing, got %d", len(f.audit.all()))
	}
}

// TestCloseRefusesAStaleVersionWith412AndWritesNothing is the other side of the same
// boundary: the code EXISTS, and somebody else wrote to it in between.
func TestCloseRefusesAStaleVersionWith412AndWritesNothing(t *testing.T) {
	campaign := openCampaignRow()
	f := seededCloseFixture(t, campaign)

	// Another administrator's change: the row moved on to version 2.
	moved := *campaign
	moved.Version = 2
	f.repo.seedRow(&moved)

	_, err := f.uc.Close(context.Background(), uuid.New(), "tet-2027", validCloseReq(1), nil)
	if err == nil {
		t.Fatal("a stale version must be refused")
	}
	if status := httpStatusOf(t, err); status != 412 {
		t.Errorf("FR-055: a stale version must be 412, got %d (%v)", status, err)
	}
	// The campaign is STILL OPEN, which is the half of FR-055 that matters: a refusal
	// that closed it anyway would have destroyed the very state the loser was told to
	// reload.
	if row := f.repo.row("tet-2027"); row == nil || row.ClosedAt != nil {
		t.Errorf("a refused close must leave the campaign open, got %+v", row)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused close must record nothing, got %d", len(f.audit.all()))
	}
}

// TestCloseReportsADatabaseFailureAsAnInfrastructureError is the failure that is easy
// to get wrong, because it arrives from the same statement that produces a 412.
//
// Reporting it as 412 would tell an operator "somebody else changed this, reload and
// try again" when in fact the database refused the statement, and the row they would
// reload is unchanged — so they would retry against a broken database forever.
func TestCloseReportsADatabaseFailureAsAnInfrastructureError(t *testing.T) {
	f := seededCloseFixture(t, openCampaignRow())
	f.repo.closeErr = context.DeadlineExceeded

	_, err := f.uc.Close(context.Background(), uuid.New(), "tet-2027", validCloseReq(1), nil)
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
		t.Errorf("FR-061: a failed close must record nothing, got %d", len(f.audit.all()))
	}
}

// ---- FR-054: 422, and the ordering of validation ----

// TestCloseRefusesABlankReasonWith422 is FR-054's `reason` row, and it is the reason
// the DTO's `reason` is a POINTER: `"reason": ""` and a missing `reason` are two
// different mistakes and FR-054 gives them two different codes (400 for the absent
// field, which the binder answers, and 422 for the blank one).
func TestCloseRefusesABlankReasonWith422(t *testing.T) {
	cases := []struct {
		name   string
		reason string
	}{
		{"an empty reason", ""},
		{"a whitespace-only reason", "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := seededCloseFixture(t, openCampaignRow())
			req := validCloseReq(1)
			req.Reason = strPtr(tc.reason)

			_, err := f.uc.Close(context.Background(), uuid.New(), "tet-2027", req, nil)
			if err == nil {
				t.Fatalf("a blank reason must be refused")
			}
			if status := httpStatusOf(t, err); status != 422 {
				t.Errorf("FR-054: a blank reason must be 422, got %d (%v)", status, err)
			}
			// And the message NAMES THE FIELD, because the operator is looking at a
			// form and a refusal that does not say which box to fill sends them round
			// the same form again.
			if !strings.Contains(err.Error(), fieldReason) {
				t.Errorf("the message must name the offending field (%q), got %q", fieldReason, err.Error())
			}
			if len(f.repo.closeVersions) != 0 || len(f.audit.all()) != 0 {
				t.Errorf("a refused close must write nothing, got %d Close calls and %d audit rows",
					len(f.repo.closeVersions), len(f.audit.all()))
			}
		})
	}
}

// TestCloseIsRejectedBeforeAnyTransactionWhenTheReasonIsBlank is the ordering FR-054's
// table implies and the transactional guarantee depends on.
//
// A malformed request must not open a transaction: every typo would take a connection
// from the pool and a row lock it does not need, and the audit trail would gain outbox
// rows for requests that never described a change.
func TestCloseIsRejectedBeforeAnyTransactionWhenTheReasonIsBlank(t *testing.T) {
	f := seededCloseFixture(t, openCampaignRow())
	req := validCloseReq(1)
	req.Reason = strPtr("  ")

	if _, err := f.uc.Close(context.Background(), uuid.New(), "tet-2027", req, nil); err == nil {
		t.Fatal("expected a refusal")
	}
	if f.uow.entries != 0 {
		t.Errorf("a rejected request must not open a transaction, got %d", f.uow.entries)
	}
}

// ---- FR-061: the rollback ----

// TestCloseRollsBackAndLeavesNoAuditRecordWhenTheAuditFails is FR-061 for close, and
// the assertion is on BOTH sides of the transaction.
//
// `fakeAuditContract` RECORDS THE COMMAND AND THEN FAILS (see its header), so the
// counter is non-zero and "nothing is left" is the rollback's doing rather than the
// fake's refusal. Without the counter this test would pass against a fake that simply
// refused to record.
func TestCloseRollsBackAndLeavesNoAuditRecordWhenTheAuditFails(t *testing.T) {
	f := seededCloseFixture(t, openCampaignRow())
	f.audit.fail = true

	_, err := f.uc.Close(context.Background(), uuid.New(), "tet-2027", validCloseReq(1), nil)
	if err == nil {
		t.Fatal("a failed audit insert must fail the close")
	}
	if status := httpStatusOf(t, err); status != 500 {
		t.Errorf("expected 500 for an audit failure, got %d (%v)", status, err)
	}
	if f.audit.issuedCount() != 1 {
		t.Fatalf("the fixture must have issued exactly one audit command, got %d - without it this "+
			"test cannot tell a rollback from a stub that never records", f.audit.issuedCount())
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a failed close must leave NO audit record, got %d", len(f.audit.all()))
	}
	if f.uow.entries != 1 {
		t.Errorf("expected the operation to have entered one transaction, got %d", f.uow.entries)
	}
}

// TestCloseACampaignThatHasNotOpenedYet is the spec.md edge case "đóng một chiến dịch
// chưa tới `starts_at`", and it is a case a clock check would have refused.
//
// An operator scheduling a promotion for next month and deciding today to cancel it is
// the most ordinary reason this endpoint exists. The window is not consulted at all
// here, so a campaign whose `starts_at` is in the future closes exactly like a running
// one — and it reports `closed` rather than `not_started`, which is the pair the
// operator would otherwise have to guess between.
func TestCloseACampaignThatHasNotOpenedYet(t *testing.T) {
	startsAt := baseInstant().AddDate(0, 1, 0)
	campaign := testCampaign("tet-2027", "premium-monthly", 100, 10, &startsAt, nil)
	f := seededCloseFixture(t, campaign)

	summary, err := f.uc.Close(context.Background(), uuid.New(), "tet-2027", validCloseReq(1), nil)
	if err != nil {
		t.Fatalf("a campaign that has not opened must still be closable: %v", err)
	}
	if summary.Status != sub_dto.CampaignStatusClosed {
		t.Errorf("status = %q, want %q — the closed state outranks `not_started` (FR-032)",
			summary.Status, sub_dto.CampaignStatusClosed)
	}
	if len(f.audit.all()) != 1 {
		t.Errorf("FR-060: a closing that changes a row must record one audit command, got %d",
			len(f.audit.all()))
	}
}

package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
)

// 026 T062 (FR-013, FR-023, FR-055, FR-060) â€” the TIME half of the edit path.
//
// WHAT A UNIT TEST CAN PROVE HERE AND WHAT IT CANNOT. Every assertion is a DECISION:
// which answer an operator is given, whether the stored window moved, and what the audit
// row says. It cannot prove that `ends_at > starts_at` is also enforced by
// `chk_campaigns_window` on a real write, nor that a moved `ends_at` changes what the
// grant path does â€” those are `campaign_guard_integration_test.go` (T057) and
// `campaign_db_time_integration_test.go` (T063).
//
// THE THREE STATES OF `endsAt` ARE THE SUBJECT, because they are the part of this
// feature that is easy to get wrong in a way no other field can be. Every other optional
// field has two states and `*T` carries both. `endsAt` has three (absent / `null` / a
// timestamp, contract Â§4.1), and `encoding/json` cannot tell the first two apart on a
// pointer â€” so the DTO records the key itself. If that recording breaks, "clear the end
// date" silently degrades into "leave it alone" and the campaign stays expired for ever.

// ---- fixtures ----

// windowedCampaignRow is a campaign whose window is OPEN on both ends: it started in
// the past and ends next month.
//
// Both ends are real instants rather than nil because a window with only one end cannot
// answer the question US4 asks â€” "does this edit leave the window legal?" â€” in an
// interesting way.
func windowedCampaignRow() *entities.Campaign {
	endsAt := baseInstant().AddDate(0, 1, 0)
	return testCampaign("tet-2027", "premium-monthly", 100, 10, timePtr(baseInstant()), &endsAt)
}

// seededWindowFixture returns a fixture whose registry already holds `campaign`.
func seededWindowFixture(t *testing.T, campaign *entities.Campaign) *registryFixture {
	t.Helper()
	f := newRegistryFixture()
	f.repo.seedRow(campaign)
	return f
}

// updateWindowReq is an edit that touches ONLY the window, so a failure points at the
// window and nothing else.
func updateWindowReq(version int, endsAt *string) *sub_dto.CampaignUpdateReq {
	return &sub_dto.CampaignUpdateReq{Version: version, EndsAt: endsAt}
}

// ---- US4 scenario 1 and 2: moving the window ----

// TestUpdateMovesTheWindowEndAndRecordsIt is the positive half of US4: an operator who
// pulls `ends_at` forward (or pushes it back) gets the row the request described.
//
// IT ALSO ASSERTS THE AUDIT SIDE, because the reason an operator can trust the screen
// is that the change is in the trail: `before` carries the instant the campaign was going
// to stop at and `after` the new one, and FR-023's promise â€” "moving the window does not
// break the grants already written" â€” is only believable if the move itself is recorded.
func TestUpdateMovesTheWindowEndAndRecordsIt(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	later := baseInstant().AddDate(0, 3, 0)
	req := updateWindowReq(1, strPtr(later.Format(time.RFC3339)))

	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err != nil {
		t.Fatalf("US4.1/2: moving ends_at must succeed, got %v", err)
	}

	row := f.repo.row("tet-2027")
	if row.EndsAt == nil || !row.EndsAt.Equal(later) {
		t.Errorf("ends_at = %v, want the requested %s", row.EndsAt, later)
	}
	if row.Version != 2 {
		t.Errorf("version = %d, want it incremented to 2 by the statement", row.Version)
	}
	if summary.EndsAt == nil || !summary.EndsAt.Equal(later) {
		t.Errorf("the response must describe the new window, got endsAt=%v", summary.EndsAt)
	}

	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("FR-060: an edit that moves the window records exactly one audit row, got %d",
			len(records))
	}
	if records[0].Action != string(adminaudit.CampaignUpdate) {
		t.Errorf("action = %q, want %q", records[0].Action, adminaudit.CampaignUpdate)
	}
	before, okBefore := records[0].Payload["before"].(map[string]any)
	after, okAfter := records[0].Payload["after"].(map[string]any)
	if !okBefore || !okAfter {
		t.Fatalf("the update audit payload must carry before and after, got %#v", records[0].Payload)
	}
	if before[fieldEndsAt] != endsAtForAudit(timePtr(baseInstant().AddDate(0, 1, 0))) {
		t.Errorf("before.endsAt = %v, want the STORED window end", before[fieldEndsAt])
	}
	if after[fieldEndsAt] != later.UTC().Format(time.RFC3339) {
		t.Errorf("after.endsAt = %v, want the requested %s", after[fieldEndsAt], later)
	}
}

// TestUpdateTreatsTheSameWindowInstantInAnotherOffsetAsNoChange keeps FR-055's no-op
// exception honest for the new third state.
//
// A client sending the same moment in a different offset has not moved the campaign, and
// calling that a change would bump the version and write an audit row for an edit that
// changed nothing â€” the defect US3 already guarded against for `startsAt`, now reachable
// through the tri-state branch as well.
func TestUpdateTreatsTheSameWindowInstantInAnotherOffsetAsNoChange(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	sameMomentElsewhere := baseInstant().AddDate(0, 1, 0).In(time.FixedZone("UTC+0", 0)).Format(time.RFC3339)
	req := updateWindowReq(1, strPtr(sameMomentElsewhere))

	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err != nil {
		t.Fatalf("an equivalent instant must not be refused, got %v", err)
	}
	if summary.Version != 1 {
		t.Errorf("version = %d, want it left at 1 â€” the same instant in another offset is not a change",
			summary.Version)
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("an equivalent instant must record nothing, got %d rows", len(f.audit.all()))
	}
}

// ---- US4: `endsAt: null` clears the window (T061) ----

// TestUpdateClearsTheWindowEndOnAnExplicitNull is the state `*string` could not
// express, and the reason T061 exists.
//
// `null` is NOT "leave it alone": it is the request that makes a finished promotion
// open-ended again. Before T061 the two arrived as the same nil pointer, so an operator
// who asked to reopen a campaign was silently ignored â€” and the failure is invisible,
// because the response is a perfectly plausible summary of a campaign that is still
// expired.
func TestUpdateClearsTheWindowEndOnAnExplicitNull(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	// Decoded the way the endpoint decodes it, so the assertion is about the contract
	// and not about a hand-built request struct.
	req := decodeUpdateReq(t, `{"version":1,"endsAt":null}`)

	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err != nil {
		t.Fatalf("US4/T061: an explicit null must clear the window end, got %v", err)
	}

	row := f.repo.row("tet-2027")
	if row.EndsAt != nil {
		t.Errorf("ends_at must become NULL, got %v", row.EndsAt)
	}
	if summary.EndsAt != nil {
		t.Errorf("the response must report an open-ended campaign, got endsAt=%v", summary.EndsAt)
	}
	// FR-055: a real change, so the version moves and ONE audit row is written â€” with
	// `after.endsAt` empty rather than absent, because "it no longer stops" and "we do
	// not know when it stops" are different facts.
	if row.Version != 2 {
		t.Errorf("version = %d, want it incremented by the clear", row.Version)
	}
	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("FR-060: clearing the window records exactly one audit row, got %d", len(records))
	}
	after, ok := records[0].Payload["after"].(map[string]any)
	if !ok {
		t.Fatalf("the update audit payload must carry after, got %#v", records[0].Payload)
	}
	if value, present := after[fieldEndsAt]; !present || value != "" {
		t.Errorf("after.endsAt must be present and empty for a cleared window, got %#v", after)
	}
	before, ok := records[0].Payload["before"].(map[string]any)
	if !ok || before[fieldEndsAt] == "" {
		t.Errorf("before.endsAt must carry the instant that was cleared, got %#v", records[0].Payload)
	}
}

// TestUpdateKeepsTheWindowEndWhenTheKeyIsAbsent is the OTHER third state, and it is the
// half that protects a running promotion.
//
// An absent `endsAt` means "leave it alone" â€” the whole content of a `PATCH`. If it were
// read as "clear", every client that only edits the budget would silently give every
// campaign an open-ended window, and no test that checked the budget would notice.
func TestUpdateKeepsTheWindowEndWhenTheKeyIsAbsent(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	req := decodeUpdateReq(t, `{"version":1,"quota":120}`)

	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil); err != nil {
		t.Fatalf("a budget-only edit must succeed, got %v", err)
	}

	row := f.repo.row("tet-2027")
	if row.EndsAt == nil || !row.EndsAt.Equal(baseInstant().AddDate(0, 1, 0)) {
		t.Errorf("an absent endsAt must keep the stored window end, got %v", row.EndsAt)
	}
	if row.Quota != 120 {
		t.Errorf("the budget edit itself must have landed, got quota=%d", row.Quota)
	}
}

// TestUpdateClearingAnAlreadyOpenEndedCampaignIsANoOp is FR-055's exception applied to
// the new state.
//
// A campaign with no `ends_at` cannot have its `ends_at` cleared, so a form that always
// resubmits the whole row would otherwise bump the version and write an audit line for an
// edit that moved nothing â€” which is exactly what US3 scenario 4 forbids.
func TestUpdateClearingAnAlreadyOpenEndedCampaignIsANoOp(t *testing.T) {
	f := seededWindowFixture(t, openCampaignRow())

	req := decodeUpdateReq(t, `{"version":1,"endsAt":null}`)

	summary, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err != nil {
		t.Fatalf("clearing a window that has no end must succeed, got %v", err)
	}
	if summary.Version != 1 {
		t.Errorf("version = %d, want it left at 1 â€” clearing an absent end changes nothing (FR-055)",
			summary.Version)
	}
	if len(f.repo.updateVersions) != 0 {
		t.Errorf("a no-op must not be written, got %d UPDATE statements", len(f.repo.updateVersions))
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-055/FR-060: a no-op must record nothing, got %d rows", len(f.audit.all()))
	}
}

// TestUpdateDistinguishesAnAbsentKeyFromAnExplicitNullAtTheBoundary is the regression
// guard for the recording itself, stated as ONE test because the two answers are the
// same request with one character of difference.
//
// Both edits below are run against the SAME row. Before T061 they produced identical
// requests; now one keeps the window and one clears it. A decoder that loses the
// distinction makes these two assertions collapse onto one answer, which is why they
// are asserted together rather than as two independent tests.
func TestUpdateDistinguishesAnAbsentKeyFromAnExplicitNullAtTheBoundary(t *testing.T) {
	keep := seededWindowFixture(t, windowedCampaignRow())
	clear := seededWindowFixture(t, windowedCampaignRow())

	if _, err := keep.uc.Update(context.Background(), uuid.New(), "tet-2027",
		decodeUpdateReq(t, `{"version":1,"quota":120}`), nil); err != nil {
		t.Fatalf("precondition: the budget edit must succeed, got %v", err)
	}
	if _, err := clear.uc.Update(context.Background(), uuid.New(), "tet-2027",
		decodeUpdateReq(t, `{"version":1,"quota":120,"endsAt":null}`), nil); err != nil {
		t.Fatalf("precondition: the budget edit with a null window end must succeed, got %v", err)
	}

	keptEnds := keep.repo.row("tet-2027").EndsAt
	clearedEnds := clear.repo.row("tet-2027").EndsAt
	if keptEnds == nil {
		t.Error(`contract Â§4.1: an ABSENT endsAt must keep the stored window end, got NULL`)
	}
	if clearedEnds != nil {
		t.Errorf(`contract Â§4.1: an explicit null must CLEAR the window end, got %v`, clearedEnds)
	}
	// Both edits carry the same budget change, so neither row may differ anywhere else â€”
	// which is what makes the window the ONLY difference between the two requests.
	if keep.repo.row("tet-2027").Quota != clear.repo.row("tet-2027").Quota {
		t.Error("precondition broken: the two requests must differ only in endsAt")
	}
}

// ---- FR-013: the MERGED window (T061, decision 2) ----

// TestUpdateRefusesAStartsAtAfterTheStoredWindowEnd is the rule US4 added and the one
// the validator alone cannot decide.
//
// The request says nothing about `endsAt`, so on its own it is perfectly legal â€” and on
// the row it would leave behind it is not: moving `startsAt` past a stored `endsAt`
// describes a window that never existed, and `chk_campaigns_window` would refuse the
// write with a raw `check_violation` an operator cannot read. The application layer has
// to catch it first and answer 422 (contract Â§4.4, FR-054).
func TestUpdateRefusesAStartsAtAfterTheStoredWindowEnd(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	afterStoredEnd := baseInstant().AddDate(0, 2, 0)
	req := decodeUpdateReq(t, `{"version":1,"startsAt":"`+
		afterStoredEnd.Format(time.RFC3339)+`"}`)

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err == nil {
		t.Fatal("FR-013: a startsAt after the STORED endsAt must be refused on the merged window")
	}
	if status := httpStatusOf(t, err); status != 422 {
		t.Errorf("FR-054: a wrong-meaning window must be 422, got %d (%v)", status, err)
	}
	// The operator is told WHICH field, because they are editing a form.
	if !strings.Contains(err.Error(), fieldEndsAt) {
		t.Errorf("the 422 must name endsAt, the field they must change, got %q", err.Error())
	}
	if len(f.repo.updateVersions) != 0 {
		t.Errorf("a refused edit must not be written, got %d UPDATE statements", len(f.repo.updateVersions))
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused edit must record nothing, got %d rows", len(f.audit.all()))
	}
	// The stored window is untouched, which is the half that matters: the operator has
	// to be able to retry with a different date.
	row := f.repo.row("tet-2027")
	if !row.StartsAt.Equal(baseInstant()) || row.EndsAt == nil {
		t.Errorf("a refused edit must change nothing, got startsAt=%s endsAt=%v", row.StartsAt, row.EndsAt)
	}
}

// TestUpdateRefusesAnEndsAtBeforeTheStoredStart is the mirror image, and it is the case
// an operator hits most often: they pull the end date back to "tonight" on a campaign
// that opens next week.
func TestUpdateRefusesAnEndsAtBeforeTheStoredStart(t *testing.T) {
	startsAt := baseInstant().AddDate(0, 2, 0)
	endsAt := baseInstant().AddDate(0, 3, 0)
	campaign := testCampaign("tet-2027", "premium-monthly", 100, 10, &startsAt, &endsAt)
	f := seededWindowFixture(t, campaign)

	beforeStoredStart := baseInstant().Add(24 * time.Hour)
	req := updateWindowReq(1, strPtr(beforeStoredStart.Format(time.RFC3339)))

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err == nil {
		t.Fatal("FR-013: an endsAt before the STORED startsAt must be refused")
	}
	if status := httpStatusOf(t, err); status != 422 {
		t.Errorf("FR-054: expected 422, got %d (%v)", status, err)
	}
	if len(f.repo.updateVersions) != 0 {
		t.Errorf("a refused edit must not be written, got %d UPDATE statements", len(f.repo.updateVersions))
	}
}

// TestUpdateRefusesAnEndsAtEqualToTheStart is the same rule with EQUALITY, and it is
// its own case because the CHECK says `>` and not `>=`.
//
// A campaign whose window closes the instant it opens grants nobody: that is the same
// meaningless state a quota of zero describes, and it is why the boundary instant is
// refused rather than accepted.
func TestUpdateRefusesAnEndsAtEqualToTheStart(t *testing.T) {
	f := seededWindowFixture(t, openCampaignRow())

	sameInstant := baseInstant().Format(time.RFC3339)
	req := &sub_dto.CampaignUpdateReq{Version: 1, StartsAt: strPtr(sameInstant), EndsAt: strPtr(sameInstant)}

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil)
	if err == nil {
		t.Fatal("FR-013: an endsAt equal to the startsAt must be refused â€” the CHECK says `>`")
	}
	if status := httpStatusOf(t, err); status != 422 {
		t.Errorf("FR-054: expected 422, got %d (%v)", status, err)
	}
}

// TestUpdateClearingTheWindowIsNeverRefusedForClearing is the negative control, and it
// keeps the check from over-reaching.
//
// An open-ended campaign is legal (FR-009), so clearing `ends_at` must never be refused
// on its own account â€” only for landing before the start. A check that compared a nil
// end against the start would refuse every operator trying to reopen a campaign.
func TestUpdateClearingTheWindowIsNeverRefusedForClearing(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	req := decodeUpdateReq(t, `{"version":1,"endsAt":null}`)
	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027", req, nil); err != nil {
		t.Fatalf("an open-ended campaign must be writable, got %v", err)
	}
}

// TestUpdateStillMovesTheWindowOfACampaignThatHasGranted is FR-023 restated for the
// tri-state, and it is the negative control for the merged-window check.
//
// The window stays editable on a campaign that has already granted, because each claim
// snapshotted its own `registered_at` and its own budget share (US4). A check that asked
// "has this campaign granted?" before ANY window change would refuse this edit â€” and
// moving `ends_at` is how an operator extends a promotion that is already running.
func TestUpdateStillMovesTheWindowOfACampaignThatHasGranted(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())
	f.repo.markGranted("tet-2027")

	later := baseInstant().AddDate(0, 3, 0)
	if _, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027",
		updateWindowReq(1, strPtr(later.Format(time.RFC3339))), nil); err != nil {
		t.Fatalf("FR-023: the window stays editable after a grant, got %v", err)
	}
	if len(f.audit.all()) != 1 {
		t.Errorf("FR-060: an edit that changes a row records one audit command, got %d",
			len(f.audit.all()))
	}
}

// ---- FR-054: 400 for a malformed timestamp ----

// TestUpdateDistinguishesAMalformedWindowEndFromAClearedOne keeps the two `endsAt`
// requests that BOTH look like "no value" apart in the error code.
//
// `"endsAt": ""` is a MALFORMED timestamp and `"endsAt": null` is the clear request.
// Before the third state existed they were the same Go value; now they must not be the
// same HTTP answer, or an operator who cleared the field would be told their request
// was unreadable.
func TestUpdateDistinguishesAMalformedWindowEndFromAClearedOne(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"an empty string is malformed", `{"version":1,"endsAt":""}`, 400},
		{"a date without a time is malformed", `{"version":1,"endsAt":"2027-01-01"}`, 400},
		{"a zone-less timestamp is malformed", `{"version":1,"endsAt":"2027-01-01T00:00:00"}`, 400},
		{"an explicit null is the clear request", `{"version":1,"endsAt":null}`, 200},
		{"an absent key leaves the window alone", `{"version":1,"quota":120}`, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := seededWindowFixture(t, windowedCampaignRow())

			_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027",
				decodeUpdateReq(t, tc.body), nil)
			if tc.want == 200 {
				if err != nil {
					t.Fatalf("expected the request to be accepted, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected a refusal with %d, got success", tc.want)
			}
			if status := httpStatusOf(t, err); status != tc.want {
				t.Errorf("FR-054: expected %d, got %d (%v)", tc.want, status, err)
			}
		})
	}
}

// TestUpdateRejectsAMalformedWindowEndBeforeOpeningATransaction is the ordering the
// transactional guarantee depends on.
//
// A malformed request must not open a transaction: every typo would take a connection
// from the pool and a lock it does not need, and the audit trail would gain outbox rows
// for requests that never described a change.
func TestUpdateRejectsAMalformedWindowEndBeforeOpeningATransaction(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027",
		decodeUpdateReq(t, `{"version":1,"endsAt":"01-01-2027"}`), nil)
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if f.uow.entries != 0 {
		t.Errorf("a rejected request must not open a transaction, got %d", f.uow.entries)
	}
}

// TestUpdateRejectsANonStringWindowEndAtTheBinder is the one `endsAt` case the use
// case never sees, and the assertion is about WHERE it is answered.
//
// A number is refused by `json.Unmarshal`, so the request never reaches the use case at
// all â€” which is the right layer for it, and the same one every other malformed body in
// the service is stopped at. What is asserted here is that the error is a
// `*json.UnmarshalTypeError` NAMING the field, because `TranslateValidationError` keys
// its standard 400 off exactly that type: a plain error would still be a 400, but
// without the field name, and an operator with two timestamps would not know which one
// to fix.
func TestUpdateRejectsANonStringWindowEndAtTheBinder(t *testing.T) {
	var req sub_dto.CampaignUpdateReq
	err := json.Unmarshal([]byte(`{"version":1,"endsAt":123}`), &req)
	if err == nil {
		t.Fatal("a number in endsAt must be refused by the decoder")
	}
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("the decoder must report *json.UnmarshalTypeError so the standard 400 keeps the "+
			"field name, got %T: %v", err, err)
	}
	if typeErr.Field != "endsAt" {
		t.Errorf("the error must name endsAt, got %q", typeErr.Field)
	}
}

// ---- FR-061: the rollback still covers the new state ----

// TestUpdateRollsBackAClearedWindowWhenTheAuditFails pins FR-061 for the third state.
//
// The audit fake RECORDS and THEN fails, so "nothing is left" is the rollback's doing
// rather than the stub's refusal. It matters here because the cleared window is the one
// edit whose damage would be worst: a campaign silently reopened after a failed request,
// with no audit row saying so.
//
// THE STORED ROW IS NOT ASSERTED HERE, and that is a property of the fixture rather than
// of the rule: `snapshottingUow` models the rollback of the OUTBOX row only, while the
// registry stub's map is not transactional. Asserting the row would test the stub. What
// the fixture CAN prove â€” and what FR-061 is about â€” is that no audit record survives a
// write whose audit failed.
func TestUpdateRollsBackAClearedWindowWhenTheAuditFails(t *testing.T) {
	f := seededWindowFixture(t, windowedCampaignRow())
	f.audit.fail = true

	_, err := f.uc.Update(context.Background(), uuid.New(), "tet-2027",
		decodeUpdateReq(t, `{"version":1,"endsAt":null}`), nil)
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

// ---- helpers ----

// decodeUpdateReq parses a request BODY the way the endpoint parses it.
//
// THROUGH `json.Unmarshal` ON THE DTO, rather than as a Go literal, and that is the
// point of the whole file: the third state of `endsAt` is decided by the decoder and
// recorded on the DTO, so a test that builds the struct by hand would exercise a
// representation no client can produce â€” and the `null` case in particular would be
// unreachable.
func decodeUpdateReq(t *testing.T, body string) *sub_dto.CampaignUpdateReq {
	t.Helper()
	var req sub_dto.CampaignUpdateReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return &req
}

// TestUpdateReqDecodesTheThreeStatesOfEndsAt is the unit-level statement of what the
// recording does, separated from the use case so a failure names the layer.
//
// FOUR BODIES, THREE ANSWERS. If the DTO ever loses the distinction, `{}` and
// `{"endsAt":null}` return the same pair here, and every use-case test that relies on
// the difference starts failing for a reason that is nowhere near the use case.
func TestUpdateReqDecodesTheThreeStatesOfEndsAt(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantCleared bool
		wantValue   bool
	}{
		{"an absent key", `{"version":1}`, false, false},
		{"an explicit null", `{"version":1,"endsAt":null}`, true, false},
		{"a timestamp", `{"version":1,"endsAt":"2027-01-01T00:00:00+07:00"}`, false, true},
		{"an empty string", `{"version":1,"endsAt":""}`, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleared, value := decodeUpdateReq(t, tc.body).EndsAtUpdate()
			if cleared != tc.wantCleared {
				t.Errorf("cleared = %v, want %v (contract Â§4.1)", cleared, tc.wantCleared)
			}
			if (value != nil) != tc.wantValue {
				t.Errorf("value present = %v, want %v", value != nil, tc.wantValue)
			}
		})
	}
}

// TestUpdateReqStillReadsTheOtherFields is the regression guard on the custom decoder.
//
// `UnmarshalJSON` replaces the one the struct would have had, so the obvious way to
// break it is to decode the tri-state field and forget the other four. This asserts the
// whole body still arrives, which is what a caller of `json.Unmarshal` â€” the shared
// binder â€” depends on.
func TestUpdateReqStillReadsTheOtherFields(t *testing.T) {
	planSlug := "premium-yearly"
	body := `{"version":7,"planSlug":"` + planSlug + `","quota":120,"reserve":25,` +
		`"startsAt":"2027-01-01T00:00:00+07:00","endsAt":"2027-02-01T00:00:00+07:00"}`
	req := decodeUpdateReq(t, body)

	if req.Version != 7 {
		t.Errorf("version = %d, want 7", req.Version)
	}
	if req.PlanSlug == nil || *req.PlanSlug != planSlug {
		t.Errorf("planSlug = %v, want %q", req.PlanSlug, planSlug)
	}
	if req.Quota == nil || *req.Quota != 120 {
		t.Errorf("quota = %v, want 120", req.Quota)
	}
	if req.Reserve == nil || *req.Reserve != 25 {
		t.Errorf("reserve = %v, want 25", req.Reserve)
	}
	if req.StartsAt == nil || *req.StartsAt != "2027-01-01T00:00:00+07:00" {
		t.Errorf("startsAt = %v, want the RFC3339 string", req.StartsAt)
	}
	cleared, value := req.EndsAtUpdate()
	if cleared || value == nil || *value != "2027-02-01T00:00:00+07:00" {
		t.Errorf("endsAt must decode as a plain timestamp, got cleared=%v value=%v", cleared, value)
	}
}

package campaign

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	shared_dto "smart-wardrobe-be/shared/application/dto"
)

// 026 T078, quickstart V5: THE READ CONTRACT DID NOT CHANGE EXCEPT BY ADDITION.
//
// FR-050 is the promise 026 makes to every client already reading 025's four read
// endpoints: no field renamed, retyped or removed, and exactly twelve `reason` codes.
// It is the promise most easily broken by accident, because adding a field is the
// normal kind of change — and a RENAMED or dropped field produces a page that renders,
// with a column quietly empty.
//
// WHY A KEY-SET ASSERTION. The twelve `reason` codes are already pinned by
// `TestAccountEligibilityUsesOnlyTheTwelveReasonCodes` (025 T080), so this file does not
// repeat them. What nothing pins is the SHAPE itself, and a shape is exactly a set of
// JSON keys. Marshalling the response and comparing the key set is therefore the
// mechanical form of "no field renamed, retyped or removed": a rename removes one key
// and adds another, a retyped field keeps its key but fails its own consumers, and an
// addition shows up as a key the expected set does not name.
//
// The expected set is written out LONGHAND rather than derived from the struct, and that
// is the whole mechanism: derived from the struct it would compare the struct with itself
// and pass for any shape whatsoever.

// campaignSummaryResponseKeys is `CampaignSummaryRes`'s JSON key set: 025's eighteen,
// plus the three 026 adds (FR-051).
//
// THE THREE NEW ONES ARE MARKED BY THE COMMENT AND NOTHING ELSE, because they are the
// only difference between the two contracts and a reader comparing this list with 025's
// `frontend-guide.md` should see exactly three names.
var campaignSummaryResponseKeys = []string{
	// ---- 025: present and unchanged ----
	"code",
	"planSlug",
	"planName",
	"quota",
	"reserve",
	"grantedMainCount",
	"grantedCompensationCount",
	"grantedTotalCount",
	"remainingMain",
	"remainingCompensation",
	"hardCap",
	"startsAt",
	"endsAt",
	"watermarkAt",
	"status",
	"degraded",
	"degradedReason",
	"lastSweepAt",
	// ---- 026 T039/T050: the three additions (FR-051) ----
	"closedAt",
	"budgetLocked",
	"version",
}

// TestCampaignSummaryResponseKeysAreExactlyTheContract is quickstart V5's read half.
//
// THE VALUE IS POPULATED, not empty, and that is the whole subtlety of the test.
// `degradedReason` carries `omitempty` — 025's deliberate choice, and the reason the
// field may be ABSENT rather than `""` when the mechanism is healthy. Marshalling a zero
// value would therefore assert a key set of 20, and "fixing" that by dropping
// `degradedReason` from the expectation would silently delete a published field from the
// contract this file exists to protect. So every key is made non-zero first: the key set
// of a fully-populated summary is the contract, and the `omitempty` behaviour is asserted
// separately and on purpose below.
//
// THE CLAIM IS BIDIRECTIONAL, and the second direction is the one that catches a
// regression. "Every expected key is present" is satisfied by a struct that gained a
// field nobody documented — which is a broken contract, because a client cannot be told
// about a key its author did not know. So the comparison is of two SETS: a missing key
// AND an unexpected key both fail, with different messages, because they are different
// mistakes.
func TestCampaignSummaryResponseKeysAreExactlyTheContract(t *testing.T) {
	if len(campaignSummaryResponseKeys) != 21 {
		t.Fatalf("the expected key set has %d entries; 025 published 18 and 026 adds 3, so 21 is the "+
			"whole contract", len(campaignSummaryResponseKeys))
	}

	summary := populatedSummary()
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal a populated summary: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode the summary keys: %v", err)
	}

	assertSameKeySet(t, "CampaignSummaryRes", decoded, campaignSummaryResponseKeys)

	// The nullable instants must serialise as JSON `null` rather than be dropped, because
	// for `endsAt`, `closedAt`, `watermarkAt` and `lastSweepAt` the difference between
	// "not set" and "zero time" is the answer the operator is reading (FR-009, FR-059,
	// FR-051). A client that received an absent key could not tell a set value from an
	// unset one without a default, and every such default would be a guess.
	for _, key := range []string{"endsAt", "closedAt", "watermarkAt", "lastSweepAt"} {
		raw, ok := decoded[key]
		if !ok {
			continue // already reported by the key-set comparison
		}
		if string(raw) != "null" {
			t.Errorf("`%s` serialised as %s, want null. These four are nil on this fixture precisely so "+
				"that the null case is what gets checked", key, raw)
		}
	}
}

// TestTheHealthySummaryOmitsDegradedReason is the `omitempty` half of the shape, stated
// as its own claim rather than as an exception inside the other one.
//
// `degradedReason` is the ONE key of `CampaignSummaryRes` that is absent when empty, and
// that is 025's design: a healthy campaign has no failure text, and emitting `""
// anyway would make a client render an empty error column. It is worth pinning because
// the asymmetry is invisible in the type — `DegradedReason string` reads as a plain
// string, and the tag is the only record of it.
//
// If this test fails because the key APPEARS on a healthy summary, do not "fix" it by
// adding the key to the healthy expectation: that would put a permanent empty column on
// every operator screen and change a published contract. The fix is in the tag.
func TestTheHealthySummaryOmitsDegradedReason(t *testing.T) {
	encoded, err := json.Marshal(&sub_dto.CampaignSummaryRes{})
	if err != nil {
		t.Fatalf("marshal an empty summary: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode the summary keys: %v", err)
	}

	if _, present := decoded["degradedReason"]; present {
		t.Errorf("a healthy summary must omit `degradedReason` (025's `omitempty`), got it present")
	}
	// Everything else is present even when zero — the emptiness of a value is not a
	// licence to drop its key, and `quota: 0` in particular is a value a screen must be
	// able to read.
	for _, key := range []string{"code", "quota", "reserve", "grantedTotalCount", "hardCap", "status",
		"degraded", "budgetLocked", "version", "closedAt"} {
		if _, present := decoded[key]; !present {
			t.Errorf("`%s` must be present even when zero; only `degradedReason` carries omitempty", key)
		}
	}
}

// populatedSummary is a summary with every SCALAR field set to a non-zero value.
//
// It exists so the key-set test measures the WHOLE contract rather than the subset that
// survives `omitempty` on a zero value. The four POINTER fields are deliberately left NIL,
// because nil is the case worth checking: they have no `omitempty`, so they must appear as
// JSON `null` rather than be dropped — and "dropped" is the failure a client cannot detect,
// because an absent key and a null key both read as "nothing here" to most code.
func populatedSummary() *sub_dto.CampaignSummaryRes {
	return &sub_dto.CampaignSummaryRes{
		Code:                     "launch-2026-10",
		PlanSlug:                 "premium-monthly",
		PlanName:                 "Premium Tháng",
		Quota:                    50,
		Reserve:                  5,
		GrantedMainCount:         50,
		GrantedCompensationCount: 5,
		GrantedTotalCount:        55,
		RemainingMain:            0,
		RemainingCompensation:    0,
		HardCap:                  55,
		StartsAt:                 baseInstant(),
		// The two scalars 026 added, non-zero so the key-set comparison cannot pass on
		// an `omitempty` they do not carry by accident.
		BudgetLocked: true,
		Version:      3,
		Status:       "closed",
		Degraded:     true,

		DegradedReason: "sweep failed",
		ClosedAt:       nil,
		EndsAt:         nil,
		WatermarkAt:    nil,
		LastSweepAt:    nil,
	}
}

// assertSameKeySet compares two key sets in BOTH directions and says which one failed.
func assertSameKeySet(t *testing.T, what string, got map[string]json.RawMessage, want []string) {
	t.Helper()

	expected := make(map[string]bool, len(want))
	for _, key := range want {
		if expected[key] {
			t.Fatalf("the expected key set lists %q twice; the comparison would be weaker than it looks", key)
		}
		expected[key] = true
	}

	var missing, unexpected []string
	for _, key := range want {
		if _, ok := got[key]; !ok {
			missing = append(missing, key)
		}
	}
	for key := range got {
		if !expected[key] {
			unexpected = append(unexpected, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(unexpected)

	if len(missing) > 0 {
		t.Errorf("%s no longer emits %v. FR-050 promised that no field of the 025 read contract is renamed "+
			"or removed, and a client reading a key that has gone sees an empty value rather than an error "+
			"(quickstart V5)", what, missing)
	}
	if len(unexpected) > 0 {
		t.Errorf("%s emits %v, which no part of the contract documents. An undocumented key is a broken "+
			"contract rather than a harmless addition: the field's author is the only person who knows what "+
			"it means (FR-050, FR-051)", what, unexpected)
	}
}

// TestAuditTrailResponseKeysAreExactlyTheContract is the same claim for the ONE response
// type 026 introduced that a client switches on field by field.
//
// `CampaignAuditLogRes` is new, so FR-050 does not bind it — but its field set is still
// what an audit screen renders, and `targetId` is load-bearing in a way that a shape test
// is the only place to pin: it MUST be present (no `omitempty`) and MUST be null for
// every row, because the campaign code travels in `payload.campaignCode` instead
// (FR-063). A future edit that adds `omitempty` to that field would make the invariant
// invisible, and the guide tells the client the field is always there.
func TestAuditTrailResponseKeysAreExactlyTheContract(t *testing.T) {
	want := []string{
		"id",
		"actorUserId",
		"action",
		"targetType",
		"targetId",
		"payload",
		"createdAt",
		// `requestIp` carries `omitempty` by design — absent when the caller sent none —
		// so it is absent from an EMPTY value and asserted separately below.
	}

	encoded, err := json.Marshal(&sub_dto.CampaignAuditLogRes{})
	if err != nil {
		t.Fatalf("marshal an empty audit row: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode the audit row keys: %v", err)
	}

	if len(decoded) != 7 {
		t.Errorf("an audit row with no values must emit exactly 7 keys, got %d: %v", len(decoded), decoded)
	}
	for _, key := range want {
		if _, ok := decoded[key]; !ok {
			t.Errorf("the audit response does not emit %q. FR-063 makes `targetId` load-bearing: it is "+
				"ALWAYS present and ALWAYS null, and dropping it with `omitempty` would make the invariant "+
				"invisible on the one endpoint that exists to show it", key)
		}
	}

	// THE INVARIANT ITSELF, on the serialised bytes rather than on the struct: the field
	// is present AND spelled `null`.
	raw, present := decoded["targetId"]
	if !present {
		return // already reported above
	}
	if string(raw) != "null" {
		t.Errorf("`targetId` serialised as %s, want null. Every campaign audit row has a NULL target_id "+
			"because the code travels in payload.campaignCode (FR-063)", raw)
	}
}

// TestACampaignThatHasNeverBeenSweptReportsNoSweep is FR-072's other half, and the one a
// client sees first on a brand-new campaign.
//
// FR-072 asks that `lastSweepAt` be NULL before the first sweep, and the distinction it
// protects is the one an operator reads off the page: `lastSweepAt: null` means "no sweep
// has ever covered this campaign", which for a campaign opened a minute ago is CORRECT and
// not a fault. A wrong answer here is worse than a missing one: a sweep-run row written
// speculatively would put a timestamp on a campaign nothing has swept, and the operator
// would conclude the mechanism is healthy when it has not run yet.
//
// The fake's health repo hands back a row with no `LastSweepAt` for a campaign it has
// never seen, which is what makes this reachable at all — and the assertion is on the
// RESPONSE, because the response is the only surface a client has.
func TestACampaignThatHasNeverBeenSweptReportsNoSweep(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	h := newTestQueryUC(t, []*entities.Campaign{testCampaign("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)}, newFakeClaimRepo(), newFakeSweepRunRepo(), newFakeUserBriefContract())

	items, _, err := h.uc.ListCampaigns(t.Context(), 1, 20)
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 campaign, got %d", len(items))
	}

	summary := items[0]
	if summary.LastSweepAt != nil {
		t.Errorf("expected lastSweepAt to be null before the first sweep, got %s (FR-072)", summary.LastSweepAt)
	}
	if summary.Degraded {
		t.Error("expected degraded to be false before any sweep has run; a campaign nobody has swept is not " +
			"a fault (FR-059)")
	}
	if summary.DegradedReason != "" {
		t.Errorf("expected an empty degradedReason before the first sweep, got %q", summary.DegradedReason)
	}

	// `watermarkAt` is the other "never happened yet" field on the same page, and it has
	// the same reading: nil means the main budget has never filled, which is a different
	// state from "filled at the zero instant". Pinning both together keeps the two
	// nullable instants from being confused for each other.
	if summary.WatermarkAt != nil {
		t.Errorf("expected watermarkAt to be null for a campaign that has granted nobody, got %s", summary.WatermarkAt)
	}

	// `closedAt` is the third, and the one 026 added: nil means NEVER force-closed, which
	// is not the same as "still open" — `endsAt` carries that.
	if summary.ClosedAt != nil {
		t.Errorf("expected closedAt to be null for a campaign nobody closed, got %s", summary.ClosedAt)
	}
}

// TestThePaginatedEnvelopeIsUnchangedBy026 is the last of V5's shape claims, and it is
// the one a client library is built on.
//
// The four read endpoints answer inside `PaginationResult`, and 026 changed no envelope
// code — but "no envelope code changed" is a claim about source, whereas a client reads
// the keys. `items` and `metadata` are the two names every generated client type in the
// frontend depends on, and a rename would be invisible in a diff that only looked at the
// campaign files.
func TestThePaginatedEnvelopeIsUnchangedBy026(t *testing.T) {
	encoded, err := json.Marshal(&shared_dto.PaginationResult[*sub_dto.CampaignSummaryRes]{})
	if err != nil {
		t.Fatalf("marshal an empty page: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode the envelope keys: %v", err)
	}

	for _, key := range []string{"items", "metadata"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("the pagination envelope no longer emits %q. Every generated client type in the "+
				"frontend is generated against these two names (FR-050)", key)
		}
	}
	if len(decoded) != 2 {
		t.Errorf("expected the envelope to carry exactly `items` and `metadata`, got %d keys: %v", len(decoded), decoded)
	}

}

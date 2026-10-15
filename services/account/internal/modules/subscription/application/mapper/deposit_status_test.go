package mapper

import (
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
)

// 029 T017 / FR-008 / data-model §2 — the 11-value status contract is a ROUND
// TRIP, and this file is where the two directions of it meet.
//
// WHY THE PAIR IS TESTED TOGETHER. FR-008's invariant is not "the API prints
// eleven strings": it is that every string the API PRINTS can be fed straight
// back as the `status` filter and selects the same status. Two separate
// switches — one for output, one for input, in two different packages — drift
// by exactly one missing case, and nothing notices until an operator filters on
// a value the screen just showed them and gets an empty page. `failed_legacy`
// is the case that WAS missing on the input side, which is why the pair lives
// in one package now (R4/T020).

// depositStatusContract is the table of data-model §2, in numeric order. The
// order is deliberate: it makes a missing row visible as a gap in the sequence,
// which is how the original bug looked.
var depositStatusContract = []struct {
	status depositstatus.DepositStatus
	name   string
}{
	{depositstatus.Pending, "pending"},
	{depositstatus.Success, "success"},
	{depositstatus.FailedLegacy, "failed_legacy"},
	{depositstatus.Creating, "creating"},
	{depositstatus.ReconciliationRequired, "reconciliation_required"},
	{depositstatus.Reconciling, "reconciling"},
	{depositstatus.CreationFailed, "creation_failed"},
	{depositstatus.Cancelled, "cancelled"},
	{depositstatus.Expired, "expired"},
	{depositstatus.InvestigationRequired, "investigation_required"},
	{depositstatus.Refunded, "refunded"},
}

// TestDepositStatusToStringCoversTheElevenContractValues pins the output half:
// one name per numeric status, spelled exactly as the contract spells it.
func TestDepositStatusToStringCoversTheElevenContractValues(t *testing.T) {
	if len(depositStatusContract) != 11 {
		t.Fatalf("data-model §2 lists 11 statuses, the table has %d", len(depositStatusContract))
	}

	seen := map[string]depositstatus.DepositStatus{}
	for _, tc := range depositStatusContract {
		got := DepositStatusToString(tc.status)
		if got != tc.name {
			t.Errorf("DepositStatusToString(%d) = %q, want %q", tc.status, got, tc.name)
		}
		if previous, clash := seen[got]; clash {
			t.Errorf("statuses %d and %d both serialise as %q; every value needs its own name",
				previous, tc.status, got)
		}
		seen[got] = tc.status
	}
}

// TestParseDepositStatusRoundTripsEverySerialisedValue is FR-008's invariant
// read in both directions: parse the printed name back to the same number, and
// print it again unchanged. A filter that dropped `failed_legacy` passes the
// first assertion for the other ten and fails here.
func TestParseDepositStatusRoundTripsEverySerialisedValue(t *testing.T) {
	for _, tc := range depositStatusContract {
		parsed, ok := ParseDepositStatus(tc.name)
		if !ok {
			t.Errorf("FR-008: the filter must accept %q — it is one of the values the API prints, "+
				"so it must be usable as an input", tc.name)
			continue
		}
		if parsed != tc.status {
			t.Errorf("ParseDepositStatus(%q) = %d, want %d", tc.name, parsed, tc.status)
		}
		if back := DepositStatusToString(parsed); back != tc.name {
			t.Errorf("the round trip must close: %q -> %d -> %q", tc.name, parsed, back)
		}
	}
}

// TestParseDepositStatusAcceptsFailedLegacy names the one newly accepted value
// on its own, because it is the whole point of the input-side change: the
// legacy status had a numeric value in the table and a name in every other
// contract, and only the filter could not say it.
func TestParseDepositStatusAcceptsFailedLegacy(t *testing.T) {
	parsed, ok := ParseDepositStatus("failed_legacy")
	if !ok {
		t.Fatal("FR-008: `failed_legacy` must be accepted by the status filter")
	}
	if parsed != depositstatus.FailedLegacy {
		t.Errorf("ParseDepositStatus(failed_legacy) = %d, want %d", parsed, depositstatus.FailedLegacy)
	}
}

// TestParseDepositStatusRejectsNamesOutsideTheContract pins the other side: a
// typo — or the label a value used to have — must be refused rather than
// silently answered with an empty page, which reads as "no such transactions".
func TestParseDepositStatusRejectsNamesOutsideTheContract(t *testing.T) {
	for _, name := range []string{
		"",
		"failed", // the old human label, never a contract value
		"unknown",
		"FAILED_LEGACY",
		"success ",
		"pending\n",
		"0",
		"2",
	} {
		if got, ok := ParseDepositStatus(name); ok {
			t.Errorf("ParseDepositStatus(%q) accepted as %d; only the 11 contract names are filter inputs",
				name, got)
		}
	}
}

// TestDepositStatusToStringNamesValuesOutsideTheContractUnknown records the
// decision data-model §2 leaves open. `status` is a smallint with no CHECK
// constraint, so a row can carry a number outside 0..10 — through a manual
// write or a future migration — and the API must still say something a screen
// can render. The empty string would look like a missing field; a 12th
// contract-looking value would be worse. `unknown` is unmistakably outside the
// contract, and it is deliberately NOT a filter input: a value with no numeric
// meaning cannot round trip.
func TestDepositStatusToStringNamesValuesOutsideTheContractUnknown(t *testing.T) {
	for _, status := range []depositstatus.DepositStatus{depositstatus.DepositStatus(11), -1} {
		if got := DepositStatusToString(status); got != "unknown" {
			t.Errorf("DepositStatusToString(%d) = %q, want %q", status, got, "unknown")
		}
	}
}

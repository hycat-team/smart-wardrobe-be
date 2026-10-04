package mapper

import (
	"reflect"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// 026 T050 (FR-020, FR-051) — `MapCampaignSummaryRes`'s two FR-051 fields.
//
// WHY THESE ARE TESTED HERE AND NOT IN A USE CASE. Both are DERIVED INSIDE this
// function — one from the row, one from the counts the caller already had — and every
// caller in the codebase goes through here (it is the only place a
// `CampaignSummaryRes` is built). A use-case test would prove one caller's arithmetic;
// this proves the single implementation all of them share.
//
// WHAT CANNOT BE TESTED HERE, AND IS NOT: whether the counts are RIGHT. They come from
// the ledger through the query use case, and a mapper test that asserted a count would
// be asserting its own argument.

func summaryRow() *entities.Campaign {
	return &entities.Campaign{
		Code:     "tet-2027",
		PlanSlug: "premium-monthly",
		Quota:    100,
		Reserve:  10,
		StartsAt: time.Date(2027, 1, 20, 0, 0, 0, 0, time.UTC),
		Version:  4,
	}
}

// TestBudgetLockedIsDerivedFromTheCounts is FR-020 expressed as the condition the spec
// states it by.
//
// The three interesting cases are the ones a naive implementation gets wrong:
//
//   - grants in the MAIN bucket lock the budget, which is the obvious case;
//   - grants in the RESERVE bucket lock it too, even though `quota` was never touched —
//     `campaign_claims` froze which part of the budget each grant drew on, so a grant
//     from the reserve is exactly as much a reason to freeze the row. Reading only the
//     main count would leave such a campaign editable;
//   - a campaign whose main budget is FULL but whose reserve still has room reports
//     `compensation`, and is locked all the same.
func TestBudgetLockedIsDerivedFromTheCounts(t *testing.T) {
	cases := []struct {
		name   string
		counts CampaignGrantCounts
		want   bool
	}{
		{"nobody has been granted", CampaignGrantCounts{}, false},
		{"one grant from the main bucket", CampaignGrantCounts{Main: 1}, true},
		{"one grant from the reserve bucket", CampaignGrantCounts{Compensation: 1}, true},
		{"both buckets", CampaignGrantCounts{Main: 100, Compensation: 10}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapCampaignSummaryRes(summaryRow(), tc.counts, nil, sub_dto.CampaignStatusRunning)
			if got.BudgetLocked != tc.want {
				t.Errorf("budgetLocked = %v, want %v for %+v", got.BudgetLocked, tc.want, tc.counts)
			}
		})
	}
}

// TestVersionIsCopiedFromTheRowIsFR-051's other half, and the COPY is the requirement.
//
// FR-055 makes the response's `version` the token the client sends back, so a value
// that came from anywhere else would make the next edit fail for its own success. The
// row the mapper is handed is the row a write re-read afterwards, which is the only
// place `version + 1` is knowable.
func TestVersionIsCopiedFromTheRow(t *testing.T) {
	got := MapCampaignSummaryRes(summaryRow(), CampaignGrantCounts{}, nil, sub_dto.CampaignStatusRunning)
	if got.Version != 4 {
		t.Errorf("version = %d, want the row's 4", got.Version)
	}
}

// TestBudgetLockedAndVersionAreAlwaysEmitted is FR-050's compatibility rule read from
// the JSON side.
//
// FR-051 is ADDITIVE: three new keys, and a client that does not know them ignores
// them. That only holds if the keys are ALWAYS present — a `false` that a
// `omitempty` swallowed would leave a pre-026 client unable to tell "not locked" from
// "this server does not know", and the one place that matters is a screen deciding
// whether to disable the budget inputs.
func TestBudgetLockedAndVersionAreAlwaysEmitted(t *testing.T) {
	res := MapCampaignSummaryRes(summaryRow(), CampaignGrantCounts{}, nil, sub_dto.CampaignStatusRunning)

	resType := reflect.TypeOf(*res)
	for _, field := range []struct{ goName, jsonTag string }{
		{"BudgetLocked", "budgetLocked"},
		{"Version", "version"},
		{"ClosedAt", "closedAt"},
	} {
		structField, found := resType.FieldByName(field.goName)
		if !found {
			t.Errorf("FR-051: CampaignSummaryRes must carry %s", field.goName)
			continue
		}
		tag := structField.Tag.Get("json")
		if tag != field.jsonTag {
			t.Errorf("FR-051: %s must be tagged `json:%q`, got %q", field.goName, field.jsonTag, tag)
		}
		if tag == "" {
			t.Errorf("FR-051: %s must carry a JSON tag — an untagged field serialises as %q",
				field.goName, field.goName)
		}
	}
}

// TestANilCampaignMapsToNil is the guard that lets the query use case hand the mapper a
// possibly-absent row without branching first.
func TestANilCampaignMapsToNil(t *testing.T) {
	if got := MapCampaignSummaryRes(nil, CampaignGrantCounts{}, nil, sub_dto.CampaignStatusRunning); got != nil {
		t.Errorf("a nil campaign must map to nil, got %+v", got)
	}
}

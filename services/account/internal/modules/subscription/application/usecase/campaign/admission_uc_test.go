package campaign

import (
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 026 T023: conditions 1-7 evaluated over REGISTRY ROWS, not configuration values.
//
// Before 026 these rules read `sharedconfig.CampaignConfig`, whose boundaries were
// RFC3339 STRINGS parsed once at load time. They now read `*entities.Campaign`,
// whose boundaries are `time.Time`/`*time.Time` read from `TIMESTAMPTZ` columns at
// run time. Three properties have to be re-established rather than assumed, because
// each one is a thing the old type could not express:
//
//   - a closed campaign is EXCLUDED, because `closed_at` did not exist before
//     (FR-031). A campaign whose window is still open but which an operator closed
//     must grant nobody.
//   - the `code DESC` tie-break is DETERMINISTIC, because two campaigns can now
//     share a `starts_at` and the winner used to depend on the order somebody wrote
//     the YAML in (FR-040/FR-041).
//   - the window still judges the ACCOUNT by its own creation instant, never by the
//     processing clock (FR-078) — unchanged, and worth pinning because it is the
//     property that makes the rule pure.

func evaluatorOver(campaigns []*entities.Campaign, now time.Time) *campaignEvaluator {
	return newCampaignEvaluator(campaigns, now)
}

// chosenCode runs one evaluation and returns the campaign code it picked.
func chosenCode(t *testing.T, campaigns []*entities.Campaign, now, registeredAt time.Time) string {
	t.Helper()
	verdict := evaluatorOver(campaigns, now).
		evaluate(t.Context(), uuid.New(), registeredAt, false, nil)
	if verdict.Campaign == nil {
		return ""
	}
	return verdict.Campaign.Code
}

// TestEvaluatorPicksTheCampaignThatStartsLatest is FR-040's primary rule, and it
// is unchanged by the move: the newest-starting candidate wins.
func TestEvaluatorPicksTheCampaignThatStartsLatest(t *testing.T) {
	now := baseInstant()
	early := now.Add(-48 * time.Hour)
	late := now.Add(-24 * time.Hour)

	got := chosenCode(t, []*entities.Campaign{
		testCampaign("older-campaign", "premium-monthly", 50, 5, &early, nil),
		testCampaign("newer-campaign", "premium-quarterly", 10, 2, &late, nil),
	}, now, now.Add(-time.Hour))

	if got != "newer-campaign" {
		t.Errorf("selected %q, want %q — FR-040 gives the account to the LATEST-STARTING candidate",
			got, "newer-campaign")
	}
}

// TestEvaluatorBreaksATieByCodeDescending is FR-041's determinism requirement and
// the rule 026 added.
//
// Two campaigns with the same `starts_at` are now ORDINARY: an operator creates
// them from a browser, in any order, minutes apart. Without a tie-break the same
// account would land in whichever row the database happened to return first, and
// FR-041 requires the choice to be a function of (rows, registered_at) and of
// nothing else.
func TestEvaluatorBreaksATieByCodeDescending(t *testing.T) {
	now := baseInstant()
	shared := now.Add(-24 * time.Hour)
	candidates := []*entities.Campaign{
		testCampaign("tie-aaa", "premium-monthly", 50, 5, &shared, nil),
		testCampaign("tie-zzz", "premium-monthly", 50, 5, &shared, nil),
		testCampaign("tie-mmm", "premium-monthly", 50, 5, &shared, nil),
	}

	// Reversed input order, and the answer must not move: the tie-break is in the
	// comparison, not in the order the rows arrived in.
	forward := chosenCode(t, candidates, now, now.Add(-time.Hour))
	reversed := chosenCode(t, []*entities.Campaign{candidates[0], candidates[2], candidates[1]}, now, now.Add(-time.Hour))

	if forward != "tie-zzz" {
		t.Errorf("selected %q, want %q — code DESC is the documented tie-break", forward, "tie-zzz")
	}
	if reversed != forward {
		t.Errorf("the same rows in a different order selected %q and %q. FR-041 requires the "+
			"choice to be a function of (rows, registered_at) alone", reversed, forward)
	}
}

// TestEvaluatorExcludesForceClosedCampaigns is FR-031, and the reason the closed
// column had to be added at all.
//
// The closed campaigns here still have OPEN windows. If the answer were "asleep
// because the window closed", the test would pass for the wrong reason and the
// closed column would be untested.
func TestEvaluatorExcludesForceClosedCampaigns(t *testing.T) {
	now := baseInstant()
	startsAt := now.Add(-48 * time.Hour)
	endsAt := now.Add(48 * time.Hour)

	closed := closedCampaign("closed-campaign", "premium-monthly", 50, 5, &startsAt, &endsAt)
	if closed.ClosedAt == nil {
		t.Fatal("precondition: the fixture must carry a closed_at")
	}
	if closed.EndsAt == nil || !now.Before(*closed.EndsAt) {
		t.Fatal("precondition: the closed campaign's window must still be OPEN, or this " +
			"test would pass because of the window rather than because of the closed column")
	}

	evaluator := evaluatorOver([]*entities.Campaign{closed}, now)
	if evaluator.isAwake(closed) {
		t.Error("FR-031: a force-closed campaign must not be awake while its window is open")
	}

	verdict := evaluator.evaluate(t.Context(), uuid.New(), now.Add(-time.Hour), false, nil)
	if verdict.Campaign != nil {
		t.Errorf("selected %q from a closed campaign; FR-031 says a closed campaign grants nobody",
			verdict.Campaign.Code)
	}
	// The reason code is the one 025 already used for "this campaign is over", which
	// is what keeps the published set of twelve `reason` codes unchanged (FR-050).
	if verdict.Reason != SkipCampaignSleeping {
		t.Errorf("reason = %q, want %q — a closed campaign is asleep, not merely unconfigured",
			verdict.Reason, SkipCampaignSleeping)
	}
}

// TestEvaluatorPrefersAnOpenCampaignOverAClosedOneWithTheSameWindow is the case
// that actually happens: an operator closes a running campaign while another one
// is live. The closed one must lose, even though both windows are open and both
// contain the account.
func TestEvaluatorPrefersAnOpenCampaignOverAClosedOneWithTheSameWindow(t *testing.T) {
	now := baseInstant()
	shared := now.Add(-24 * time.Hour)

	got := chosenCode(t, []*entities.Campaign{
		closedCampaign("closed-campaign", "premium-monthly", 50, 5, &shared, nil),
		testCampaign("open-campaign", "premium-monthly", 50, 5, &shared, nil),
	}, now, now.Add(-time.Hour))

	if got != "open-campaign" {
		t.Errorf("selected %q, want %q — FR-031 excludes a closed campaign from the candidate set "+
			"entirely, so the tie-break never gets to compare the two codes", got, "open-campaign")
	}
}

// TestEvaluatorJudgesTheWindowByRegistrationNotByTheClock is FR-078, unchanged by
// the move and worth pinning because the entity now carries plain `time.Time`
// values where the configuration carried pre-normalised ones.
//
// An account registered INSIDE the window must still be admitted days later, and an
// account registered OUTSIDE it must be refused even while the campaign is running:
// the two questions are different and using "now" for both would discard an account
// for no reason at all.
func TestEvaluatorJudgesTheWindowByRegistrationNotByTheClock(t *testing.T) {
	now := baseInstant()
	// The campaign is STILL AWAKE, so condition 2 passes and the window test is what
	// decides. Using a campaign whose own window has expired would make the test
	// pass on condition 2 and prove nothing about the account's instant.
	startsAt := now.Add(-30 * 24 * time.Hour)
	endsAt := now.Add(24 * time.Hour)
	campaign := testCampaign("windowed-campaign", "premium-monthly", 50, 5, &startsAt, &endsAt)

	// Registered thirty days ago, inside the window. A rule that compared the
	// account against "now" would still admit this one, so the positive case alone is
	// weak; the negative case below is what pins the two instants apart.
	if got := chosenCode(t, []*entities.Campaign{campaign}, now, startsAt.Add(time.Hour)); got != "windowed-campaign" {
		t.Errorf("selected %q for an account registered inside the window, want the campaign — "+
			"the window is judged on the account's own creation instant", got)
	}

	// Registered BEFORE the window opened while the campaign is running: refused with
	// the out-of-window code. Judging this one by "now" would have granted it.
	verdict := evaluatorOver([]*entities.Campaign{campaign}, now).
		evaluate(t.Context(), uuid.New(), startsAt.Add(-time.Hour), false, nil)
	if verdict.Campaign != nil {
		t.Errorf("selected %q for an account registered BEFORE the campaign opened", verdict.Campaign.Code)
	}
	if verdict.Reason != SkipOutsideWindow {
		t.Errorf("reason = %q, want %q", verdict.Reason, SkipOutsideWindow)
	}

	// Registered AFTER the window closes, in the future: also out of window, and a
	// clock-based rule would have granted it because the campaign is still awake.
	late := endsAt.Add(time.Hour)
	lateVerdict := evaluatorOver([]*entities.Campaign{campaign}, now).
		evaluate(t.Context(), uuid.New(), late, false, nil)
	if lateVerdict.Campaign != nil {
		t.Errorf("selected %q for an account registered AFTER the campaign closes", lateVerdict.Campaign.Code)
	}
}

// TestTheFakeRegistryAndTheEvaluatorAgreeOnAwake keeps the two definitions of
// "awake" from drifting apart without either test hiding inside the other.
//
// `isCampaignAwake` in the fakes is a deliberately SEPARATE copy of
// data-model.md §4.3. If it agreed with the evaluator only because it called the
// evaluator, a bug in `isAwake` would make the fakes agree with it and every
// behaviour test would still pass.
func TestTheFakeRegistryAndTheEvaluatorAgreeOnAwake(t *testing.T) {
	now := baseInstant()
	openStart := now.Add(-24 * time.Hour)
	futureStart := now.Add(24 * time.Hour)
	pastEnd := now.Add(-time.Hour)
	futureEnd := now.Add(24 * time.Hour)

	cases := []struct {
		name     string
		campaign *entities.Campaign
		want     bool
	}{
		{"open-ended and running", testCampaign("a", "premium-monthly", 1, 0, &openStart, nil), true},
		{"window still ahead", testCampaign("b", "premium-monthly", 1, 0, &openStart, &futureEnd), true},
		{"window past", testCampaign("c", "premium-monthly", 1, 0, &openStart, &pastEnd), false},
		{"not started yet", testCampaign("d", "premium-monthly", 1, 0, &futureStart, nil), false},
		{"force closed, window open", closedCampaign("e", "premium-monthly", 1, 0, &openStart, &futureEnd), false},
		{"starts exactly now", testCampaign("f", "premium-monthly", 1, 0, &now, nil), true},
		{"ends exactly now", testCampaign("g", "premium-monthly", 1, 0, &openStart, &now), false},
	}

	evaluator := evaluatorOver(nil, now)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			production := evaluator.isAwake(tc.campaign)
			fake := isCampaignAwake(tc.campaign, now)
			if production != tc.want {
				t.Errorf("campaignEvaluator.isAwake = %v, want %v (data-model.md §4.3)", production, tc.want)
			}
			if fake != production {
				t.Errorf("the test fake's copy says %v while the evaluator says %v; the two "+
					"definitions of \"awake\" have drifted and every behaviour test would still pass",
					fake, production)
			}
		})
	}
}

// TestEvaluatorReportsAnEmptyRegistryAsNotConfigured is SC-008's behavioural half,
// and the reason the reason code list did not grow a thirteenth entry.
//
// An empty registry is a NORMAL state after the cut-over — a fresh environment has
// none until an operator opens one — so it must be a quiet skip, not an error and
// not a warning.
func TestEvaluatorReportsAnEmptyRegistryAsNotConfigured(t *testing.T) {
	verdict := evaluatorOver(nil, baseInstant()).
		evaluate(t.Context(), uuid.New(), baseInstant(), false, nil)
	if verdict.Campaign != nil {
		t.Fatalf("an empty registry must select nothing, got %q", verdict.Campaign.Code)
	}
	if verdict.Reason != SkipCampaignNotConfigured {
		t.Errorf("reason = %q, want %q", verdict.Reason, SkipCampaignNotConfigured)
	}
}

// TestEvaluatorStillSkipsAdministratorCreatedAccounts is FR-091, pinned here
// because the fixture list changed type and the predicate must not have moved with
// it: condition 3 is checked BEFORE the campaign is chosen, so it costs no
// selection work and reports its own code.
func TestEvaluatorStillSkipsAdministratorCreatedAccounts(t *testing.T) {
	now := baseInstant()
	startsAt := now.Add(-time.Hour)

	verdict := evaluatorOver([]*entities.Campaign{
		testCampaign("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil),
	}, now).evaluate(t.Context(), uuid.New(), now, true, nil)

	if verdict.Campaign != nil {
		t.Fatalf("an administrator's account must not be granted from any campaign")
	}
	if verdict.Reason != SkipCreatedByAdmin {
		t.Errorf("reason = %q, want %q", verdict.Reason, SkipCreatedByAdmin)
	}
}

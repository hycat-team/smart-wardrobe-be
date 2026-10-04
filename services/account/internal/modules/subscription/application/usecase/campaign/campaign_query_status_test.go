package campaign

import (
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 025 T079d: the campaign state machine (SC-012).
//
// WHY THIS TEST IS ABOUT PAIRS RATHER THAN VALUES. A state machine whose five
// states were checked one at a time would pass even if two of them were the same
// string, and the pair that matters is exactly the pair a single-value assertion
// cannot see:
//
//   - `compensation` vs `running` — the difference is whether the RESERVE is still
//     usable. That budget exists for the accounts the main budget turned away
//     (FR-031), so an operator told "running" for a campaign that is actually in
//     compensation will believe giveaways are still flowing through the main
//     budget, and will tell a customer to keep trying.
//   - `exhausted` vs `expired` — the difference is whether the campaign ran out or
//     was CLOSED. One is "we gave everything away"; the other is "we stopped". An
//     operator who cannot tell them apart either keeps advertising a closed
//     campaign or reopens a finished one.
//
// The grant path's own rules can tell `running` from `compensation` without any of
// this: it reads the watermark. The look-up CANNOT — it is a separate call that
// arrives after the campaign has settled, and it has to reconstruct the state from
// counts alone. That asymmetry is why the machine needs its own test.

// seededCampaign builds a configured campaign plus a claim ledger in a chosen
// state.
//
// `main` and `reserve` are how many grants exist in each part; the last main grant
// carries the watermark when the main budget is full, because the watermark is
// written by the grant that FILLED it (FR-030) and a ledger claiming the budget
// is full without one is the anomaly state.
func seededCampaign(t *testing.T,
	quota, reserve int,
	startsAt, endsAt *time.Time,
	main, reserveGrants int,
) (*queryHarness, *entities.Campaign) {
	campaign := testCampaign(queryCode, "premium-monthly", quota, reserve, startsAt, endsAt)
	claims := newFakeClaimRepo()
	for i := 0; i < main; i++ {
		claims.seed(
			queryCode,
			uuid.New(),
			baseInstant().Add(-time.Duration(main-i)*time.Minute),
			entities.GrantBucketMain,
			// The watermark is the grant that brought the main count TO the quota,
			// so only the last of them carries it.
			i == main-1 && main == quota,
		)
	}
	for i := 0; i < reserveGrants; i++ {
		claims.seed(
			queryCode,
			uuid.New(),
			baseInstant().Add(-time.Duration(reserveGrants-i)*time.Second),
			entities.GrantBucketReserve,
			false,
		)
	}
	h := newTestQueryUC(
		t,
		[]*entities.Campaign{campaign},
		claims,
		newFakeSweepRunRepo(),
		newFakeUserBriefContract(),
	)
	return h, campaign
}

// TestCampaignStatusDistinguishesRunningFromCompensation is T079d / SC-012.
//
// Two campaigns, identical except for how full the main budget is. The assertion
// is that their statuses DIFFER, and then that each is the expected one — the
// second half alone would pass if the machine answered `compensation` for
// everything.
func TestCampaignStatusDistinguishesRunningFromCompensation(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)

	// quota 2, one main grant: the budget is not full, so the reserve is not even
	// reachable yet (FR-032) and the campaign is still running.
	running, _ := seededCampaign(t, queryQuota, queryReserve, &startsAt, nil, 1, 0)
	// quota 2, two main grants: the main budget is full, the watermark is frozen
	// and the reserve still has room — which is what `compensation` names.
	compensation, _ := seededCampaign(t, queryQuota, queryReserve, &startsAt, nil, queryQuota, 0)

	runningSummary := summaryOf(t, running)
	compensationSummary := summaryOf(t, compensation)

	if runningSummary.Status == compensationSummary.Status {
		t.Fatalf("FR-030/SC-012: a campaign whose main budget is not full and one whose main budget is full "+
			"both reported %q — the reserve is usable in exactly one of them, and an operator cannot tell which",
			runningSummary.Status)
	}
	if runningSummary.Status != sub_dto.CampaignStatusRunning {
		t.Errorf("expected %q with the main budget not full, got %q",
			sub_dto.CampaignStatusRunning, runningSummary.Status)
	}
	if compensationSummary.Status != sub_dto.CampaignStatusCompensation {
		t.Errorf("expected %q once the watermark is frozen and the reserve has room, got %q",
			sub_dto.CampaignStatusCompensation, compensationSummary.Status)
	}
}

// TestCampaignStatusDistinguishesExhaustedFromExpired is T079d / SC-012, and it
// is the pair a single-value test would miss.
//
// `exhausted` is a BUDGET fact: quota + reserve grants exist. `expired` is a CLOCK
// fact: `ends_at` has passed. They are separate states because they need different
// people to act — one means "we gave everything away", the other "we closed it" —
// and an operator who sees the wrong one either keeps selling a closed campaign or
// reopens a finished one.
func TestCampaignStatusDistinguishesExhaustedFromExpired(t *testing.T) {
	startsAt := baseInstant().Add(-24 * time.Hour)
	futureEnd := baseInstant().Add(24 * time.Hour)
	pastEnd := baseInstant().Add(-time.Hour)

	// The hard ceiling reached, with the campaign still inside its window: nothing
	// more can be granted, so it sleeps on its own (FR-013).
	exhausted, _ := seededCampaign(
		t, queryQuota, queryReserve, &startsAt, &futureEnd,
		queryQuota, queryReserve,
	)
	// The window closed with budget to spare. `ends_at` is honoured whether or not
	// the budget is spent (FR-009, FR-013), so this must NOT read as `running`
	// just because there is room left.
	expired, _ := seededCampaign(
		t, queryQuota, queryReserve, &startsAt, &pastEnd,
		1, 0,
	)

	exhaustedSummary := summaryOf(t, exhausted)
	expiredSummary := summaryOf(t, expired)

	if exhaustedSummary.Status == expiredSummary.Status {
		t.Fatalf("SC-012: a campaign that ran out of budget and one that ran out of time both reported %q; "+
			"they need different people to act", exhaustedSummary.Status)
	}
	if exhaustedSummary.Status != sub_dto.CampaignStatusExhausted {
		t.Errorf("expected %q at the hard ceiling quota+reserve, got %q",
			sub_dto.CampaignStatusExhausted, exhaustedSummary.Status)
	}
	if expiredSummary.Status != sub_dto.CampaignStatusExpired {
		t.Errorf("expected %q past ends_at, got %q — the window closes whether or not the budget is spent (FR-013)",
			sub_dto.CampaignStatusExpired, expiredSummary.Status)
	}
}

// TestCampaignStatusCoversAllFiveStates is T079d / SC-012.
//
// A single test that walks the machine end to end, so a state that is never
// reachable is caught. Each of the five has a condition that produces it and at
// least one that must NOT.
func TestCampaignStatusCoversAllFiveStates(t *testing.T) {
	futureStart := baseInstant().Add(time.Hour)
	pastStart := baseInstant().Add(-time.Hour)
	futureEnd := baseInstant().Add(time.Hour)
	pastEnd := baseInstant().Add(-time.Hour)

	cases := []struct {
		name string
		want string
		// build returns a harness seeded into the state under test.
		build func(t *testing.T) *queryHarness
	}{
		{
			// Now is before `starts_at`. Nothing can be granted, and reporting
			// anything else would tell an operator the campaign is live.
			name: "not started",
			want: sub_dto.CampaignStatusNotStarted,
			build: func(t *testing.T) *queryHarness {
				h, _ := seededCampaign(t, queryQuota, queryReserve, &futureStart, &futureEnd, 0, 0)
				return h
			},
		},
		{
			name: "running",
			want: sub_dto.CampaignStatusRunning,
			build: func(t *testing.T) *queryHarness {
				h, _ := seededCampaign(t, queryQuota, queryReserve, &pastStart, &futureEnd, 1, 0)
				return h
			},
		},
		{
			name: "compensation",
			want: sub_dto.CampaignStatusCompensation,
			build: func(t *testing.T) *queryHarness {
				h, _ := seededCampaign(t, queryQuota, queryReserve, &pastStart, &futureEnd, queryQuota, 0)
				return h
			},
		},
		{
			name: "exhausted",
			want: sub_dto.CampaignStatusExhausted,
			build: func(t *testing.T) *queryHarness {
				h, _ := seededCampaign(t, queryQuota, queryReserve, &pastStart, &futureEnd, queryQuota, queryReserve)
				return h
			},
		},
		{
			// Past `ends_at` AND at the ceiling. `expired` wins, and the reason is
			// in the implementation: a closed campaign cannot be revived by a
			// budget edit, so reporting it as `exhausted` would invite exactly that.
			name: "expired",
			want: sub_dto.CampaignStatusExpired,
			build: func(t *testing.T) *queryHarness {
				h, _ := seededCampaign(t, queryQuota, queryReserve, &pastStart, &pastEnd, queryQuota, queryReserve)
				return h
			},
		},
	}

	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := summaryOf(t, tc.build(t)).Status
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
			seen[tc.want] = true
		})
	}

	// Every state of the machine must be reachable. Five is not a number that
	// happens by accident: it is what `data-model.md` §5 draws, and a state that
	// no configuration can produce is a state the contract promises and the code
	// does not deliver.
	for _, want := range []string{
		sub_dto.CampaignStatusNotStarted,
		sub_dto.CampaignStatusRunning,
		sub_dto.CampaignStatusCompensation,
		sub_dto.CampaignStatusExhausted,
		sub_dto.CampaignStatusExpired,
	} {
		if !seen[want] {
			t.Errorf("no scenario produced %q; the machine does not cover all five states of data-model.md §5", want)
		}
	}
}

// TestCampaignCountsAreDerivedNotStored is FR-065 / FR-066.
//
// The two bucket counts must be INDEPENDENT numbers read from `grant_bucket`, and
// the response must not carry a counter that could drift from them. This asserts
// the arithmetic an operator reads — the total, the two remainders, the hard cap —
// because a wrong budget number is worse than no number: it is acted upon.
func TestCampaignCountsAreDerivedNotStored(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	h, _ := seededCampaign(t, queryQuota, queryReserve, &startsAt, nil, queryQuota, queryReserve)

	summary := summaryOf(t, h)

	if summary.GrantedMainCount != queryQuota {
		t.Errorf("expected %d main grants, got %d", queryQuota, summary.GrantedMainCount)
	}
	if summary.GrantedCompensationCount != queryReserve {
		t.Errorf("expected %d reserve grants, got %d", queryReserve, summary.GrantedCompensationCount)
	}
	if summary.GrantedTotalCount != queryQuota+queryReserve {
		t.Errorf("expected a total of %d, got %d", queryQuota+queryReserve, summary.GrantedTotalCount)
	}
	if summary.RemainingMain != 0 || summary.RemainingCompensation != 0 {
		t.Errorf("expected no slots left, got main=%d reserve=%d",
			summary.RemainingMain, summary.RemainingCompensation)
	}
	if summary.HardCap != queryQuota+queryReserve {
		t.Errorf("FR-022: expected the hard ceiling to be quota+reserve = %d, got %d",
			queryQuota+queryReserve, summary.HardCap)
	}
	// FR-022 is the invariant the client can now check, and it is the reason
	// `grantedTotalCount` exists on the response at all.
	if summary.GrantedTotalCount > summary.HardCap {
		t.Errorf("FR-022: %d grants exceed the hard ceiling %d", summary.GrantedTotalCount, summary.HardCap)
	}
	if summary.WatermarkAt == nil {
		t.Errorf("the main budget is full, so the watermark must be set (FR-030)")
	}
}

// TestCampaignRemainingNeverGoesNegative is the degenerate-config guard.
//
// A quota lowered on a LIVE campaign — which data-model.md §3.1 explicitly allows
// as an operational mistake — puts the ledger above the configured budget. The
// arithmetic then goes negative, and a negative "slots remaining" on a page an
// operator reads at a glance is a number that invites a support ticket about a
// campaign that is simply finished.
func TestCampaignRemainingNeverGoesNegative(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	// The ledger holds three main grants; the configuration now says quota 2.
	h, _ := seededCampaign(t, queryQuota, queryReserve, &startsAt, nil, 3, 0)

	summary := summaryOf(t, h)

	if summary.RemainingMain < 0 || summary.RemainingCompensation < 0 {
		t.Fatalf("a budget edited below what was already granted must not produce a negative remainder: "+
			"main=%d reserve=%d", summary.RemainingMain, summary.RemainingCompensation)
	}
	// It is already at or past the ceiling, so the machine must say so rather than
	// reporting a campaign that is still handing out plans.
	if summary.Status != sub_dto.CampaignStatusExhausted {
		t.Errorf("expected %q once the ledger is at the ceiling, got %q",
			sub_dto.CampaignStatusExhausted, summary.Status)
	}
}

// TestClosedOutranksEveryOtherStatus is T046 / FR-032 / SC-002, and it is the case a
// single-value test of `closed` would miss.
//
// `closed` is not a sixth alternative to the five existing states; it is checked
// BEFORE all of them, so it wins even when the clock and the budget would both have
// produced a different answer. Every combination below is one where the old machine
// had a perfectly good answer of its own — `expired`, `exhausted`, `compensation`,
// `running`, `not_started` — and the one thing a force-close means is that the
// operator's decision outranks all of them.
//
// WHY EACH WRONG ANSWER WOULD COST SOMEBODY SOMETHING. Reporting `running` for a
// closed campaign tells an operator the promotion is still handing out plans, which is
// the exact promise FR-031 withdraws. Reporting `expired` invites reopening it, and
// there is no reopen (FR-034). Reporting `not_started` for a campaign that has not
// opened is merely confusing — but it is the same confusion that would make an
// operator wait for a launch that has been cancelled.
func TestClosedOutranksEveryOtherStatus(t *testing.T) {
	pastStart := baseInstant().Add(-24 * time.Hour)
	futureStart := baseInstant().Add(time.Hour)
	pastEnd := baseInstant().Add(-time.Hour)
	futureEnd := baseInstant().Add(time.Hour)

	cases := []struct {
		name string
		// the state the five-state machine would have reported WITHOUT the close.
		otherwise string
		build     func(t *testing.T) *queryHarness
	}{
		{
			name:      "closed rather than running",
			otherwise: sub_dto.CampaignStatusRunning,
			build: func(t *testing.T) *queryHarness {
				startsAt := pastStart
				closed := closedCampaign(queryCode, "premium-monthly", queryQuota, queryReserve, &startsAt, &futureEnd)
				h, _ := seededClosed(t, closed, 1, 0)
				return h
			},
		},
		{
			name:      "closed rather than not_started",
			otherwise: sub_dto.CampaignStatusNotStarted,
			build: func(t *testing.T) *queryHarness {
				startsAt := futureStart
				closed := closedCampaign(queryCode, "premium-monthly", queryQuota, queryReserve, &startsAt, &futureEnd)
				h, _ := seededClosed(t, closed, 0, 0)
				return h
			},
		},
		{
			name:      "closed rather than compensation",
			otherwise: sub_dto.CampaignStatusCompensation,
			build: func(t *testing.T) *queryHarness {
				startsAt := pastStart
				closed := closedCampaign(queryCode, "premium-monthly", queryQuota, queryReserve, &startsAt, &futureEnd)
				h, _ := seededClosed(t, closed, queryQuota, 0)
				return h
			},
		},
		{
			name:      "closed rather than exhausted",
			otherwise: sub_dto.CampaignStatusExhausted,
			build: func(t *testing.T) *queryHarness {
				startsAt := pastStart
				closed := closedCampaign(queryCode, "premium-monthly", queryQuota, queryReserve, &startsAt, &futureEnd)
				h, _ := seededClosed(t, closed, queryQuota, queryReserve)
				return h
			},
		},
		{
			// The pair that matters most: `expired` is what a campaign past `ends_at`
			// reports, and it reads as "it finished on its own". A closure reads as
			// "somebody stopped it", which is a different event with a different
			// explanation.
			name:      "closed rather than expired",
			otherwise: sub_dto.CampaignStatusExpired,
			build: func(t *testing.T) *queryHarness {
				startsAt := pastStart
				closed := closedCampaign(queryCode, "premium-monthly", queryQuota, queryReserve, &startsAt, &pastEnd)
				h, _ := seededClosed(t, closed, queryQuota, queryReserve)
				return h
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := summaryOf(t, tc.build(t))
			if summary.Status != sub_dto.CampaignStatusClosed {
				t.Fatalf("a force-closed campaign must report %q whatever the clock and the budget say, got %q",
					sub_dto.CampaignStatusClosed, summary.Status)
			}
			if summary.Status == tc.otherwise {
				t.Errorf("the close had no effect on the reported state")
			}
			// FR-051: `closedAt` rides along with the status, so a screen can show WHEN
			// it was closed rather than only that it is.
			if summary.ClosedAt == nil {
				t.Errorf("a campaign reported as %q must carry closedAt", sub_dto.CampaignStatusClosed)
			}
		})
	}
}

// TestACampaignThatWasNeverClosedStillReportsNoClosedAt is the negative control.
//
// Without it, "every closed campaign carries `closedAt`" would also be satisfied by a
// mapper that sets the field unconditionally, and an operator would see a campaign
// that is running with a closure instant on it.
func TestACampaignThatWasNeverClosedStillReportsNoClosedAt(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	h, _ := seededCampaign(t, queryQuota, queryReserve, &startsAt, nil, 1, 0)

	summary := summaryOf(t, h)

	if summary.Status != sub_dto.CampaignStatusRunning {
		t.Errorf("expected %q for an open campaign with budget to spare, got %q",
			sub_dto.CampaignStatusRunning, summary.Status)
	}
	if summary.ClosedAt != nil {
		t.Errorf("FR-031: a campaign nobody closed must report closedAt = null, got %s", summary.ClosedAt)
	}
}

// seededClosed builds the look-up harness over a campaign that has been force-closed,
// with the given number of grants in each budget part.
//
// `closedCampaign` builds the row with its window still OPEN, so a test using it is
// testing the closed column and nothing else — see that helper. The grants are seeded
// exactly as `seededCampaign` does, including the watermark on the grant that filled
// the main budget, so the five-state machine would reach a real state rather than the
// anomaly one.
func seededClosed(
	t *testing.T,
	campaign *entities.Campaign,
	main, reserveGrants int,
) (*queryHarness, *entities.Campaign) {
	claims := newFakeClaimRepo()
	for i := 0; i < main; i++ {
		claims.seed(
			queryCode,
			uuid.New(),
			baseInstant().Add(-time.Duration(main-i)*time.Minute),
			entities.GrantBucketMain,
			i == main-1 && main == campaign.Quota,
		)
	}
	for i := 0; i < reserveGrants; i++ {
		claims.seed(
			queryCode,
			uuid.New(),
			baseInstant().Add(-time.Duration(reserveGrants-i)*time.Second),
			entities.GrantBucketReserve,
			false,
		)
	}
	h := newTestQueryUC(
		t,
		[]*entities.Campaign{campaign},
		claims,
		newFakeSweepRunRepo(),
		newFakeUserBriefContract(),
	)
	return h, campaign
}

// summaryOf reads the single configured campaign out of the harness.
//
// One assertion that the list has exactly one row, because every later assertion
// in this file reads `summary` and would silently check the zero value of a nil
// pointer instead of failing with a useful message.
func summaryOf(t *testing.T, h *queryHarness) *sub_dto.CampaignSummaryRes {
	t.Helper()
	items, total, err := h.uc.ListCampaigns(t.Context(), 1, 20)
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected exactly one configured campaign, got total=%d items=%d", total, len(items))
	}
	return items[0]
}

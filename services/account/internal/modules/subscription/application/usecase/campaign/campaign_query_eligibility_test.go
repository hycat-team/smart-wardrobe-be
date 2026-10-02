package campaign

import (
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedconfig "smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// 025 T074 / T080: the single-account look-up (FR-068, FR-033, FR-079, SC-011).
//
// THE ASSERTION IS ABOUT THE RESPONSES DIFFERING FROM EACH OTHER. FR-068 does not
// ask for correct values; it asks that the states be DISTINGUISHABLE, because they
// are different things to tell a customer:
//
//	"you were given the plan"
//	"you qualified; the grant is still on its way"
//	"you qualified, but the giveaways ran out"
//	"the campaign does not apply to you"
//
// So the test first asserts that the verdicts are pairwise different, and only
// then that each is the expected one. Checking the values one at a time would
// pass an implementation that answered `not_eligible` for all of them — which is
// exactly the shape of the original defect this phase exists to fix, and which
// satisfies every field-level check.

// eligibilityScenario is one account placed in one of the states.
type eligibilityScenario struct {
	name  string
	want  string
	build func(t *testing.T, h *queryHarness) uuid.UUID
}

// TestAccountEligibilityHasExactlySixValues is T080 / contract §3.4.
//
// The contract lists SIX `eligibility` values and a client branches on them, so
// the set is DOCUMENTED rather than emergent: a seventh value shipped by accident
// leaves every client with an unhandled case, and an implementation quietly
// dropping `eligible_pending` back into `eligible_but_exhausted` looks healthy
// while telling a customer the giveaways ran out.
//
// Asserting the COUNT rather than the membership is deliberate. Membership is
// already covered by the scenario test below, which reaches each value through a
// real look-up; what membership cannot catch is a seventh value added later that
// no scenario exercises.
func TestAccountEligibilityHasExactlySixValues(t *testing.T) {
	// The documented vocabulary, one entry per row of the §3.4 table.
	declared := []string{
		sub_dto.CampaignEligibilityGranted,
		sub_dto.CampaignEligibilityEligiblePending,
		sub_dto.CampaignEligibilityEligibleButExhausted,
		sub_dto.CampaignEligibilityNotEligible,
		sub_dto.CampaignEligibilityAlreadyClaimedOtherCampaign,
		sub_dto.CampaignEligibilityCreatedByAdmin,
	}
	if len(declared) != 6 {
		t.Fatalf("contracts/campaign-query-api.md §3.4 defines exactly 6 eligibility values; got %d",
			len(declared))
	}
	seen := map[string]bool{}
	for _, value := range declared {
		if seen[value] {
			t.Errorf("eligibility value %q is declared twice; a client branching on it cannot tell the two apart", value)
		}
		seen[value] = true
	}
}

// TestAccountEligibilityReturnsDifferentAnswers is T074 / FR-068 / SC-011.
func TestAccountEligibilityReturnsDifferentAnswers(t *testing.T) {
	// The campaign under test: quota 2, reserve 1, opened an hour ago and still
	// open. Quota 2 so the FULL main budget is reachable with two seeded grants,
	// which is what puts the fourth account into "qualified but out of stock".
	const quota = 2
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig(queryCode, "premium-monthly", quota, 1, &startsAt, nil)
	campaigns := []sharedconfig.CampaignConfig{campaign}

	scenarios := []eligibilityScenario{
		{
			// GRANTED. The account holds a claim from this campaign, which is the
			// one answer that needs nothing else to be right.
			name: "granted",
			want: sub_dto.CampaignEligibilityGranted,
			build: func(t *testing.T, h *queryHarness) uuid.UUID {
				account := queryAccount(baseInstant().Add(-30*time.Minute), false)
				h.accounts.accounts[account.ID] = account
				h.claims.seed(queryCode, account.ID, account.CreatedAt, entities.GrantBucketMain, true)
				return account.ID
			},
		},
		{
			// ELIGIBLE PENDING. The account satisfies every condition, the budget is
			// untouched, and there is NO claim row for it — the grant has not
			// happened yet, the message is still in flight or the catch-up sweep has
			// not reached it (FR-079).
			//
			// This scenario exists because it used to answer
			// `eligible_but_exhausted`, and that was wrong in the way that matters:
			// an operator reading `exhausted` tells a customer "we have run out of
			// giveaways" when the campaign still has quota and their grant is on its
			// way. It is the most common question on the phone, so it gets its own
			// value and an EMPTY reason — there is no fault to name.
			name: "eligible pending",
			want: sub_dto.CampaignEligibilityEligiblePending,
			build: func(t *testing.T, h *queryHarness) uuid.UUID {
				account := queryAccount(baseInstant().Add(-25*time.Minute), false)
				h.accounts.accounts[account.ID] = account
				return account.ID
			},
		},
		{
			// ELIGIBLE BUT EXHAUSTED. The account satisfies every condition, and the
			// budget is full: two main grants and one reserve grant. This is the
			// middle state FR-033 requires to be separate from `not_eligible`, and
			// the one an operator who cannot see it will get wrong — by telling a
			// qualifying customer they were not eligible. Unlike the scenario above
			// it carries a reason, because here there IS a supply shortage to name.
			name: "eligible but exhausted",
			want: sub_dto.CampaignEligibilityEligibleButExhausted,
			build: func(t *testing.T, h *queryHarness) uuid.UUID {
				account := queryAccount(baseInstant().Add(-40*time.Minute), false)
				h.accounts.accounts[account.ID] = account
				for i, bucket := range []entities.GrantBucket{
					entities.GrantBucketMain, entities.GrantBucketMain, entities.GrantBucketReserve,
				} {
					h.claims.seed(
						queryCode,
						uuid.New(),
						baseInstant().Add(-time.Duration(20-i)*time.Minute),
						bucket,
						bucket == entities.GrantBucketMain && i == 1,
					)
				}
				return account.ID
			},
		},
		{
			// NOT ELIGIBLE. The account registered BEFORE this campaign opened, so
			// condition 7 puts it outside the window (FR-017).
			//
			// It is deliberately NOT an administrator account: that one answers
			// `created_by_admin`, which is one of the six verdicts but not the one
			// this case is about. Reaching for the nearest way to make an account
			// ineligible would have tested the wrong branch — and the failure would
			// have shown up as "two scenarios answered the same thing", which is
			// exactly the mistake the pairwise assertion below exists to catch.
			name: "not eligible",
			want: sub_dto.CampaignEligibilityNotEligible,
			build: func(t *testing.T, h *queryHarness) uuid.UUID {
				account := queryAccount(baseInstant().Add(-2*time.Hour), false)
				h.accounts.accounts[account.ID] = account
				return account.ID
			},
		},
		{
			// CREATED BY ADMIN. The campaign simply does not apply to the account
			// (FR-091).
			name: "created by admin",
			want: sub_dto.CampaignEligibilityCreatedByAdmin,
			build: func(t *testing.T, h *queryHarness) uuid.UUID {
				account := queryAccount(baseInstant().Add(-20*time.Minute), true)
				h.accounts.accounts[account.ID] = account
				return account.ID
			},
		},
	}

	answers := make(map[string]string, len(scenarios))
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			claims := newFakeClaimRepo()
			h := newTestQueryUC(t, campaigns, claims, newFakeSweepRunRepo(), newFakeUserBriefContract())
			// The plan must resolve, or every account would answer
			// `plan_unavailable` and the verdicts would collapse into one — which is
			// itself the failure this test exists to catch.
			h.plans.put(testPlan("premium-monthly", 30))

			userID := scenario.build(t, h)
			got := eligibilityOf(t, h, userID)

			if got.Eligibility != scenario.want {
				t.Fatalf("expected %q, got %q (reason %q)", scenario.want, got.Eligibility, got.Reason)
			}
			if (got.Eligibility == sub_dto.CampaignEligibilityGranted) != got.Granted {
				t.Errorf("eligibility %q and granted=%v disagree", got.Eligibility, got.Granted)
			}
			answers[scenario.name] = got.Eligibility
		})
	}

	// THE CENTRAL ASSERTION. Every scenario answers differently — pairwise distinct.
	seen := map[string]string{}
	for name, answer := range answers {
		if previous, clash := seen[answer]; clash {
			t.Fatalf("FR-068: the accounts %q and %q both answered %q; the states must be "+
				"DISTINGUISHABLE, and returning the same answer for two of them is the defect this "+
				"phase exists to fix", previous, name, answer)
		}
		seen[answer] = name
	}
	if len(seen) != len(scenarios) {
		t.Fatalf("expected %d distinct answers, got %d", len(scenarios), len(seen))
	}
}

// TestAccountEligibilityReportsEligiblePendingWithNoReason is T080 / FR-079 /
// contract §3.4.
//
// The account passes EVERY condition, the campaign still has quota, and no claim
// row exists yet: the grant is in flight, or the catch-up sweep has not reached
// it. That is `eligible_pending`, `granted = false`, and an EMPTY reason.
//
// All three halves are asserted because each one is load-bearing. The value is
// what stops an operator telling the customer "we ran out of giveaways" — false
// in both senses, the campaign has quota and their grant is on its way. The
// `granted = false` is what stops a client counting them as served. And the empty
// reason is deliberate: every other verdict names a condition to act on, and
// there is no code that says "in flight" — inventing one would point the operator
// at a budget that is not the problem.
//
// THE ACCOUNT IS DELIBERATELY UNCLAIMED AND THE BUDGET DELIBERATELY UNTOUCHED.
// Both are load-bearing: with the budget full this account answers
// `eligible_but_exhausted`, and with a claim row it answers `granted`, so the
// fixture is the only thing separating this case from the two verdicts on either
// side of it.
func TestAccountEligibilityReportsEligiblePendingWithNoReason(t *testing.T) {
	// Quota 10 with nothing granted: room to spare, and a campaign still open.
	startsAt := baseInstant().Add(-time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig(queryCode, "premium-monthly", 10, 2, &startsAt, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestQueryUC(t, campaigns, claims, newFakeSweepRunRepo(), newFakeUserBriefContract())
	h.plans.put(testPlan("premium-monthly", 30))

	// Registered inside the window, not by an administrator, no subscription, and
	// no claim row — so conditions 1 to 8 all pass.
	account := queryAccount(baseInstant().Add(-30*time.Minute), false)
	h.accounts.accounts[account.ID] = account

	got := eligibilityOf(t, h, account.ID)

	if got.Eligibility != sub_dto.CampaignEligibilityEligiblePending {
		t.Fatalf("FR-079: an eligible account with quota left and no claim yet must answer %q, got %q (reason %q)",
			sub_dto.CampaignEligibilityEligiblePending, got.Eligibility, got.Reason)
	}
	if got.Eligibility == sub_dto.CampaignEligibilityEligibleButExhausted {
		t.Errorf("contract §3.4: %q must not stand in for %q — there is no shortage to report here",
			sub_dto.CampaignEligibilityEligibleButExhausted, sub_dto.CampaignEligibilityEligiblePending)
	}
	if got.Granted {
		t.Errorf("granted must be false: the claim has not been written yet")
	}
	if got.Reason != "" {
		t.Errorf("expected an empty reason — nothing has gone wrong yet — got %q", got.Reason)
	}
}

// TestAccountEligibilityReportsGrantedWithNoReason is T080 / contract §3.4.
//
// `granted` is the one verdict with an EMPTY reason, and that emptiness is
// meaningful: every other verdict names a condition to act on, and there is
// nothing to diagnose about a grant that happened. A reason here would be
// invented rather than derived.
func TestAccountEligibilityReportsGrantedWithNoReason(t *testing.T) {
	claims := newFakeClaimRepo()
	h := newTestQueryUC(t, healthCampaigns(), claims, newFakeSweepRunRepo(), newFakeUserBriefContract())
	h.plans.put(testPlan("premium-monthly", 30))

	account := queryAccount(baseInstant().Add(-30*time.Minute), false)
	h.accounts.accounts[account.ID] = account
	claims.seed(queryCode, account.ID, account.CreatedAt, entities.GrantBucketMain, true)

	got := eligibilityOf(t, h, account.ID)

	if !got.Granted {
		t.Fatalf("expected granted = true, got false")
	}
	if got.Reason != "" {
		t.Errorf("a granted account must carry no reason to act on, got %q", got.Reason)
	}
	// The envelope an operator needs to check the answer themselves (FR-030).
	if got.WatermarkAt == nil {
		t.Errorf("the response must carry the watermark so an operator can see the boundary the account was measured against")
	}
	if got.UserRegisteredAt == nil || !got.UserRegisteredAt.Equal(account.CreatedAt) {
		t.Errorf("the response must carry the account's creation instant; it is what the priority rule compares")
	}
	if got.CampaignCode != queryCode || got.UserID != account.ID {
		t.Errorf("the response must echo the campaign and account asked about, got %q / %s",
			got.CampaignCode, got.UserID)
	}
}

// TestAccountEligibilityUsesOnlyTheTwelveReasonCodes is T080 / contract §3.5.
//
// The `reason` vocabulary is EXACTLY the twelve `SkipReason` constants, and the
// four codes the contract deliberately excludes (`temporary_error`,
// `duplicate_claim`, `quota_exhausted`, `not_eligible`) must never appear.
//
// This is a DOCUMENTED SET, not a convention: a client switches on these strings,
// so a code that exists in the code but not in the contract leaves an operator
// with no row to look up and no way to know what to do. The exclusion check is
// here because the excluded four are precisely the ones an implementation would
// reach for: `not_eligible` looks like a reason and is not one, and
// `temporary_error` is a transport concern that belongs to
// `sharedevent.RetryableError` rather than here.
func TestAccountEligibilityUsesOnlyTheTwelveReasonCodes(t *testing.T) {
	allowed := map[string]bool{
		string(SkipCampaignNotConfigured): true,
		string(SkipCampaignNotStarted):    true,
		string(SkipCampaignSleeping):      true,
		string(SkipCreatedByAdmin):        true,
		string(SkipAlreadyClaimed):        true,
		string(SkipAlreadyOnPlan):         true,
		string(SkipPlanNotSettled):        true,
		string(SkipOutsideWindow):         true,
		string(SkipQuotaMain):             true,
		string(SkipQuotaCompensation):     true,
		string(SkipQuotaAnomaly):          true,
		string(SkipPlanUnavailable):       true,
	}
	if len(allowed) != 12 {
		t.Fatalf("contracts/campaign-query-api.md §3.5 defines exactly 12 reason codes; the list here has %d",
			len(allowed))
	}

	// The four the contract excludes BY NAME, with the reason each is excluded.
	forbidden := map[string]string{
		"temporary_error": "FR-062's retryable/permanent split is already implemented through sharedevent; a reason is never written because a retried message's transaction rolled back",
		"duplicate_claim": "condition 4 already blocks it with already_claimed; a UNIQUE violation is an anomaly to investigate, not a customer-facing reason",
		"quota_exhausted": "the reserve is a CEILING, so 'reserve empty' IS the hard cap — quota_compensation already says it",
		"not_eligible":    "that is an `eligibility` value, not a `reason`; a catch-all reason would defeat the point of the vocabulary",
	}

	// Walk a spread of situations and check every reason the look-up produced.
	reasons := reasonsFromEverySituation(t)
	if len(reasons) == 0 {
		t.Fatal("no situation produced a reason; the sweep of states is not reaching the code")
	}
	for _, reason := range reasons {
		if reason == "" {
			continue
		}
		if !allowed[reason] {
			t.Errorf("reason %q is not one of the twelve codes in contracts/campaign-query-api.md §3.5", reason)
		}
		for code, why := range forbidden {
			if reason == code {
				t.Errorf("reason %q is one of the four codes the contract deliberately EXCLUDES: %s", code, why)
			}
		}
	}
}

// TestAccountEligibilityReportsAlreadyClaimedOtherCampaign is T080 / FR-012.
//
// The account holds a claim written by an EARLIER campaign. Its verdict is its own
// value rather than a flavour of `not_eligible`, because it is a good customer who
// arrived too late — and because a lookup scoped to the campaign asked about would
// find nothing and report the account as eligible, which is the specific wrong
// answer this case exists to prevent.
func TestAccountEligibilityReportsAlreadyClaimedOtherCampaign(t *testing.T) {
	const earlierCode = "autumn-2026-09"
	claims := newFakeClaimRepo()
	h := newTestQueryUC(t, healthCampaigns(), claims, newFakeSweepRunRepo(), newFakeUserBriefContract())
	h.plans.put(testPlan("premium-monthly", 30))

	account := queryAccount(baseInstant().Add(-2*time.Hour), false)
	h.accounts.accounts[account.ID] = account
	claims.seed(earlierCode, account.ID, account.CreatedAt, entities.GrantBucketMain, true)

	got := eligibilityOf(t, h, account.ID)

	if got.Eligibility != sub_dto.CampaignEligibilityAlreadyClaimedOtherCampaign {
		t.Fatalf("FR-012: an account already granted by an earlier campaign must answer %q, got %q (reason %q)",
			sub_dto.CampaignEligibilityAlreadyClaimedOtherCampaign, got.Eligibility, got.Reason)
	}
	if got.Reason != string(SkipAlreadyClaimed) {
		t.Errorf("expected reason %q, got %q", SkipAlreadyClaimed, got.Reason)
	}
	if got.Granted {
		t.Errorf("granted must be false: the campaign asked about did not grant this account")
	}
}

// TestClaimsFilterNarrowsOnTheGrantInstant is T076c / FR-067.
//
// The filter axis is `granted_at`, and the bounds are half-open. Both matter and
// the second is easy to get wrong: with an inclusive `to`, an operator exporting
// "everything between 09:00 and 10:00" gets the 10:00 grants twice — once in this
// export and once in the next one — and never notices, because each export looks
// complete.
func TestClaimsFilterNarrowsOnTheGrantInstant(t *testing.T) {
	const quota = 10
	startsAt := baseInstant().Add(-24 * time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig(queryCode, "premium-monthly", quota, 0, &startsAt, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestQueryUC(t, campaigns, claims, newFakeSweepRunRepo(), newFakeUserBriefContract())

	// Three grants, an hour apart, so every bound below has a grant strictly
	// inside it and one strictly outside.
	base := baseInstant().Add(-3 * time.Hour)
	inWindow := []time.Time{
		base,
		base.Add(time.Hour),
		base.Add(2 * time.Hour),
	}
	for _, grantedAt := range inWindow {
		account := queryAccount(grantedAt.Add(-time.Minute), false)
		h.accounts.accounts[account.ID] = account
		claims.claims[account.ID] = &entities.CampaignClaim{
			ID:           uuid.New(),
			UserID:       account.ID,
			CampaignCode: queryCode,
			PlanSlug:     "premium-monthly",
			PlanID:       uuid.New(),
			RegisteredAt: account.CreatedAt,
			GrantedAt:    grantedAt,
			ExpiresAt:    grantedAt.AddDate(0, 0, 30),
			GrantBucket:  entities.GrantBucketMain,
		}
	}

	from := inWindow[0].Add(30 * time.Minute)
	to := inWindow[1].Add(30 * time.Minute)

	items, total, err := h.uc.ListClaims(t.Context(), queryCode, repositories.CampaignClaimsFilter{
		From: &from,
		To:   &to,
	}, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("FR-067: expected exactly 1 grant inside [from, to), got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 row, got %d", len(items))
	}
	if !items[0].GrantedAt.Equal(inWindow[1]) {
		t.Errorf("expected the middle grant, got %s want %s", items[0].GrantedAt.UTC(), inWindow[1].UTC())
	}
	// The join ran: the username is here, which means it came through the account
	// contract rather than being left empty.
	if items[0].Username == "" {
		t.Errorf("the claims list must join the account's username through the identity contract")
	}
	// FR-034: the term is anchored on the GRANT, not on the registration. The
	// account registered a minute before it was granted, so the two differ by
	// exactly that minute and the response must show the grant-based one.
	if items[0].ExpiresAt.Equal(accountRegisteredAt(items[0], inWindow[1])) {
		t.Errorf("FR-034: expiresAt must be grantedAt + durationDays, not registeredAt + durationDays")
	}
}

// TestClaimsFilterRefusesAReversedRange is T076c / FR-067.
//
// `from > to` is a BUG in the caller, and answering it with an empty page would
// hide that bug until somebody noticed a grant that "never happened". So it is
// refused.
func TestClaimsFilterRefusesAReversedRange(t *testing.T) {
	h := newTestQueryUC(t, healthCampaigns(), newFakeClaimRepo(), newFakeSweepRunRepo(), newFakeUserBriefContract())

	from := baseInstant()
	to := baseInstant().Add(-time.Hour)

	items, total, err := h.uc.ListClaims(t.Context(), queryCode, repositories.CampaignClaimsFilter{
		From: &from,
		To:   &to,
	}, 1, 20)

	if err == nil {
		t.Fatalf("FR-067: a reversed from/to must be refused; got %d rows and total %d", len(items), total)
	}
	if items != nil {
		t.Errorf("a refused range must return no rows, got %d", len(items))
	}
}

// TestUnknownCampaignIsRefusedNotEmpty is T076 / FR-072.
//
// An unknown code is an ERROR naming the code. An empty page would read as "this
// campaign granted nobody" — a completely different fact, and the one that would
// send an operator hunting for a budget fault that does not exist.
func TestUnknownCampaignIsRefusedNotEmpty(t *testing.T) {
	h := newTestQueryUC(t, healthCampaigns(), newFakeClaimRepo(), newFakeSweepRunRepo(), newFakeUserBriefContract())

	items, total, err := h.uc.ListClaims(t.Context(), "khong-ton-tai", repositories.CampaignClaimsFilter{}, 1, 20)

	if err == nil {
		t.Fatalf("FR-072: an unknown campaign code must be an error, not an empty page; got %d rows", len(items))
	}
	if items != nil || total != 0 {
		t.Errorf("a refused lookup must return nothing alongside the error, got %d rows / total %d",
			len(items), total)
	}
	// And the same for the single-campaign read.
	if _, err := h.uc.GetCampaign(t.Context(), "khong-ton-tai"); err == nil {
		t.Errorf("FR-072: GetCampaign must refuse an unknown code rather than return an empty summary")
	}
}

// TestUnknownAccountIsRefused is T076b / contract §4.
//
// A `:userId` that does not exist is a 404, and it is a DIFFERENT error from an
// unknown campaign: the two are typed by two different people, and merging them
// leaves an operator not knowing which of the two they mistyped.
func TestUnknownAccountIsRefused(t *testing.T) {
	h := newTestQueryUC(t, healthCampaigns(), newFakeClaimRepo(), newFakeSweepRunRepo(), newFakeUserBriefContract())
	h.plans.put(testPlan("premium-monthly", 30))

	if _, err := h.uc.GetAccountEligibility(t.Context(), queryCode, uuid.New()); err == nil {
		t.Fatalf("contract §4: an account that does not exist must be refused, not answered as ineligible")
	}
}

// ---- helpers ----

// eligibilityOf runs the single-account look-up and fails the test on error.
func eligibilityOf(t *testing.T, h *queryHarness, userID uuid.UUID) *sub_dto.CampaignAccountEligibilityRes {
	t.Helper()
	got, err := h.uc.GetAccountEligibility(t.Context(), queryCode, userID)
	if err != nil {
		t.Fatalf("look up eligibility for %s: %v", userID, err)
	}
	return got
}

// reasonsFromEverySituation drives a spread of look-ups and collects every reason
// code they produced.
//
// It is a SWEEP rather than one scenario because a wrong code tends to appear only
// on a branch: the full campaign, the campaign that has not opened, the closed
// campaign, the missing plan. Checking one branch would miss the other three.
func reasonsFromEverySituation(t *testing.T) []string {
	t.Helper()

	futureStart := baseInstant().Add(time.Hour)
	pastStart := baseInstant().Add(-time.Hour)
	futureEnd := baseInstant().Add(time.Hour)
	pastEnd := baseInstant().Add(-time.Hour)

	campaignSets := [][]sharedconfig.CampaignConfig{
		// An open campaign with room: the account qualifies and the budget allows.
		{testCampaignConfig(queryCode, "premium-monthly", 10, 2, &pastStart, &futureEnd)},
		// The same campaign with its budget spent.
		{testCampaignConfig(queryCode, "premium-monthly", 10, 2, &pastStart, &futureEnd)},
		// A campaign that has not opened.
		{testCampaignConfig(queryCode, "premium-monthly", 10, 2, &futureStart, &futureEnd)},
		// A campaign that has closed.
		{testCampaignConfig(queryCode, "premium-monthly", 10, 2, &pastStart, &pastEnd)},
		// A campaign whose plan_slug names nothing (FR-010).
		{testCampaignConfig(queryCode, "khong-ton-tai-goi", 10, 2, &pastStart, &futureEnd)},
		// No campaign at all.
		{},
	}

	var out []string
	for i, campaigns := range campaignSets {
		claims := newFakeClaimRepo()
		h := newTestQueryUC(t, campaigns, claims, newFakeSweepRunRepo(), newFakeUserBriefContract())
		h.plans.put(testPlan("premium-monthly", 30))

		if i == 1 {
			// Spend the budget: 10 main + 2 reserve, so condition 8 refuses.
			for j := 0; j < 12; j++ {
				bucket := entities.GrantBucketMain
				if j >= 10 {
					bucket = entities.GrantBucketReserve
				}
				claims.seed(
					queryCode, uuid.New(),
					baseInstant().Add(-time.Duration(30-j)*time.Minute),
					bucket, j == 9,
				)
			}
		}

		for j, createdByAdmin := range []bool{false, true} {
			// The account has to sit inside the campaign window for the window
			// rules to be the thing under test; outside it the reason is
			// `outside_window`, which is also worth collecting.
			createdAt := baseInstant().Add(-time.Hour)
			if j == 1 {
				createdAt = baseInstant().Add(-48 * time.Hour)
			}
			account := queryAccount(createdAt, createdByAdmin)
			h.accounts.accounts[account.ID] = account

			got, err := h.uc.GetAccountEligibility(t.Context(), queryCode, account.ID)
			if err != nil {
				// "No campaign configured" is a legitimate answer for the last set,
				// and this sweep is about the CODES rather than the errors.
				continue
			}
			out = append(out, got.Reason)
		}
	}
	return out
}

// accountRegisteredAt returns the registration instant implied by a response's
// expiry, so a test can tell the two anchors apart.
//
// `expiresAt = registeredAt + 30d` would equal `grantedAt + 30d` only if the two
// instants were the same, and the fixture deliberately separates them by a minute —
// so this reconstructs what a registration-anchored term WOULD have been, and the
// assertion is that the response does not carry it.
func accountRegisteredAt(item *sub_dto.CampaignClaimRes, grantedAt time.Time) time.Time {
	return item.ExpiresAt.Add(-30 * 24 * time.Hour).Add(time.Minute)
}

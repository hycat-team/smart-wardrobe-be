package campaign

import (
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// 026 T078, quickstart V6: THE SWEEP FINDS A CAMPAIGN THAT DID NOT EXIST WHEN THE
// PROCESS STARTED.
//
// WHY THIS IS THE SCENARIO THAT MATTERS MOST FOR 026. Before 026 the campaign list came
// from YAML, read once at startup, so "a campaign that was not in the file when the
// process booted" could not exist as a case at all — the deployment that introduced it
// WAS the mechanism. V6 exists to show that moving the list into a TABLE made a campaign
// effective without a restart, which is SC-007/FR-070 and the whole operational promise
// of the feature.
//
// THE ARRANGEMENT IS THE POINT, and it is why this cannot be folded into
// `TestCampaignSweepGrantsAnAccountThatNeverReceivedAMessage` (T102). That one builds
// the sweep over a registry that already holds the campaign, so it cannot tell a sweep
// that READS the registry from one that captured a slice at construction time — both
// answer identically. Here the registry starts EMPTY, the use case is constructed, and
// only then does the campaign appear. A sweep holding a construction-time snapshot sees
// nothing and sweeps an empty registry quietly, which is exactly what the operator would
// experience: a campaign open in the database that grants nobody.
//
// TWO CLAIMS, and the second is the one worth reading:
//
//  1. the account is GRANTED — the claim lands in the ledger naming the new campaign and
//     the plan is applied. A sweep that merely COUNTED the campaign would satisfy a
//     report assertion and nothing else.
//  2. the sweep asked the registry AGAIN. The fake counts `ListOpen` calls, so this is
//     a fact about the number of reads rather than an inference from the outcome — which
//     is what makes it survive an edit that keeps the outcome and loses the read.
func TestCampaignSweepFindsACampaignOpenedAfterStartup(t *testing.T) {
	// The registry is EMPTY when the use case is built. That single fact is the
	// difference between this test and every other sweep test in the package.
	registeredAt := baseInstant().Add(-20 * time.Minute)
	customer := sweepAccount(registeredAt, false)

	h, _, sweep := newTestSweep(t, nil, []*identitydto.UserBriefContractDTO{customer})

	// The premise, asserted rather than assumed: with a claim already present, or with
	// the campaign present from the start, this test would prove nothing.
	if got := h.claims.total(); got != 0 {
		t.Fatalf("the ledger must start empty: %d claims present before the sweep", got)
	}
	if open := h.campaigns.openCalls; open != 0 {
		t.Fatalf("the registry must not have been read before the sweep; got %d reads", open)
	}

	// THE OPERATOR'S CALL. `put` is the fake's equivalent of
	// `POST /api/v1/admin/campaigns`: a row appearing in `subscription.campaigns` with
	// no restart, no redeploy, and no restart of this test either.
	const code = "launch-2026-10"
	startsAt := registeredAt.Add(-time.Hour)
	h.campaigns.put(testCampaign(code, "premium-monthly", 50, 5, &startsAt, nil))

	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CLAIM 2 — the read happened, at sweep time.
	//
	// Asserted BEFORE the outcome on purpose. If it came second it would be unreachable
	// in the failure that matters: a sweep that skipped the registry and still happened
	// to grant would pass the outcome check.
	if h.campaigns.openCalls == 0 {
		t.Errorf("the sweep never read the registry (FR-040, FR-070). quickstart V6's whole claim is that " +
			"a campaign opened after startup reaches the grant path; a sweep holding a construction-time " +
			"list would grant nothing and report an empty registry, which is the pre-026 behaviour")
	}

	// The report is the sweep's own claim about what it did.
	if report.Scanned != 1 {
		t.Errorf("expected the sweep to scan 1 account, got %d", report.Scanned)
	}
	if report.Forwarded != 1 {
		t.Errorf("expected 1 account forwarded to the shared path, got %d", report.Forwarded)
	}
	if report.Skipped != 0 || report.Failed != 0 {
		t.Errorf("expected 0 skipped and 0 failed, got %d and %d", report.Skipped, report.Failed)
	}

	// CLAIM 1 — the grant is real.
	claim := h.claims.claimFor(customer.ID)
	if claim == nil {
		t.Fatalf("the sweep granted nobody from a campaign that was open in the registry (FR-070, SC-007)")
	}
	if claim.CampaignCode != code {
		t.Errorf("the claim names campaign %q, want %q", claim.CampaignCode, code)
	}
	if claim.PlanSlug != "premium-monthly" {
		t.Errorf("the claim names plan %q, want premium-monthly", claim.PlanSlug)
	}

	// The plan was actually applied, which is what a customer notices.
	sub, err := h.subs.GetByUserID(t.Context(), customer.ID)
	if err != nil || sub == nil {
		t.Fatalf("read the granted subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-monthly" {
		t.Errorf("expected the customer to be on premium-monthly, got %q", sub.CurrentPlanCode)
	}
	if got := h.claims.countBucket(code, entities.GrantBucketMain); got != 1 {
		t.Errorf("main grants = %d, want exactly 1", got)
	}
}

// TestCampaignOpenedAfterStartupAlsoReachesTheMessagePath is the same claim on the
// OTHER door, and it is a separate test because the two doors are separate code.
//
// V6 says a newly opened campaign is effective; "effective" for a signup that arrives
// AFTER the campaign is open is the message path, and for one that arrived before it is
// the sweep. `TestCampaignSweepFindsACampaignOpenedAfterStartup` covers the second. This
// covers the first, and the arrangement that makes it meaningful is again the timing: the
// registry holds nothing when the use case is BUILT, the campaign is written, and only
// then is the message handled.
//
// It also pins FR-072's boundary in the direction that is easy to get wrong. A path that
// decided from a construction-time snapshot would answer `campaign_not_configured` — the
// quietest wrong answer available, because it looks exactly like "no promotion is
// running" and nobody escalates it.
func TestCampaignOpenedAfterStartupAlsoReachesTheMessagePath(t *testing.T) {
	registeredAt := baseInstant().Add(-20 * time.Minute)
	startsAt := registeredAt.Add(-time.Hour)

	// Built EMPTY, exactly as in the sweep test.
	h := newTestUseCase(t, nil, newFakeClaimRepo())
	userID := sweepAccount(registeredAt, false).ID
	h.campaigns.put(testCampaign("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil))

	payload := identitydto.UserCreatedEventPayload{UserID: userID, RegisteredAt: registeredAt}
	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("process the signup: %v", err)
	}

	claim := h.claims.claimFor(userID)
	if claim == nil {
		t.Fatalf("the message path granted nobody from a campaign that was open in the registry when the " +
			"message was handled (FR-040, FR-070, SC-007). A path reading a construction-time snapshot " +
			"would answer campaign_not_configured, which reads as 'no promotion is running'")
	}
	if claim.CampaignCode != "launch-2026-10" {
		t.Errorf("the claim names campaign %q, want launch-2026-10", claim.CampaignCode)
	}

	// A message that was granted AND logged as skipped is how an operator ends up
	// looking for a reason code that says nothing happened.
	if reason, skipped := h.logger.skipReasonFor(userID); skipped {
		t.Errorf("the account was granted but the message path logged it skipped with reason %q", reason)
	}
}

// TestASweepOverAnEmptyRegistryWritesNoHealthRow is the other half of V6, and the half
// that would otherwise be untested by accident.
//
// V6 says a new campaign is picked up IN A CYCLE. The dangerous neighbouring behaviour is
// a health write for a campaign the sweep did nothing about: `campaign_sweep_runs` would
// gain a row per campaign per cycle over an empty registry, and `degraded` would then
// have something to report for a campaign that was never scanned. T026/SC-008 covers the
// message path and the sweep's own silence; this covers the TABLE.
//
// IT BUILDS THE SWEEP BY RATHER THAN THROUGH `newTestSweep`, because that helper does not
// hand back the sweep-run fake and the assertion is about the fake's write count. Asserting
// on the REPORT instead would be the weaker test: a sweep that reports "0 campaigns" and
// writes a health row per campaign anyway satisfies every report assertion.
func TestASweepOverAnEmptyRegistryWritesNoHealthRow(t *testing.T) {
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, nil, claims)
	runs := newFakeSweepRunRepo()

	sweep := NewCampaignSweepUseCase(
		h.uc.cfg,
		newFakeAccountLister(),
		h.campaigns,
		claims,
		runs,
		h.uc,
		h.logger,
	).(*CampaignSweepUseCase)
	sweep.now = func() time.Time { return baseInstant() }

	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("an empty registry must not be an error: %v", err)
	}

	if report.Scanned != 0 || report.Forwarded != 0 || report.Skipped != 0 || report.Failed != 0 {
		t.Errorf("expected a completely empty report over an empty registry, got %+v", report)
	}
	if got := runs.writeCount(); got != 0 {
		t.Errorf("the sweep wrote %d health rows over an empty registry; an empty result set is not a "+
			"write. A row here would give `degraded` and `lastSweepAt` something to report for a campaign "+
			"that was never scanned, and an operator would read it as a fault (SC-008)", got)
	}

	// One plan change would be worse than a health row: it would apply a plan to an
	// account nobody qualified for.
	if got := len(h.events.published()); got != 0 {
		t.Errorf("expected no plan change over an empty registry, got %d", got)
	}
}

package campaign

import (
	"context"
	"sync"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// 025 T100-T102: the catch-up sweep (sweep_uc.go), proven through the ledger.
//
// WHY THIS FILE IS ABOUT ORDER, NOT ABOUT FINAL VALUES. The end-to-end run that
// preceded it found a class of defect no single-component test could see: the
// account service provisions the free plan in the signup transaction, and that
// plan-change event carries a real "now". The campaign grant used to be stamped
// with the account's REGISTRATION time, so its event arrived carrying an
// `occurredAt` OLDER than the record already on file. The AI service enforces a
// monotonic guard on `effective_from`, refused every campaign grant as
// `stale_event`, and dead-lettered the message. The account then sat on
// premium-monthly with the free plan's AI entitlements and no way back, because
// the grant had been written and the message was gone.
//
// Two consequences shape every test here:
//
//   - A grant is asserted on the LEDGER (claims, plans, history rows), never on
//     the return value. A path that answered "granted" while writing nothing, or
//     wrote a claim with no plan, satisfies any return-value assertion.
//   - Where two services meet, the assertion is a RELATION between their events
//     (`not before`), not a value in isolation.

// sweepRacers is the number of concurrent catch-up runs in T100.
const sweepRacers = 2

// runConcurrentSweeps starts `runs` sweeps at the same instant and waits for all
// of them.
//
// The barrier is a closed channel rather than a plain loop, because the
// alternative is the failure mode this feature has already produced once: a
// check that passes because the scheduler happened to serialise the racers, and
// then fails only in production where the timing differs. Every goroutine is
// inside `Sweep` before any of them is released.
func runConcurrentSweeps(t *testing.T, sweep uc_interfaces.ICampaignSweepUseCase, runs int) []uc_interfaces.CampaignSweepReport {
	t.Helper()
	start := make(chan struct{})
	reports := make([]uc_interfaces.CampaignSweepReport, runs)
	errs := make([]error, runs)

	var wg sync.WaitGroup
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			reports[idx], errs[idx] = sweep.Sweep(t.Context())
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("sweep %d: unexpected error: %v", i, err)
		}
	}
	return reports
}

// parkBothSweepsInTheWindowQuery makes two runs genuinely concurrent.
//
// The sweep asks for its first page of the window and then decides one account
// at a time. Left alone, two runs interleave somewhere in that loop and
// sometimes not at all — and "sometimes not" is exactly the condition under
// which a concurrency defect survives review and reaches production. Holding
// both inside `ListCreatedInWindow` until they have both arrived removes the
// luck: the two runs are provably reading the same page at the same moment.
func parkBothSweepsInTheWindowQuery(t *testing.T, lister *fakeAccountLister) {
	t.Helper()
	lister.entered = make(chan struct{}, sweepRacers)
	lister.release = make(chan struct{})
	go func() {
		for i := 0; i < sweepRacers; i++ {
			<-lister.entered
		}
		close(lister.release)
	}()
}

// assertWatermarkAndCeiling is the T100 invariant check, run after the race for
// the last main slot and again after the settled campaign is re-swept.
//
// FR-030 asks for exactly one flagged row whose `registered_at` never moves;
// FR-022 asks for the hard ceiling. The third assertion is the one a
// return-value test could never make: a RESERVE grant must be an account created
// BEFORE the watermark, never at or after it (FR-031). Under concurrency that is
// the pair most likely to invert, and an inverted reserve grant is a permanent
// silent wrong answer — nobody would see the customer receive the wrong thing,
// only that they received something.
func assertWatermarkAndCeiling(
	t *testing.T,
	claims *fakeClaimRepo,
	campaignCode string,
	quota, reserve int,
) *entities.CampaignClaim {
	t.Helper()

	if got := claims.watermarkCount(campaignCode); got != 1 {
		t.Fatalf("expected exactly 1 fills_main_quota row per campaign (FR-030), got %d", got)
	}
	watermark := claims.flaggedClaim(campaignCode)
	if watermark == nil {
		t.Fatalf("expected to read the watermark row back, got none")
	}
	if watermark.RegisteredAt.IsZero() {
		t.Fatalf("the watermark row carries no registered_at; FR-030 freezes the ACCOUNT's creation instant")
	}

	if got := claims.countBucket(campaignCode, entities.GrantBucketMain); got > quota {
		t.Fatalf("main grants = %d exceed quota %d (FR-022)", got, quota)
	}
	if got := claims.countBucket(campaignCode, entities.GrantBucketReserve); got > reserve {
		t.Fatalf("reserve grants = %d exceed reserve %d (FR-022)", got, reserve)
	}
	if got := claims.total(); got > quota+reserve {
		t.Fatalf("total claims = %d exceed the hard ceiling quota+reserve = %d (FR-022)", got, quota+reserve)
	}
	if got := reserveGrantsAtOrAfter(watermark, claims, campaignCode); got != 0 {
		t.Fatalf("found %d reserve grant(s) with registered_at >= the watermark %s (FR-031); "+
			"the priority rule was decided out of order", got, watermark.RegisteredAt.UTC())
	}
	return watermark
}

// reserveGrantsAtOrAfter counts reserve grants that are not older than the
// watermark. A test helper rather than a production one: the rule already
// refuses them, and what is asserted here is that it refused them while two runs
// were racing.
func reserveGrantsAtOrAfter(watermark *entities.CampaignClaim, claims *fakeClaimRepo, campaignCode string) int {
	claims.mu.Lock()
	defer claims.mu.Unlock()
	count := 0
	for _, claim := range claims.claims {
		if claim.CampaignCode != campaignCode || claim.GrantBucket != entities.GrantBucketReserve {
			continue
		}
		if !claim.RegisteredAt.Before(watermark.RegisteredAt) {
			count++
		}
	}
	return count
}

// TestCampaignSweepWatermarkIsFrozenWhenTwoSweepsRace is T100 / FR-030 / SC-001.
//
// The scenario is the worst one available: exactly ONE main slot is left, and two
// catch-up runs are walking the same window at the same time. Exactly one of them
// may take that slot, and the row that takes it becomes the priority watermark —
// the value every later decision is compared against, including which accounts
// may still take the reserve.
//
// The test runs in two phases on purpose. Phase one is where the race happens;
// phase two re-sweeps the settled campaign and asserts the watermark has not
// moved. One phase could only prove the flag was SET once. What FR-030 actually
// promises is that the frozen instant is DURABLE, and only a second pass over
// the same ledger can tell a frozen watermark from a lucky one.
//
// Two main grants are seeded so the main count is 2 against a quota of 3. That
// is what leaves the single contested slot, and it is also what makes the
// reserve branch reachable at all: the watermark is the third account processed,
// so only accounts older than it are reserve candidates.
func TestCampaignSweepWatermarkIsFrozenWhenTwoSweepsRace(t *testing.T) {
	const (
		quota   = 3
		reserve = 2
		code    = "launch-2026-10"
	)
	registeredBase := baseInstant().Add(-8 * time.Hour)
	startsAt := registeredBase.Add(-time.Hour)
	campaigns := []*entities.Campaign{
		testCampaign(code, "premium-monthly", quota, reserve, &startsAt, nil),
	}

	// Twelve accounts inside the campaign window, oldest first — more than the
	// ceiling of 5, so both budget parts are under real pressure, spaced far
	// enough apart that the priority ordering is unambiguous.
	accounts := make([]*identitydto.UserBriefContractDTO, 0, 12)
	for i := 0; i < 12; i++ {
		accounts = append(accounts, sweepAccount(registeredBase.Add(time.Duration(i)*time.Minute), false))
	}

	h, lister, sweep := newTestSweep(t, campaigns, accounts)
	claims := h.claims

	// Make the race real rather than incidental. `rotatePerCaller` gives the two
	// runs windows that start in different halves of the list, which is the drift
	// between two replicas that the advisory lock exists to make harmless, and
	// `countHold` guarantees two budget reads overlap instead of leaving that to
	// the scheduler. Without both, two runs walk the same accounts in the same
	// order, collide on `ux_campaign_claims_user`, and the test passes even with
	// the lock removed.
	lister.rotatePerCaller = true
	claims.countHold = 2 * time.Millisecond

	claims.seed(code, accounts[10].ID, accounts[10].CreatedAt, entities.GrantBucketMain, false)
	claims.seed(code, accounts[11].ID, accounts[11].CreatedAt, entities.GrantBucketMain, false)
	if got := claims.watermarkCount(code); got != 0 {
		t.Fatalf("the ledger must start with no watermark, got %d flagged rows", got)
	}
	if got := claims.countBucket(code, entities.GrantBucketMain); got != quota-1 {
		t.Fatalf("the seed must leave exactly one contested main slot, main count = %d", got)
	}

	parkBothSweepsInTheWindowQuery(t, lister)
	runConcurrentSweeps(t, sweep, sweepRacers)

	frozen := assertWatermarkAndCeiling(t, claims, code, quota, reserve)
	if got := claims.countBucket(code, entities.GrantBucketMain); got != quota {
		t.Fatalf("expected the main budget to be filled exactly once (quota %d), got %d grants", quota, got)
	}

	// Phase two: two more concurrent runs over the same window. Granted accounts
	// are skipped by the claim set and refused ones are refused again for the
	// same reason, so nothing may be written — above all the watermark may not
	// move.
	before := claims.total()
	parkBothSweepsInTheWindowQuery(t, lister)
	runConcurrentSweeps(t, sweep, sweepRacers)
	after := assertWatermarkAndCeiling(t, claims, code, quota, reserve)

	if after.UserID != frozen.UserID || !after.RegisteredAt.Equal(frozen.RegisteredAt) {
		t.Fatalf("the priority watermark moved between runs (FR-030): was %s at %s, now %s at %s",
			frozen.UserID, frozen.RegisteredAt.UTC(), after.UserID, after.RegisteredAt.UTC())
	}
	if got := claims.total(); got != before {
		t.Fatalf("a settled campaign must not gain claims on a later sweep: %d before, %d after", before, got)
	}
}

// TestCampaignSweepGrantsALateArrivalAndMarksItLate is T101 / FR-078 / FR-082.
//
// Two requirements, and the second is the one a "did it grant" assertion would
// miss entirely.
//
// FR-078 is a commitment to the customer: eligible accounts are granted within
// five minutes. The account here registered EIGHT hours ago and its message was
// never delivered, so the sweep is the only thing that can still grant it. The
// half of FR-078 that is easy to get wrong is believing lateness entitles the
// mechanism to give up — an account eight hours old is still eligible, and the
// term it was promised is a full month from the grant, not a month from a
// deadline that has already passed.
//
// FR-082 asks for that lateness to be an OBSERVABLE EVENT, distinguishable from a
// message-path grant. That is the assertion with teeth: the audit row's
// `grantSource` is one of only two places the distinction can live (FR-092
// forbids a new event type and FR-087 forbids widening the payload), so a grant
// leaving no such mark would be indistinguishable from one the queue settled on
// time — and "the system is slow" would look exactly like "the system is fine".
//
// The third assertion is the ORDER one. The signup transaction already
// provisioned the free plan and published that plan change with a real clock
// reading, and it is already on the wire. The sweep's grant event must not be
// stamped earlier than it, or the AI service's monotonic guard rejects the grant
// as `stale_event` — after the claim and the plan are already written.
func TestCampaignSweepGrantsALateArrivalAndMarksItLate(t *testing.T) {
	const code = "launch-2026-10"
	registeredAt := baseInstant().Add(-8 * time.Hour)
	startsAt := registeredAt.Add(-time.Hour)
	campaigns := []*entities.Campaign{
		testCampaign(code, "premium-monthly", 50, 5, &startsAt, nil),
	}

	account := sweepAccount(registeredAt, false)
	h, _, sweep := newTestSweep(t, campaigns, []*identitydto.UserBriefContractDTO{account})

	// What the signup transaction does BEFORE any campaign message exists: the
	// free plan is provisioned and its plan change published with a real clock
	// reading. The sweep neither redoes nor contradicts it.
	provisionFreePlanAtSignup(t, h, account.ID)
	if got := len(h.publisher.payloads()); got != 1 {
		t.Fatalf("expected the signup transaction to have published 1 plan change, got %d", got)
	}

	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("a late but eligible account must not fail the sweep: %v", err)
	}

	// THE ORDERING ASSERTION, and it comes first because it is the one that has
	// already cost real accounts. The AI service refuses a plan change whose
	// effective_from is not newer than the one it already applied, so the sweep's
	// grant event must not precede the free plan's provisioning event. The two
	// services meet here and neither can see the other's clock, so this relation
	// is only checkable from the outside — which is why it needs a test and not a
	// code review.
	published := h.publisher.payloads()
	if len(published) != 2 {
		t.Fatalf("expected 2 published plan changes (free plan provisioning + sweep grant), got %d", len(published))
	}
	freeEventAt := occurredAtOf(t, published[0].Payload)
	grantEventAt := occurredAtOf(t, published[1].Payload)
	if grantEventAt.Before(freeEventAt) {
		t.Fatalf("the sweep's grant event is stamped %s, EARLIER than the free plan's provisioning event %s; "+
			"the AI service rejects it as stale_event and the account keeps the free plan's AI entitlements "+
			"with no way back", grantEventAt, freeEventAt)
	}
	if grantEventAt.Equal(registeredAt.UTC()) {
		t.Fatalf("the sweep's grant event is stamped with the account's registration time %s; "+
			"that is the stale_event defect", grantEventAt)
	}

	// FR-078: the grant happened, and the term is a FULL month from the grant
	// instant rather than a month from a deadline eight hours gone.
	sub, err := h.subs.GetByUserID(context.Background(), account.ID)
	if err != nil || sub == nil {
		t.Fatalf("read the swept account's subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-monthly" {
		t.Fatalf("expected the late account to be granted premium-monthly, got %q", sub.CurrentPlanCode)
	}
	wantExpiry := baseInstant().AddDate(0, 0, 30)
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expected a full 30-day term from the grant instant (%s), got %v — "+
			"lateness must not shorten what the account was promised", wantExpiry, sub.ExpiresAt)
	}
	if sub.IsAutoRenewEnabled {
		t.Fatalf("expected auto-renew OFF on a promotional grant (FR-035)")
	}
	if claim := h.claims.claimFor(account.ID); claim == nil {
		t.Fatalf("the late account holds no claim; a plan with no claim is granted twice (FR-039)")
	}

	// FR-082: the lateness is an observable event, not an internal counter.
	if report.LateAccounts != 1 {
		t.Fatalf("expected the sweep to record 1 late account (FR-082), got %d", report.LateAccounts)
	}
	wantDelay := int(baseInstant().Sub(registeredAt) / time.Second)
	if report.MaxDelaySeconds != wantDelay {
		t.Fatalf("expected the worst delay to be %ds, got %ds", wantDelay, report.MaxDelaySeconds)
	}
	// A completed run with a non-zero lateness count means SLOW, not BROKEN —
	// the two states FR-082 exists to tell apart.
	//
	// The degraded flag itself is no longer readable from the sweep (025 T079c):
	// it lives in `campaign_sweep_runs` because this service runs several
	// replicas, each with its own worker set, and an in-memory flag read by one
	// replica says nothing about a failure another produced. What this test can
	// still assert is that the RUN REPORTED NO ERROR, which is the local
	// equivalent — and the durable flag is covered by
	// `TestSweepRecordsDegradedInTheDatabaseWhenItFails`.
	if err != nil {
		t.Fatalf("a sweep that granted a late account must not fail: %v", err)
	}
	if report.LateAccounts != 1 {
		t.Fatalf("expected the sweep to record 1 late account (FR-082), got %d", report.LateAccounts)
	}
	if report.MaxDelaySeconds != wantDelay {
		t.Fatalf("expected the worst delay to be %ds, got %ds", wantDelay, report.MaxDelaySeconds)
	}

	// The trace names WHICH PATH settled this account. A message-path grant
	// records the other origin (asserted at the end of this test), so an operator
	// reading the audit trail can tell that the queue did not deliver this one.
	record, ok := auditFor(h, account.ID)
	if !ok {
		t.Fatalf("no audit record names the late account")
	}
	if got := record.Payload["grantSource"]; got != string(grantOriginSweep) {
		t.Fatalf("expected grantSource=%q on a sweep grant (FR-082), got %#v — "+
			"a late grant must be distinguishable from a message-path grant", grantOriginSweep, got)
	}

	// The two origins are genuinely distinguishable. A sweep that stamped
	// "campaign_sweep" on EVERY grant would satisfy the assertions above and
	// answer FR-082 with a value that distinguishes nothing, so the message path
	// is exercised here too and must record the other origin.
	viaMessage := sweepAccount(baseInstant(), false)
	viaMessage.ID = uuid.New()
	if err := h.uc.ProcessUserCreated(context.Background(), identitydto.UserCreatedEventPayload{
		UserID:       viaMessage.ID,
		RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("unexpected error on the message path: %v", err)
	}
	messageRecord, ok := auditFor(h, viaMessage.ID)
	if !ok {
		t.Fatalf("no audit record names the message-path account")
	}
	if got := messageRecord.Payload["grantSource"]; got != string(grantOriginMessage) {
		t.Fatalf("expected grantSource=%q on a message-path grant, got %#v", grantOriginMessage, got)
	}
	if string(grantOriginMessage) == string(grantOriginSweep) {
		t.Fatalf("the two grant origins are the same value, so no grant is distinguishable (FR-082)")
	}
}

// TestCampaignSweepGrantsAnAccountThatNeverReceivedAMessage is T102 / FR-079 /
// SC-016.
//
// The catch-up sweep exists for exactly one account: the one that signed up,
// qualified, and whose message was lost. Nothing in the message path ever runs
// for it, so the sweep has to find it by walking the window and the grant has to
// be real rather than reported.
//
// The account here is NOT messaged: `ProcessUserCreated` is never called for it,
// which is the simulation of the lost message. Asserting the outcome on the LEDGER
// and not on `Sweep`'s return value is what makes this a test of the grant: a
// report saying "forwarded 1" with no claim and no plan would satisfy every
// return-value assertion in the specification.
//
// The ineligible account is the other half. A sweep that granted everything it
// found would also pass an assertion that checks only the eligible one, and the
// account it wrongly granted is an administrator's — the one FR-091 singles out.
func TestCampaignSweepGrantsAnAccountThatNeverReceivedAMessage(t *testing.T) {
	const code = "launch-2026-10"
	registeredAt := baseInstant().Add(-20 * time.Minute)
	startsAt := registeredAt.Add(-time.Hour)
	campaigns := []*entities.Campaign{
		testCampaign(code, "premium-monthly", 50, 5, &startsAt, nil),
	}

	// Eligible, inside the window, and no message was ever delivered: the sweep is
	// the ONLY thing that can settle it.
	orphaned := sweepAccount(registeredAt, false)
	// FR-091: created by an administrator, so the campaign does not apply.
	adminAccount := sweepAccount(registeredAt.Add(time.Minute), true)

	h, _, sweep := newTestSweep(t, campaigns, []*identitydto.UserBriefContractDTO{orphaned, adminAccount})

	// Nothing has touched the shared path. Asserted rather than assumed, because
	// it is the premise of the whole test: with a claim already present the sweep
	// would skip the account and this would prove nothing.
	if got := h.claims.total(); got != 0 {
		t.Fatalf("the ledger must start empty: %d claims present before the sweep", got)
	}
	if got := len(h.events.published()); got != 0 {
		t.Fatalf("no plan change may exist before the sweep, got %d", got)
	}

	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The claim ledger. One row for the orphaned account, naming the campaign and
	// the plan — the row that stops a redelivered message from granting the same
	// account twice (FR-025).
	claim := h.claims.claimFor(orphaned.ID)
	if claim == nil {
		t.Fatalf("the sweep granted no claim for the account that never received a message (FR-079)")
	}
	if claim.CampaignCode != code {
		t.Fatalf("expected the claim to name campaign %q, got %q", code, claim.CampaignCode)
	}
	if claim.PlanSlug != "premium-monthly" {
		t.Fatalf("expected the claim to name the campaign plan, got %q", claim.PlanSlug)
	}
	if !claim.RegisteredAt.Equal(orphaned.CreatedAt) {
		t.Fatalf("the claim must record the account's CREATION instant, got %s want %s",
			claim.RegisteredAt.UTC(), orphaned.CreatedAt.UTC())
	}

	// FR-091: the administrator's account is dropped by the sweep, and the reason
	// code it logs is the SHARED one rather than a second spelling of the same
	// rule — which is what keeps the two paths from disagreeing (FR-080).
	//
	// The reason is deliberately NOT read back from the log here. The sweep logs
	// its own skips at Debug, because on a full campaign every account in the
	// window is skipped that way every fifteen minutes and an Info line each
	// would bury the grants; `harness.lastSkip` reads the Info and Warn levels,
	// and a test that reached for it would be asserting on a level the design
	// refuses to write. What is observable — and what FR-091 is actually about —
	// is that the account leaves with no claim, no plan and no entitlement.
	if got := h.claims.claimFor(adminAccount.ID); got != nil {
		t.Fatalf("an account created by an administrator must not be granted (FR-091), got claim %#v", got)
	}
	adminSub, err := h.subs.GetByUserID(context.Background(), adminAccount.ID)
	if err != nil {
		t.Fatalf("read the administrator's subscription: %v", err)
	}
	if adminSub != nil {
		t.Fatalf("the administrator's account must be left exactly as it was (FR-091), got plan %q", adminSub.CurrentPlanCode)
	}

	// The plan table. A claim with no plan is the silent half-state FR-039 names:
	// the account is recorded as granted while holding the free plan, and nothing
	// reports it.
	sub, err := h.subs.GetByUserID(context.Background(), orphaned.ID)
	if err != nil || sub == nil {
		t.Fatalf("read the swept account's subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-monthly" {
		t.Fatalf("expected the plan to be applied, got %q — a claim without a plan is FR-039", sub.CurrentPlanCode)
	}
	if sub.SubscriptionPlanID == uuid.Nil {
		t.Fatalf("expected the applied plan's id on the subscription row, got the nil UUID")
	}
	if sub.IsAutoRenewEnabled {
		t.Fatalf("expected auto-renew OFF on a promotional grant (FR-035)")
	}

	// The history row. One grant writes exactly one plan-change event; two would
	// let the AI service apply the entitlement twice.
	rows := h.events.published()
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 plan-change history row for one grant, got %d", len(rows))
	}
	if rows[0].Event.UserID != orphaned.ID {
		t.Fatalf("the history row belongs to %s, want the swept account %s", rows[0].Event.UserID, orphaned.ID)
	}
	if rows[0].Event.OccurredAt.Before(baseInstant().UTC()) {
		t.Fatalf("the history row is stamped %s, older than the processing clock %s (R13); "+
			"the AI service rejects it as stale_event",
			rows[0].Event.OccurredAt.UTC(), baseInstant().UTC())
	}

	// The report, last. The counters are the sweep's account of what it did, and
	// they are worth asserting only once the tables agree with them.
	if report.Scanned != 2 {
		t.Fatalf("expected the sweep to scan 2 accounts, got %d", report.Scanned)
	}
	if report.Forwarded != 1 {
		t.Fatalf("expected 1 account forwarded to the shared path, got %d", report.Forwarded)
	}
	if report.Skipped != 1 {
		t.Fatalf("expected 1 account skipped (the administrator's), got %d", report.Skipped)
	}
	if report.Failed != 0 {
		t.Fatalf("expected 0 failures, got %d", report.Failed)
	}

	// A second run is a no-op. The claim set makes it one, and that is the whole
	// reason a periodic sweep is safe to run forever: it must converge rather than
	// accumulate.
	second, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("unexpected error on the second sweep: %v", err)
	}
	if got := h.claims.total(); got != 1 {
		t.Fatalf("a repeat sweep must not grant again, got %d claims", got)
	}
	if got := len(h.events.published()); got != 1 {
		t.Fatalf("a repeat sweep must not publish a second plan change, got %d", got)
	}
	if second.Forwarded != 0 {
		t.Fatalf("expected the settled account to be skipped rather than forwarded, forwarded %d", second.Forwarded)
	}
}

// provisionFreePlanAtSignup reproduces what the signup transaction does before
// any campaign message is published: the free plan is provisioned on the account
// and its plan change goes out with a real clock reading.
//
// It lives here rather than in the fakes because exactly one assertion needs it —
// the ORDERING assertion in T101, which is about an event from the OTHER service
// already on the wire before the sweep touches anything.
func provisionFreePlanAtSignup(t *testing.T, h *testHarness, userID uuid.UUID) {
	t.Helper()
	h.plans.put(testFreePlan())
	freePlan, err := h.plans.GetBySlug(context.Background(), "free")
	if err != nil || freePlan == nil {
		t.Fatalf("the signup transaction needs an active free plan to provision: %v", err)
	}
	if err := h.subs.Create(context.Background(), &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: freePlan.ID,
		CurrentPlanCode:    freePlan.Slug,
		CurrentTierRank:    freePlan.TierRank,
		CurrentPlanKind:    freePlan.PlanKind,
		StartedAt:          baseInstant(),
	}); err != nil {
		t.Fatalf("provision the free plan: %v", err)
	}

	// The published event is what the AI service holds, and its clock reading is
	// the PROCESSING instant — the same one the campaign grant must not precede.
	publisher := subscription.NewPlanChangePublisher(h.events, h.publisher)
	slug := freePlan.Slug
	if _, err := publisher.PublishPlanChange(context.Background(), &entities.UserSubscriptionEvent{
		EventKey:    "SIGNUP_PROVISION:" + userID.String(),
		UserID:      userID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		ToPlanCode:  &slug,
		OccurredAt:  baseInstant(),
		EffectiveAt: baseInstant(),
	}, &entities.UserSubscription{UserID: userID}, freePlan); err != nil {
		t.Fatalf("publish the free plan's plan change: %v", err)
	}
}

// auditFor returns the audit record naming this account.
func auditFor(h *testHarness, userID uuid.UUID) (recordedAudit, bool) {
	for _, record := range h.audit.all() {
		if record.TargetID != nil && *record.TargetID == userID {
			return record, true
		}
		if record.Payload["targetUserId"] == userID.String() {
			return record, true
		}
	}
	return recordedAudit{}, false
}

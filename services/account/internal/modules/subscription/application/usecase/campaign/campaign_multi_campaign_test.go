package campaign

// 025 T084-T086: US5 — a second campaign, opened by configuration alone.
//
// A campaign is DATA. These tests exist to show that claim is load-bearing rather
// than aspirational: one binary, one rule set, one code path serving two
// campaigns at once (T084); a campaign that closes for accounts created after its
// end however late their message arrives (T085); and one participation per
// account for the whole lifetime of the mechanism (T086).
//
// NOTHING HERE ADDS PRODUCTION CODE, and that is the deliverable of Phase 8. The
// phase's output is the EVIDENCE that opening a campaign is a configuration edit
// (T089). A test that could only pass after a new code path was written would
// disprove the feature it is meant to prove.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedconfig "smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// TestASecondCampaignNeedsNoSourceChange is T084 / FR-011.
//
// The two campaigns differ in every field an operator controls — code, plan,
// budget, window — and the ONLY difference between "one campaign" and "two
// campaigns" is one more element in the slice the loader produced. No new
// branch, no new constant, no new table, nothing to deploy.
func TestASecondCampaignNeedsNoSourceChange(t *testing.T) {
	firstStarts := baseInstant().Add(-72 * time.Hour)
	secondStarts := baseInstant().Add(-1 * time.Hour)

	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &firstStarts, nil),
		testCampaignConfig("sample-2026-11", "premium-quarterly", 10, 2, &secondStarts, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	// Registered inside BOTH windows, so FR-011 alone decides where it goes.
	newcomer := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant().Add(-30 * time.Minute),
	}
	if !h.granted(t, newcomer) {
		t.Fatalf("expected the newcomer to be granted by one of the two campaigns")
	}

	claim := claims.claimFor(newcomer.UserID)
	if claim == nil {
		t.Fatalf("expected a claim row for the newcomer")
	}
	if claim.CampaignCode != "sample-2026-11" {
		t.Fatalf("expected the NEW campaign to take an account registered after it opened, got %q", claim.CampaignCode)
	}
	if claim.PlanSlug != "premium-quarterly" {
		t.Fatalf("expected the new campaign's OWN plan to be granted, got %q", claim.PlanSlug)
	}
	sub, err := h.subs.GetByUserID(t.Context(), newcomer.UserID)
	if err != nil || sub == nil {
		t.Fatalf("read granted subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-quarterly" {
		t.Fatalf("expected the subscription to be on premium-quarterly, got %q", sub.CurrentPlanCode)
	}

	// OPENING a campaign must not spend from the one already running. This is the
	// assertion an implementation that hard-coded "the launch campaign" could
	// not pass, however well it handled a single campaign.
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 0 {
		t.Fatalf("expected 0 claims on the already-running campaign, got %d: opening a campaign must not spend its budget", got)
	}

	// And the new campaign does not reach BACKWARDS. An account that registered
	// before it opened still belongs to the campaign that was live then — a
	// campaign must not retroactively absorb the signups of the one before it.
	registeredBeforeSecond := baseInstant().Add(-48 * time.Hour)
	early := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredBeforeSecond}
	if !h.granted(t, early) {
		t.Fatalf("expected the account that registered before the second campaign opened to be granted")
	}
	earlyClaim := claims.claimFor(early.UserID)
	if earlyClaim == nil || earlyClaim.CampaignCode != "launch-2026-10" {
		t.Fatalf("expected the earlier account to stay in launch-2026-10, got %+v", earlyClaim)
	}
}

// TestNoCampaignCodeIsWrittenIntoTheSource is the checkable half of "no source
// change".
//
// The mechanism must not know any campaign BY NAME. It scans code rather than
// prose — only string literals, reached through the AST — because what would
// break FR-011 is a comparison against a campaign code, and a code can only reach
// code as a literal. A comment that mentions a campaign while explaining the
// rules is documentation, not a violation.
//
// Add a code here when a campaign is declared. A campaign that is never named in
// code needs no entry at all, which is the whole claim this test makes.
func TestNoCampaignCodeIsWrittenIntoTheSource(t *testing.T) {
	codes := []string{"launch-2026-10", "sample-2026-11"}
	sources := campaignSourceFiles(t)
	if len(sources) == 0 {
		t.Fatalf("no campaign source files found; the guard would pass vacuously")
	}
	for _, file := range sources {
		for _, literal := range stringLiteralsOf(t, file) {
			for _, code := range codes {
				if strings.Contains(literal, code) {
					t.Errorf("%s contains the literal %q naming the campaign %q; a campaign is "+
						"configuration, not code (FR-011)", file.name, literal, code)
				}
			}
		}
	}
}

// stringLiteralsOf returns every string literal in one source file.
//
// The AST rather than a substring search, because a substring search cannot tell
// a comparison against a campaign code from a sentence that mentions one, and a
// guard that cries wolf over comments is a guard that gets deleted.
func stringLiteralsOf(t *testing.T, file campaignSourceFile) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file.name, file.body, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", file.name, err)
	}
	var literals []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			literals = append(literals, lit.Value)
		}
		return true
	})
	return literals
}

// TestAnAccountCreatedAfterEndsAtIsNotGrantedWhenTheMessageArrivesLate is
// T085 / FR-017.
//
// `ends_at` closes a campaign for accounts CREATED after it, and the only instant
// that decides this is `registered_at` — never the moment the message is finally
// consumed. A queue that backs up for five days must not quietly become a campaign
// extension: the lateness of the MESSAGE is the queue's problem, while the
// customer registered outside the window. The control below pins the other side of
// the same boundary, so the refusal cannot be attributed to any other rule.
func TestAnAccountCreatedAfterEndsAtIsNotGrantedWhenTheMessageArrivesLate(t *testing.T) {
	closedStarts := baseInstant().Add(-10 * 24 * time.Hour)
	closedEnds := baseInstant().Add(-5 * 24 * time.Hour)
	openStarts := baseInstant().Add(-3 * 24 * time.Hour)

	// A second campaign, still awake, so the refusal below is attributable to the
	// closed campaign's WINDOW and not to the mechanism having switched itself
	// off. Without it the answer would be `campaign_sleeping`, and the test would
	// prove nothing at all about `ends_at`.
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &closedStarts, &closedEnds),
		testCampaignConfig("sample-2026-11", "premium-monthly", 10, 2, &openStarts, nil),
	}

	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	// Control: the same campaigns, with an account registered two hours BEFORE
	// the window shut and a clock still inside it — the on-time case, which is
	// granted. Its only job is to make `ends_at` the discriminating fact: without
	// it, a refusal below could be blamed on any other rule in the table.
	control := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: closedEnds.Add(-2 * time.Hour),
	}
	h.uc.now = func() time.Time { return closedEnds.Add(-time.Hour) }
	if !h.granted(t, control) {
		t.Fatalf("control: an account registered inside the window must be granted")
	}
	controlClaim := claims.claimFor(control.UserID)
	if controlClaim == nil || controlClaim.CampaignCode != "launch-2026-10" {
		t.Fatalf("control: expected the grant to come from launch-2026-10, got %+v", controlClaim)
	}

	// The account under test: registered one second AFTER `ends_at`, and its
	// message arrives five days later — long after the campaign closed and while
	// a different campaign is running.
	tooLate := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: closedEnds.Add(time.Second),
	}
	h.uc.now = func() time.Time { return baseInstant() }
	if err := h.uc.ProcessUserCreated(t.Context(), tooLate); err != nil {
		t.Fatalf("unexpected error: a skip is a success, not a failure: %v", err)
	}

	if claims.claimFor(tooLate.UserID) != nil {
		t.Fatalf("an account created after ends_at must not be granted, however late its message arrives (FR-017)")
	}
	if sub, _ := h.subs.GetByUserID(t.Context(), tooLate.UserID); sub != nil {
		t.Fatalf("expected no plan to be applied to an account created after ends_at, got plan %q", sub.CurrentPlanCode)
	}
	// The reason names the rule that refused, which is what an operator reads to
	// decide whether to act: `outside_window` means "this account arrived too
	// late", and it is a different answer from "the campaign is over".
	reason, ok := h.lastSkip(t, tooLate.UserID)
	if !ok || reason != SkipOutsideWindow {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipOutsideWindow, reason, ok)
	}

	// A refused account must not have consumed budget either. Only the control
	// account holds a claim on the closed campaign.
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 1 {
		t.Fatalf("expected only the control account to hold a claim on launch-2026-10, got %d", got)
	}
}

// TestAnExpiredCampaignDoesNotGrantAnAccountTheRunningCampaignMisses is §6.1's
// first term — `c.still_running(now)` — on the only shape of configuration where
// it can be observed at all: a second campaign that is still open.
//
// The single-campaign version of FR-013 is already covered above, because
// condition 2 (`anyCampaignAwake`) refuses a mechanism whose every campaign has
// closed. It cannot see a campaign that has closed while ANOTHER one runs, and
// that is precisely the configuration a seasonal programme produces: the
// summer campaign keeps its real `ends_at` in the file after it is over, and the
// autumn campaign is declared beside it.
//
// The account under test registered inside the EXPIRED campaign's window and
// before the running one opened — so it belonged to a campaign that has since
// closed, and belongs to no campaign at all now. `selectCampaign` used to ignore
// `still_running` entirely and grant it from the closed campaign, handing out a
// plan whose giveaway window had been shut for days.
func TestAnExpiredCampaignDoesNotGrantAnAccountTheRunningCampaignMisses(t *testing.T) {
	closedStarts := baseInstant().Add(-10 * 24 * time.Hour)
	closedEnds := baseInstant().Add(-5 * 24 * time.Hour)
	openStarts := baseInstant().Add(-3 * 24 * time.Hour)

	campaigns := []sharedconfig.CampaignConfig{
		// A real `ends_at` that has already passed — the only kind of end there is.
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &closedStarts, &closedEnds),
		testCampaignConfig("sample-2026-11", "premium-monthly", 10, 2, &openStarts, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	// Control: an account registered inside the RUNNING campaign's window is
	// granted by it, at this very clock. Without it, the refusal below could be
	// explained by the mechanism being off, by an exhausted budget, or by the
	// second campaign simply not applying to anyone.
	fresh := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant().Add(-time.Hour),
	}
	if !h.granted(t, fresh) {
		t.Fatalf("control: the running campaign must still grant an account inside its window")
	}
	freshClaim := claims.claimFor(fresh.UserID)
	if freshClaim == nil || freshClaim.CampaignCode != "sample-2026-11" {
		t.Fatalf("control: expected the running campaign to grant, got %+v", freshClaim)
	}

	// The account under test: inside the CLOSED campaign's window by creation time,
	// before the running campaign opened. Its message arrives while the second
	// campaign is live, which is the ordinary case for a queue that is days behind.
	orphan := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: closedStarts.Add(time.Hour),
	}
	if err := h.uc.ProcessUserCreated(t.Context(), orphan); err != nil {
		t.Fatalf("unexpected error: a skip is a success, not a failure: %v", err)
	}

	if claims.claimFor(orphan.UserID) != nil {
		t.Fatalf("a campaign past ends_at must not grant from its old window (FR-013, data-model.md §6.1); " +
			"the grant came from a campaign that had already closed")
	}
	if sub, _ := h.subs.GetByUserID(t.Context(), orphan.UserID); sub != nil {
		t.Fatalf("expected no plan applied to the account, got plan %q", sub.CurrentPlanCode)
	}

	// The reason is `outside_window`, and the alternative is worth naming. It is
	// NOT `campaign_sleeping`: condition 2 refuses only when EVERY campaign has
	// closed, and one is running right here — reporting "sleeping" would send an
	// operator to a campaign that is open and healthy. `outside_window` is
	// condition 7's honest answer: no live campaign's window covers this account.
	reason, ok := h.lastSkip(t, orphan.UserID)
	if !ok || reason != SkipOutsideWindow {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipOutsideWindow, reason, ok)
	}

	// The closed campaign spent nothing, and the only grant in the ledger is the
	// control account's. Both matter commercially: the closed campaign's budget
	// belongs to whoever runs it next, and a silent grant would look identical to
	// a correct one in every number the operator sees.
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 0 {
		t.Fatalf("expected 0 claims on the closed campaign, got %d", got)
	}
	if got := claims.countBucket("sample-2026-11", entities.GrantBucketMain); got != 1 {
		t.Fatalf("expected exactly 1 claim on the running campaign (the control), got %d", got)
	}
	if got := h.publisher.count(); got != 1 {
		t.Fatalf("expected exactly 1 plan-change event in the whole run, got %d", got)
	}
}

// TestAnAccountThatJoinedAnEarlierCampaignIsSkippedByALaterOne is T086 /
// FR-012: one account, one participation, for the whole lifetime of the
// mechanism.
//
// The scenario is the one a second campaign actually creates. The account
// registers while campaign A is the only campaign declared and is granted from A.
// The operator then opens campaign B — configuration only — with a window
// deliberately covering that account's registration, because a new campaign is
// normally back-dated to the moment it was decided. From that point B wins FR-011
// for this account: it starts later than A and still covers the account. The only
// thing standing between the customer and a second free plan is the
// one-participation rule.
func TestAnAccountThatJoinedAnEarlierCampaignIsSkippedByALaterOne(t *testing.T) {
	earlierStarts := baseInstant().Add(-72 * time.Hour)
	laterStarts := baseInstant().Add(-48 * time.Hour)

	earlier := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &earlierStarts, nil)
	// A DIFFERENT plan for the second campaign, and the reason is the test's:
	// condition 5 (FR-018) refuses an account already on the chosen campaign's
	// plan, so sharing `plan_slug` would let that rule answer first and the
	// one-participation rule would never be exercised.
	later := testCampaignConfig("sample-2026-11", "premium-quarterly", 10, 2, &laterStarts, nil)

	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{earlier}, claims)

	veteran := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant().Add(-time.Hour),
	}
	if !h.granted(t, veteran) {
		t.Fatalf("expected the only declared campaign to grant the account")
	}
	firstClaim := claims.claimFor(veteran.UserID)
	if firstClaim == nil || firstClaim.CampaignCode != "launch-2026-10" {
		t.Fatalf("expected the first grant to come from launch-2026-10, got %+v", firstClaim)
	}

	// Restart with the second campaign declared. Only the configuration is new:
	// the durable ledger and the subscription row carry over exactly as they
	// would across a deploy — FR-012 holds "even after a restart", so cross one.
	carried, err := h.subs.GetByUserID(t.Context(), veteran.UserID)
	if err != nil || carried == nil || carried.ExpiresAt == nil {
		t.Fatalf("read the granted subscription: %v", err)
	}
	firstExpiry := *carried.ExpiresAt
	restarted := newTestUseCase(t, []sharedconfig.CampaignConfig{earlier, later}, claims)
	restarted.subs.put(carried)

	// The control runs first, so the refusal below cannot be mistaken for "the
	// second campaign does not apply to that account".
	publishedBefore := assertSecondCampaignTakesAFreshAccount(t, restarted, claims, veteran.RegisteredAt)

	// Now the account that already participated.
	if err := restarted.uc.ProcessUserCreated(t.Context(), veteran); err != nil {
		t.Fatalf("unexpected error: a skip is a success, not a failure: %v", err)
	}
	reason, ok := restarted.lastSkip(t, veteran.UserID)
	if !ok || reason != SkipAlreadyClaimed {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipAlreadyClaimed, reason, ok)
	}

	// Exactly one claim for that account, and the one the FIRST campaign wrote.
	claim := claims.claimFor(veteran.UserID)
	if claim == nil || claim.CampaignCode != "launch-2026-10" || claim.PlanSlug != "premium-monthly" {
		t.Fatalf("expected the account's only claim to remain the one from launch-2026-10, got %+v", claim)
	}

	// And no second grant. FR-012's harm is not the missing claim, it is the
	// STACKING: a second term handed out and the customer paying nothing for it.
	if got := restarted.publisher.count(); got != publishedBefore {
		t.Fatalf("expected no plan-change event for an account that already participated, got %d", got-publishedBefore)
	}
	sub, err := restarted.subs.GetByUserID(t.Context(), veteran.UserID)
	if err != nil || sub == nil {
		t.Fatalf("read subscription after the refusal: %v", err)
	}
	if sub.CurrentPlanCode != "premium-monthly" {
		t.Fatalf("expected the account to stay on premium-monthly, got %q", sub.CurrentPlanCode)
	}
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(firstExpiry) {
		t.Fatalf("the term was extended to %v; the first campaign granted it to %s", sub.ExpiresAt, firstExpiry)
	}

	// Neither campaign spent budget on an account it may not have.
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 1 {
		t.Fatalf("expected exactly 1 claim on launch-2026-10, got %d", got)
	}
	if got := claims.countBucket("sample-2026-11", entities.GrantBucketMain); got != 1 {
		t.Fatalf("expected exactly 1 claim on sample-2026-11 (the control account only), got %d", got)
	}
}

// assertSecondCampaignTakesAFreshAccount is the CONTROL of the one-participation
// test, kept as its own scenario because it is a different claim about the
// mechanism: with the second campaign declared, an account that has never
// participated and registers at the same instant IS granted by it.
//
// Without this control, a refusal could be read as "the second campaign does not
// apply to that account" — a bug in campaign selection wearing the same symptom
// as FR-012 working. It returns the number of published plan changes so far, which
// is the baseline the refusal is measured against.
func assertSecondCampaignTakesAFreshAccount(
	t *testing.T,
	restarted *testHarness,
	claims *fakeClaimRepo,
	registeredAt time.Time,
) int {
	t.Helper()
	fresh := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}
	if !restarted.granted(t, fresh) {
		t.Fatalf("control: expected the second campaign to grant an eligible account")
	}
	freshClaim := claims.claimFor(fresh.UserID)
	if freshClaim == nil || freshClaim.CampaignCode != "sample-2026-11" {
		t.Fatalf("control: expected the second campaign to win for a fresh account at the same instant, got %+v", freshClaim)
	}
	// The control's own grant is the baseline: only a GROWTH after the re-run is
	// a second grant.
	return restarted.publisher.count()
}

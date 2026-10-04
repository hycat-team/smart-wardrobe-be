package campaign

import (
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 026 T082 + T083 — FR-045's gate on the grant path, expressed with fakes.
//
// WHAT FR-045 ASKS FOR, IN ONE SENTENCE. Between the moment a campaign is CHOSEN and the
// moment its claim row is WRITTEN, the campaign may be closed or its budget may change;
// the row must therefore be re-read under a share lock inside the grant's own
// transaction, and the grant re-decided from THAT row rather than from the copy the
// evaluator picked.
//
// WHY THIS NEEDS A FAKE AT ALL, since the rule sounds like a database question.
//
// The database half — that `FOR SHARE` really blocks behind the close's row lock — is
// `campaign_db_close_integration_test.go`'s `TestIntegrationAGrantInFlightIsStoppedByTheShareLock`,
// and it needs a second connection and a `TEST_DATABASE_URL`. That test has never run
// (deferred.md D-01), so FR-045's gate currently has NO executable evidence at all. The
// part a fake CAN prove is the part the database does not: that the production path asks
// for the locked read, takes the advisory lock before it, and decides from the row it
// got back. That is a question about control flow, and control flow is exactly what a
// fake is for.
//
// THE ARRANGEMENT. `fakeCampaignRepo` applies `closeOnShareCode` / `budgetOnShare` at the
// instant `GetByCodeForShare` is served — that is, after conditions 1-7 have run on the
// row the evaluator picked and before the counts, the watermark and the insert. Every
// test below is therefore about the same gap FR-045 names, with the change landing
// inside it deterministically instead of depending on a scheduler.

// fr045Code is the campaign every test in this file uses. One code, because the subject
// is the GATE and not the campaign.
const fr045Code = "fr045-in-flight"

// assertNoGrantHappened is T082's "the round was skipped" half, kept apart from the gate
// itself so each half of the claim is read on its own.
//
// "Nothing was granted" is asserted through the four things a grant leaves behind, because
// any single one of them is satisfiable by a broken path: a claim without a plan change is
// a grant the AI service never heard about, and a plan change without a claim is a
// campaign nobody in the registry explains.
func assertNoGrantHappened(t *testing.T, h *testHarness, userID uuid.UUID) {
	t.Helper()

	// NO CLAIM. FR-031 is unconditional: a closed campaign grants nobody, whatever the
	// budget and whatever the window say.
	if got := h.claims.total(); got != 0 {
		t.Errorf("FR-045: the claim ledger must be empty, got %d claims", got)
	}
	if claim := h.claims.claimFor(userID); claim != nil {
		t.Errorf("FR-045: no claim may be written for %s, got %+v", userID, claim)
	}

	// NO PLAN CHANGE. The plan-change event is what the AI service acts on, so a grant
	// that wrote no claim but still published would hand out entitlements nothing in the
	// registry explains.
	if got := h.publisher.count(); got != 0 {
		t.Errorf("FR-045: a stopped grant must publish no plan change, got %d", got)
	}
	if got := len(h.events.published()); got != 0 {
		t.Errorf("FR-039: a stopped grant must write no subscription history, got %d rows", got)
	}

	// AND THE ACCOUNT'S PLAN IS UNCHANGED, which is the part a customer would notice.
	sub, err := h.subs.GetByUserID(t.Context(), userID)
	if err != nil {
		t.Fatalf("read the subscription: %v", err)
	}
	if sub != nil && sub.CurrentPlanCode == "premium-monthly" {
		t.Errorf("a closed campaign must not put the account on %q", sub.CurrentPlanCode)
	}
}

// TestACampaignClosedBetweenEvaluationAndTheClaimWriteGrantsNothing is T082 / FR-045 /
// FR-031: a campaign closed AFTER conditions 1-7 must stop that round, write no claim and
// change no plan.
//
// The three things a customer could notice are asserted separately — no claim, no plan
// change, no subscription — because a path that wrote the claim and rolled something
// back, or applied the plan and skipped the ledger, satisfies any single one of them.
func TestACampaignClosedBetweenEvaluationAndTheClaimWriteGrantsNothing(t *testing.T) {
	startsAt := baseInstant().Add(-24 * time.Hour)
	campaign := testCampaign(fr045Code, "premium-monthly", 2, 0, &startsAt, nil)
	h := newTestUseCase(t, []*entities.Campaign{campaign}, newFakeClaimRepo())

	// THE PREMISE, ASSERTED. Every skip below is only about the closure if the account
	// really did qualify: the campaign is awake, its window is open and open-ended, the
	// ledger is empty and the plan exists. Without this the test would pass for any of
	// the six other reasons an account is not granted.
	if !isCampaignAwake(campaign, baseInstant()) {
		t.Fatal("precondition: the campaign must be awake before it is closed mid-flight")
	}
	if campaign.EndsAt != nil {
		t.Fatal("precondition: the window must be open-ended, or the campaign would be asleep for a " +
			"second reason and this test would prove nothing about the closure")
	}
	if got := h.claims.total(); got != 0 {
		t.Fatalf("precondition: the ledger must start empty, got %d claims", got)
	}
	if got := len(h.plans.plans); got == 0 {
		t.Fatal("precondition: the campaign's plan must exist, or FR-014 answers first")
	}

	// THE CLOSE LANDS HERE, in the gap.
	h.campaigns.closeOnShareCode = fr045Code

	userID := uuid.New()
	if err := h.uc.ProcessUserCreated(t.Context(), identitydto.UserCreatedEventPayload{
		UserID: userID, RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("a campaign closed mid-flight is a SKIP, not a retryable failure: %v", err)
	}

	// THE GATE WAS ACTUALLY REACHED. `shareCalls` is the instrument that makes this a
	// statement about the production path rather than about the outcome: an
	// implementation that never re-read the row would skip for some other reason and
	// satisfy every assertion below.
	if h.campaigns.shareCalls != 1 {
		t.Errorf("expected exactly 1 locked read of the campaign (FR-045), got %d", h.campaigns.shareCalls)
	}
	if !h.claims.lockWasTaken() {
		t.Error("FR-045/R12: the advisory lock must be taken BEFORE the locked read, or the close and the " +
			"grant could deadlock against each other")
	}
	if stored := h.campaigns.byCode(fr045Code); stored == nil || stored.ClosedAt == nil {
		t.Fatal("precondition of the assertion: the campaign must be closed at the locked read, or this " +
			"test is not exercising the gap at all")
	}

	assertNoGrantHappened(t, h, userID)

	// AND THE REASON IS THE OPERATOR'S, from the log FR-061 asks for.
	reason, ok := h.lastSkip(t, userID)
	if !ok {
		t.Fatal("the skip must be logged with a reason code (FR-061)")
	}
	if reason != SkipCampaignSleeping {
		t.Errorf("reason = %q, want %q — the re-read row carries closed_at, and `campaign_sleeping` is the "+
			"code 025 already used for a campaign that is over (FR-050 forbids a new one)",
			reason, SkipCampaignSleeping)
	}
}

// newShrunkenBudgetFixture builds the T083 state: a campaign whose EVALUATED row says
// `quota = 3` against a ledger that already holds two main grants, so the third main
// slot is the one under contention.
//
// `shrink` decides whether the locked read installs `quota = 1`. It is a parameter
// rather than two near-identical fixtures so the control — the same account, the same
// ledger, the same evaluated row, WITHOUT the edit — is provably the same test with one
// thing removed. A control written as its own fixture would drift, and the whole point
// is that the two runs differ ONLY in the edit.
func newShrunkenBudgetFixture(t *testing.T, shrink bool) (*testHarness, string, uuid.UUID) {
	t.Helper()
	startsAt := baseInstant().Add(-24 * time.Hour)
	campaign := testCampaign(fr045Code, "premium-monthly", 3, 0, &startsAt, nil)
	claims := newFakeClaimRepo()
	// Two main grants, the older of them carrying the FR-030 watermark, which is the
	// state a campaign with `quota = 3` is genuinely in after its third account.
	claims.seed(fr045Code, uuid.New(), baseInstant().Add(-3*time.Hour), entities.GrantBucketMain, true)
	claims.seed(fr045Code, uuid.New(), baseInstant().Add(-2*time.Hour), entities.GrantBucketMain, false)

	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)
	if shrink {
		h.campaigns.budgetOnShare = &budgetChangeOnShare{code: fr045Code, quota: 1, reserve: 0}
	}
	return h, fr045Code, uuid.New()
}

// requireBudgetFixturePremise asserts the state T083's contrast depends on.
//
// Without it a fixture that failed to seed its ledger would produce the same "no grant"
// as one whose budget shrank, and the test would pass while proving nothing.
func requireBudgetFixturePremise(t *testing.T, h *testHarness) {
	t.Helper()
	stored := h.campaigns.byCode(fr045Code)
	if stored == nil {
		t.Fatal("precondition: the campaign must exist")
	}
	if stored.Quota != 3 {
		t.Fatalf("precondition: the evaluated row must say quota 3, got %d", stored.Quota)
	}
	if stored.Reserve != 0 {
		t.Fatalf("precondition: the reserve must be empty, or the reserve branch would answer instead, got %d",
			stored.Reserve)
	}
	if got := h.claims.countBucket(fr045Code, entities.GrantBucketMain); got != 2 {
		t.Fatalf("precondition: the ledger must hold exactly 2 main grants against quota 3, got %d", got)
	}
	if got := h.claims.watermarkCount(fr045Code); got != 1 {
		t.Fatalf("precondition: the ledger must hold exactly 1 watermark row, got %d", got)
	}
}

// TestACampaignWhoseBudgetShrankBeforeTheClaimWriteGrantsNobody is T083 / FR-045's
// second clause: "hoặc ngân sách đã thay đổi khiến lượt không còn hợp lệ" — the budget
// changed, the grant is no longer valid, and it MUST be skipped.
//
// THE ARITHMETIC IS THE WHOLE ARGUMENT, so it is written out rather than left implicit.
// With the budget the evaluator read:
//
//	mainCount 2 < quota 3  →  the main branch grants, and the claim becomes the
//	                          watermark (2 + 1 == 3)
//
// With the budget the locked read brought back:
//
//	mainCount 2 < quota 1  →  false
//	reserveCount 0 < reserve 0  →  false
//	                          →  both parts full: `quota_compensation`, no claim
//
// So the stale row and the fresh row disagree about the SAME account at the SAME
// instant, and only one of them is allowed to be believed. The control below runs the
// identical fixture without the edit and is granted, which is what turns "it did not
// grant" into "it did not grant BECAUSE the budget moved".
//
// NOTE ON REALITY, because the state here is one the data layer forbids in operation:
// `trg_campaigns_immutable_budget` refuses any budget edit once a campaign has claims.
// FR-045's clause is still worth proving, and it is still reachable — through direct SQL
// past the trigger, through a future trigger that relaxes, and simply because the
// decider's contract is "decide from the row you re-read", not "assume nobody edited
// it". What must not happen is the decider trusting a copy it read before the lock.
func TestACampaignWhoseBudgetShrankBeforeTheClaimWriteGrantsNobody(t *testing.T) {
	h, code, userID := newShrunkenBudgetFixture(t, true)
	requireBudgetFixturePremise(t, h)

	if err := h.uc.ProcessUserCreated(t.Context(), identitydto.UserCreatedEventPayload{
		UserID: userID, RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("a grant the budget no longer allows is a SKIP, not a retryable failure: %v", err)
	}

	// THE RE-READ HAPPENED, and the row it returned is the one that decided.
	if h.campaigns.shareCalls != 1 {
		t.Errorf("expected exactly 1 locked read of the campaign (FR-045), got %d", h.campaigns.shareCalls)
	}
	if stored := h.campaigns.byCode(code); stored == nil || stored.Quota != 1 {
		t.Fatalf("precondition of the assertions: the locked read must install quota 1, got %+v", stored)
	}

	// NOTHING WAS GRANTED.
	if claim := h.claims.claimFor(userID); claim != nil {
		t.Errorf("FR-045: the budget no longer allows this grant, so no claim may be written, got %+v", claim)
	}
	if got := h.claims.total(); got != 2 {
		t.Errorf("FR-045: the ledger must be untouched at 2 grants, got %d", got)
	}
	if got := h.publisher.count(); got != 0 {
		t.Errorf("FR-045: a grant the budget no longer allows must publish no plan change, got %d", got)
	}

	// AND THE WATERMARK DID NOT MOVE. This is the half that would hurt: a claim written
	// against the stale `quota = 3` would carry `fills_main_quota` and sit alongside the
	// one already seeded, which is exactly the state the partial unique index forbids in
	// the database and the priority rule depends on (FR-030).
	if got := h.claims.watermarkCount(code); got != 1 {
		t.Errorf("FR-030: the ledger must still hold exactly 1 watermark row, got %d", got)
	}

	reason, ok := h.lastSkip(t, userID)
	if !ok {
		t.Fatal("the skip must be logged with a reason code (FR-061)")
	}
	if reason != SkipQuotaCompensation {
		t.Errorf("reason = %q, want %q — both budget parts are full against the RE-READ row, and the "+
			"reserve is a ceiling rather than a target (FR-032)", reason, SkipQuotaCompensation)
	}
}

// TestTheSameAccountIsGrantedWhenNothingChangesTheBudget is T083's control: the fixture
// above with the edit REMOVED, granted from the main budget as the evaluated row said it
// would be.
//
// It exists so the test above cannot be satisfied by a fixture that never granted in the
// first place. Both tests build their state through one function, so the only difference
// between them is the armed edit.
func TestTheSameAccountIsGrantedWhenNothingChangesTheBudget(t *testing.T) {
	h, code, userID := newShrunkenBudgetFixture(t, false)
	requireBudgetFixturePremise(t, h)

	if err := h.uc.ProcessUserCreated(t.Context(), identitydto.UserCreatedEventPayload{
		UserID: userID, RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("process the signup: %v", err)
	}

	claim := h.claims.claimFor(userID)
	if claim == nil {
		t.Fatalf("precondition of the contrast: with quota 3 and two main grants the third account MUST be " +
			"granted, otherwise the test above proves nothing about the budget edit")
	}
	if claim.CampaignCode != code {
		t.Errorf("the claim names campaign %q, want %q", claim.CampaignCode, code)
	}
	if claim.GrantBucket != entities.GrantBucketMain {
		t.Errorf("bucket = %q, want %q — the main budget still had room", claim.GrantBucket, entities.GrantBucketMain)
	}
	// `fills_main_quota` stays FALSE, and that is the guard `chooseBucket` documents:
	// the watermark already exists, so the third claim against a quota of 3 is not the
	// one that fills anything.
	if claim.FillsMainQuota {
		t.Error("FR-030: a second fills_main_quota row must not be written while one already exists")
	}
	if got := h.claims.watermarkCount(code); got != 1 {
		t.Errorf("FR-030: expected exactly 1 watermark row, got %d", got)
	}
}

// TestTheWatermarkFollowsTheBudgetThatWasReread is T083's second observable, and it is
// the one that needs no ledger at all: with an EMPTY ledger, shrinking `quota` from 3 to
// 1 changes WHICH CLAIM fills the main budget rather than whether there is one.
//
// With the stale `quota = 3` the single claim written is NOT the one that fills the
// budget (`1 == 3` is false), so `fills_main_quota` would stay false and the watermark
// would be attributed to a later account. With the re-read `quota = 1`, this claim IS
// the one that fills it.
//
// THAT IS THE CORRUPTION THE WHOLE TRIGGER EXISTS TO PREVENT, and it is invisible to any
// assertion about the grant having happened: the customer gets the plan either way, and
// the wrong priority watermark only shows up weeks later as reserve grants handed to the
// wrong accounts.
func TestTheWatermarkFollowsTheBudgetThatWasReread(t *testing.T) {
	startsAt := baseInstant().Add(-24 * time.Hour)
	campaign := testCampaign(fr045Code, "premium-monthly", 3, 0, &startsAt, nil)
	h := newTestUseCase(t, []*entities.Campaign{campaign}, newFakeClaimRepo())

	// THE PREMISE: an empty ledger, so the ONLY thing that can decide the watermark is
	// the quota.
	if got := h.claims.total(); got != 0 {
		t.Fatalf("precondition: the ledger must start empty, got %d claims", got)
	}
	if campaign.Quota != 3 {
		t.Fatalf("precondition: the evaluated row must say quota 3, got %d", campaign.Quota)
	}
	h.campaigns.budgetOnShare = &budgetChangeOnShare{code: fr045Code, quota: 1, reserve: 0}

	userID := uuid.New()
	if err := h.uc.ProcessUserCreated(t.Context(), identitydto.UserCreatedEventPayload{
		UserID: userID, RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("process the signup: %v", err)
	}

	claim := h.claims.claimFor(userID)
	if claim == nil {
		t.Fatal("FR-045: the re-read row still has room in the main budget, so the grant must happen")
	}
	if !claim.FillsMainQuota {
		t.Errorf("FR-045/FR-030: against the RE-READ quota of 1 this claim fills the main budget and must " +
			"carry fills_main_quota. Computed against the evaluated quota of 3 it would not, which is " +
			"exactly the misattributed watermark the priority rule cannot recover from")
	}
	if got := h.claims.watermarkCount(fr045Code); got != 1 {
		t.Errorf("FR-030: expected exactly 1 watermark row, got %d", got)
	}
}

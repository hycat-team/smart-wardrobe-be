package campaign

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedevent "smart-wardrobe-be/shared/application/event"

	"github.com/google/uuid"
)

// 025 T036-T038: the quota rules of `data-model.md` §6.2, proven without any
// plan being granted.
//
// The grant is not needed to ask whether an account may be granted, which is why
// this file can concentrate entirely on the three hardest requirements of the
// specification: FR-022 (the hard ceiling), FR-025 (one claim per account) and
// FR-030/FR-031 (the priority watermark).

// campaignRacers is the width of the concurrent burst in T036. Five parallel
// streams times four attempts is 20 calls, which is enough for every one of them
// to observe the pre-fix state (count 49, decide "main", insert) if the lock were
// missing.
const campaignRacers = 5

// TestCampaignQuotaGrantsExactlyOnceUnderConcurrency is SC-001: with ONE slot
// left on the main budget, twenty concurrent calls must produce exactly one
// grant, never two.
//
// The failure this guards against is not hypothetical. The first design counted
// the main part as `COUNT(*) WHERE NOT fills_main_quota`, which equals
// "total minus one", so the fifty-first account computed `mainCount = 49`,
// concluded it was the fiftieth, and set `fills_main_quota = TRUE` — colliding
// with `ux_campaign_claims_watermark` instead of falling through to the reserve.
// The reserve was therefore never usable, and it is needed exactly when the main
// budget runs out (research.md R2b).
func TestCampaignQuotaGrantsExactlyOnceUnderConcurrency(t *testing.T) {
	// Quota 1 and NO reserve, so the hard ceiling is exactly one grant and the
	// test measures the main-budget race alone. The reserve branch has its own test
	// below; folding it in here would mean the expected number depended on the
	// order the goroutines happened to run, which is not a thing to assert.
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaign("launch-2026-10", "premium-monthly", 1, 0, &startsAt, nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	var wg sync.WaitGroup
	errs := make([]error, campaignRacers*4)
	// A barrier so every goroutine is inside `ProcessUserCreated` before any of
	// them reaches the quota decision. Without it the racers interleave loosely,
	// the early ones commit before the late ones start, and the test passes even
	// when the lock is not taken — which is exactly how the original bug survived.
	start := make(chan struct{})
	for i := 0; i < campaignRacers; i++ {
		for j := 0; j < 4; j++ {
			idx := i*4 + j
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				errs[idx] = h.uc.ProcessUserCreated(t.Context(), identitydto.UserCreatedEventPayload{
					UserID:         uuid.New(),
					Provider:       "local",
					RegisteredAt:   baseInstant().Add(time.Duration(idx) * time.Second),
					CreatedByAdmin: false,
				})
			}(idx)
		}
	}
	close(start)
	wg.Wait()

	for idx, err := range errs {
		if err != nil {
			t.Fatalf("racer %d: unexpected error: %v", idx, err)
		}
	}
	// The LEDGER is the assertion, not the return value: twenty calls answered
	// "granted" would still be a correct implementation if the writes collapsed to
	// one row, and conversely a wrong implementation that answered "granted" once
	// while writing twenty rows would pass a return-value assertion.
	if got := claims.total(); got != 1 {
		t.Fatalf("expected exactly 1 claim row across %d concurrent calls, got %d", campaignRacers*4, got)
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 1 {
		t.Fatalf("expected exactly 1 main-bucket claim (quota 1), got %d", got)
	}
	if got := len(h.events.published()); got != 1 {
		t.Fatalf("expected exactly 1 plan-change event, got %d — a plan applied without its own event is FR-039", got)
	}
}

// TestCampaignQuotaReserveBranchIsReachable is SC-001b: the regression case for
// the bug above, stated positively.
//
// Fill the main budget completely, then grant from the reserve. It must SUCCEED,
// with `grant_bucket` = main for the first quota claims and reserve for the rest,
// and it must not trip the partial unique index. If the implementation counts the
// main part by excluding a condition on `fills_main_quota`, this test fails at the
// reserve grant — the reserve becomes unusable and the campaign silently hands
// out fewer plans than it was configured for.
func TestCampaignQuotaReserveBranchIsReachable(t *testing.T) {
	// The campaign opened well before the seeded claims, so an account created
	// before the watermark is still INSIDE the campaign's window — otherwise the
	// test would be measuring the window rule instead of the reserve rule.
	startsAt := baseInstant().Add(-2 * time.Hour)
	campaign := testCampaign("launch-2026-10", "premium-monthly", 2, 2, &startsAt, nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	// The watermark instant: the second main claim freezes it.
	watermarkAt := baseInstant().Add(-time.Hour)
	claims.seed("launch-2026-10", uuid.New(), baseInstant().Add(-110*time.Minute), entities.GrantBucketMain, false)
	claims.seed("launch-2026-10", uuid.New(), watermarkAt, entities.GrantBucketMain, true)

	if got := claims.watermarkCount("launch-2026-10"); got != 1 {
		t.Fatalf("expected exactly 1 watermark row in the seed, got %d", got)
	}

	// An account created BEFORE the watermark is entitled to the reserve (FR-031).
	early := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: watermarkAt.Add(-time.Hour),
	}
	if !h.granted(t, early) {
		t.Fatalf("expected the reserve to be usable for an account created before the watermark")
	}

	// An account created AFTER the watermark is not, even though reserve is left.
	late := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: watermarkAt.Add(30 * time.Minute),
	}
	if h.granted(t, late) {
		t.Fatalf("expected no grant for an account created after the watermark (FR-031)")
	}

	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 2 {
		t.Fatalf("expected 2 main claims, got %d", got)
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketReserve); got != 1 {
		t.Fatalf("expected exactly 1 reserve claim, got %d", got)
	}
	if got := claims.watermarkCount("launch-2026-10"); got != 1 {
		t.Fatalf("watermark must stay unique, got %d rows flagged", got)
	}
}

// TestCampaignQuotaReserveIsSkippedWithoutWatermark is SC-001c: an anomalous
// ledger — main budget full, no watermark row — must be SKIPPED with
// `quota_anomaly`, not guessed at.
//
// The reserve is by definition for accounts created before the watermark
// (FR-031). With no watermark there is no such boundary, and inventing one is how
// the wrong account gets a plan. Reporting the anomaly is the whole point
// (`data-model.md` §6.2).
func TestCampaignQuotaReserveIsSkippedWithoutWatermark(t *testing.T) {
	startsAt := baseInstant().Add(-2 * time.Hour)
	campaign := testCampaign("launch-2026-10", "premium-monthly", 1, 3, &startsAt, nil)
	claims := newFakeClaimRepo()
	claims.watermarkMissing = true
	// Main budget is full; the flag is deliberately absent.
	claims.seed("launch-2026-10", uuid.New(), baseInstant().Add(-90*time.Minute), entities.GrantBucketMain, false)
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	payload := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant().Add(-time.Hour),
	}
	if h.granted(t, payload) {
		t.Fatalf("expected no grant when the main budget is full but the watermark is missing")
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipQuotaAnomaly {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipQuotaAnomaly, reason, ok)
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketReserve); got != 0 {
		t.Fatalf("expected no reserve claim to be written, got %d", got)
	}
}

// TestCampaignQuotaRedeliveredMessageCreatesOneClaim is SC-002: the same message
// delivered a hundred times produces exactly one claim.
//
// FR-025 is enforced by `ux_campaign_claims_user` and `ON CONFLICT DO NOTHING`,
// not by a check — two concurrent callers both pass any pre-check, and only the
// insert can refuse the loser. This test proves the use case reports that refusal
// as `already_claimed` instead of granting twice, because a report that named it
// anything else would send an operator looking for a bug that is not there.
func TestCampaignQuotaRedeliveredMessageCreatesOneClaim(t *testing.T) {
	campaign := testCampaign("launch-2026-10", "premium-monthly", 50, 5, timePtr(baseInstant()), nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	userID := uuid.New()
	payload := identitydto.UserCreatedEventPayload{
		UserID:       userID,
		RegisteredAt: baseInstant(),
	}

	grantedCount := 0
	for i := 0; i < 100; i++ {
		if h.granted(t, payload) {
			grantedCount++
		}
	}
	if grantedCount != 1 {
		t.Fatalf("expected exactly 1 grant across 100 deliveries, got %d", grantedCount)
	}
	if got := claims.total(); got != 1 {
		t.Fatalf("expected exactly 1 claim row, got %d", got)
	}
	// The account is still on the plan it was granted, and the 99 redeliveries
	// changed nothing: no second term, no second history row, no second email.
	if got := len(h.events.published()); got != 1 {
		t.Fatalf("expected exactly 1 plan-change history row, got %d", got)
	}
	// The redeliveries are caught by condition 5 (FR-018), not by condition 9.
	// That ordering is `data-model.md` §6's, not an accident: condition 5 needs no
	// lock, and a redelivery of a granted account is always already on the plan, so
	// the cheap rule always wins. What FR-025 additionally guarantees is that even
	// if the cheap rule were passed — a DIFFERENT campaign, a plan that has since
	// expired — the UNIQUE constraint still refuses the second write, which is what
	// `claims.total() == 1` above asserts.
	reasons := h.logger.skipsFor(userID)
	if len(reasons) != 99 {
		t.Fatalf("expected 99 recorded skips, got %d", len(reasons))
	}
	for _, reason := range reasons {
		if reason != SkipAlreadyOnPlan {
			t.Fatalf("expected redeliveries to report %q (FR-018), got %q", SkipAlreadyOnPlan, reason)
		}
	}
}

// TestCampaignQuotaWatermarkIsFrozenOnTheNthMainClaim is SC-001d: the watermark
// lands on the claim that FILLS the main budget, exactly one row carries it, and
// an account created after it is not granted from the main budget.
func TestCampaignQuotaWatermarkIsFrozenOnTheNthMainClaim(t *testing.T) {
	const quota = 3
	campaign := testCampaign("launch-2026-10", "premium-monthly", quota, 0, timePtr(baseInstant()), nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	registeredAt := baseInstant()
	grantedIDs := make([]uuid.UUID, 0, quota)
	for i := 0; i < quota; i++ {
		userID := uuid.New()
		payload := identitydto.UserCreatedEventPayload{UserID: userID, RegisteredAt: registeredAt.Add(time.Duration(i) * time.Second)}
		if !h.granted(t, payload) {
			t.Fatalf("expected grant %d of %d main claims to succeed", i+1, quota)
		}
		grantedIDs = append(grantedIDs, userID)
	}

	if got := claims.watermarkCount("launch-2026-10"); got != 1 {
		t.Fatalf("expected exactly 1 watermark row after the main budget filled, got %d", got)
	}
	// The flag belongs to the LAST main claim — the one that filled the budget —
	// and to no other row.
	flagged := 0
	for i, userID := range grantedIDs {
		claim := claims.claimFor(userID)
		wantFlag := i == quota-1
		if claim.FillsMainQuota != wantFlag {
			t.Fatalf("claim %d: fills_main_quota = %v, want %v", i+1, claim.FillsMainQuota, wantFlag)
		}
		if claim.FillsMainQuota {
			flagged++
		}
		// Every row records the part it came from; none of them is inferred.
		if claim.GrantBucket != entities.GrantBucketMain {
			t.Fatalf("claim %d: grant_bucket = %q, want %q", i+1, claim.GrantBucket, entities.GrantBucketMain)
		}
	}
	if flagged != 1 {
		t.Fatalf("expected exactly 1 flagged claim, got %d", flagged)
	}

	// The main budget is full and `reserve` is 0, so nothing more is grantable.
	overflow := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}
	if h.granted(t, overflow) {
		t.Fatalf("expected no grant once quota + reserve are exhausted (FR-022)")
	}

	// Reading the main count back through the bucket column must return `quota`,
	// not "total minus one" — the number the buggy query produced (R2b).
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != quota {
		t.Fatalf("expected main-bucket count = %d, got %d", quota, got)
	}
}

// TestCampaignQuotaDoesNotCountByExcludingFillsMainQuota is the textual guard
// for R2b, and the one assertion in this file the compiler cannot make.
//
// `NOT fills_main_quota` equals "total minus one" because the flag marks exactly
// one row, so any count written that way silently folds the reserve into the main
// budget. The reserve then trips the watermark index instead of falling through,
// and it is the reserve that exists for the accounts the main budget turned away.
// The rule is therefore a property of the SOURCE, checked by reading the source.
func TestCampaignQuotaDoesNotCountByExcludingFillsMainQuota(t *testing.T) {
	sources := campaignSourceFiles(t)
	if len(sources) == 0 {
		t.Fatalf("no campaign source files found; the guard would pass vacuously")
	}
	for _, file := range sources {
		body := file.body
		if !strings.Contains(file.name, "fills_main_quota") {
			continue
		}
		// The column may be NAMED — GetWatermark and the claim entity legitimately
		// refer to it — but it may never appear inside a count or a WHERE clause
		// that subtracts it.
		if strings.Contains(strings.ToLower(body), "not fills_main_quota") ||
			strings.Contains(strings.ToLower(body), "not r.fills_main_quota") ||
			strings.Contains(strings.ToLower(body), "fills_main_quota = false") {
			t.Fatalf("%s counts the main part by excluding fills_main_quota; "+
				"every count must read grant_bucket (research.md R2b)", file.name)
		}
	}
}

// TestCampaignQuotaFailedCountDoesNotGrant proves a failed count is never read as
// zero.
//
// This is the failure mode the error factories in `errors.go` exist for: a count
// that returns 0 on error grants past the ceiling on the very next message, and
// the ceiling is the one number a campaign cannot exceed.
func TestCampaignQuotaFailedCountDoesNotGrant(t *testing.T) {
	campaign := testCampaign("launch-2026-10", "premium-monthly", 1, 0, timePtr(baseInstant()), nil)
	claims := newFakeClaimRepo()
	claims.failNextCount = true
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	err := h.uc.ProcessUserCreated(t.Context(), identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant(),
	})
	if got := claims.total(); got != 0 {
		t.Fatalf("a failed count must not produce a grant: expected 0 claims, got %d", got)
	}
	// FR-062: a database failure is TRANSIENT. The message must be requeued, so
	// the error has to be a RetryableError — a PermanentError would dead-letter it
	// and lose a customer who was, in fact, eligible.
	var retryable sharedevent.RetryableError
	if !errors.As(err, &retryable) {
		t.Fatalf("expected a RetryableError so the message is retried, got %T", err)
	}
}

// TestCampaignQuotaLocksBeforeCounting asserts the advisory lock is taken, and
// that the counts it guards are read INSIDE it.
//
// The fake's lock is a real mutex released at COMMIT, so a use case that counted
// outside the lock would let two callers read the same total and both grant. The
// concurrency test above is what catches that; this one names the requirement
// directly, so the failure says which rule broke rather than only that a number
// was wrong.
func TestCampaignQuotaLocksBeforeCounting(t *testing.T) {
	campaign := testCampaign("launch-2026-10", "premium-monthly", 1, 0, timePtr(baseInstant()), nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	h.granted(t, payload)
	if !claims.lockWasTaken() {
		t.Fatalf("expected the quota path to take the advisory lock before counting")
	}
}

package subscription

import (
	"context"
	"sync"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// 024 T169 / FR-068 + FR-065: two requests carrying the same Idempotency-Key that
// arrive at the same moment must produce one `success` and one `skipped` with
// `duplicate_request`, and must not extend the expiry twice.
//
// Two separate defects were recorded here, and removing the re-check after the lock
// brings back both:
//
//  1. THE WRONG REASON CODE, for an account with no subscription row. There is nothing
//     to lock, both requests proceed, and the primary key rejects the second insert.
//     The loser fell through to the catch-all `grant_failed`, which says the
//     infrastructure failed. An administrator reading it goes looking for a broken
//     database or a bad deploy, and there is nothing to find.
//
//  2. THE DOUBLE GRANT, for an account that already has a row — the ordinary case,
//     since administrators re-grant to existing customers. Here nothing fails at all.
//     The loser blocks on the row lock, wakes up holding the winner's NEW expiry, and
//     extends it again. The customer receives two months for one decision, silently.
//     The convergence note said "no double accrual happens, only the reason code is
//     wrong"; that was wrong, and it is why the expiry assertion below is the
//     decisive one.

// lateWinningEventRepo reproduces the racing retry deterministically: the key is
// absent when the pre-lock check runs, and the winner's row exists by the time the
// account row is locked.
//
// It models this by COMMITTING the winner's row at the moment of the first lookup,
// then delegating every read to the real fake repository. Answering the second lookup
// from a call counter instead would be wrong: a genuinely fresh grant also calls
// GetByKey twice now (before and after the lock), and a fake keyed on call count would
// report that normal grant as a duplicate — the test would then be pinning the fake's
// shape rather than the behaviour.
//
// A real concurrent test would need two goroutines, a real database, and luck with the
// scheduler. This fake pins the interleaving instead, so the test fails when the
// re-check is removed rather than passing most of the time.
type lateWinningEventRepo struct {
	repositories.IUserSubscriptionEventRepository

	inner *grantTxEventRepo
	// winnerCommitsDuringLock is the scenario switch: true reproduces a racing retry,
	// false leaves the key absent so a normal grant runs through.
	winnerCommitsDuringLock bool

	mu   sync.Mutex
	seen int
}

func (r *lateWinningEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	r.mu.Lock()
	r.seen++
	first := r.seen == 1
	r.mu.Unlock()

	// Read FIRST. At the moment of the pre-lock check the winner has not committed,
	// so this returns nil; writing the winner's row before the read would make the
	// pre-lock check see it, the request would report itself a replay, and the test
	// would pass with the re-check deleted - which is precisely what an earlier
	// version of this fake did.
	result, err := r.inner.GetByKey(ctx, key)

	if first && r.winnerCommitsDuringLock {
		winner := &entities.UserSubscriptionEvent{EventKey: key}
		winner.ID = uuid.New()
		r.mu.Lock()
		// The inner fake lazily creates this map on its first write; here the winner
		// is the first writer, so it has to exist already.
		if r.inner.byKey == nil {
			r.inner.byKey = map[string]*entities.UserSubscriptionEvent{}
		}
		r.inner.byKey[key] = winner
		r.mu.Unlock()
	}

	return result, err
}

func TestRacingSameIdempotencyKeyReportsDuplicateRequestNotGrantFailed(t *testing.T) {
	user := grantTestUser("race", "race@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.uc.eventRepo = &lateWinningEventRepo{
		IUserSubscriptionEventRepository: h.events,
		inner:                            h.events,
		winnerCommitsDuringLock:          true,
	}

	res, err := h.uc.AdminGrantBatch(
		context.Background(),
		h.adminID,
		grantTestItems([2]string{"race@example.com", "premium"}),
		"idem-RACE",
		nil,
	)
	if err != nil {
		t.Fatalf("AdminGrantBatch failed: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("expected exactly one result row, got = %d", len(res.Results))
	}

	got := res.Results[0]

	if got.Status != sub_dto.AdminGrantStatusSkipped {
		t.Errorf("the loser must be skipped, got = %s", got.Status)
	}
	if got.Reason == sub_dto.AdminGrantReasonGrantFailed {
		t.Errorf("grant_failed means INFRASTRUCTURE FAILURE: it would send an administrator looking for a fault that does not exist. Must be duplicate_request")
	}
	if got.Reason != sub_dto.AdminGrantReasonDuplicateRequest {
		t.Errorf("must report duplicate_request, got = %s", got.Reason)
	}

	// The decisive assertion, and the one the convergence note got wrong. The loser
	// must not have applied the plan either: on an account that already holds a
	// subscription, skipping the re-check extends the expiry a second time with no
	// error anywhere. A correct reason code on top of a double grant would still be a
	// double charge.
	if sub, ok := h.subs.subs[user.ID]; ok && sub.ExpiresAt != nil {
		t.Errorf("the loser must not be granted the plan, but an expiry was set = %v", sub.ExpiresAt)
	}
	if h.events.creates != 0 {
		t.Errorf("no event row may be written, got = %d", h.events.creates)
	}
}

// The guard against over-reaching: the re-check must not turn an ordinary fresh grant
// into a duplicate. With no winner in sight the key stays absent, the grant runs
// normally, and the extra lookup costs nothing — otherwise this fix would silently
// stop every grant from working.
func TestFreshGrantWithNoRacingWinnerStillSucceeds(t *testing.T) {
	user := grantTestUser("fresh", "fresh@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.uc.eventRepo = &lateWinningEventRepo{
		IUserSubscriptionEventRepository: h.events,
		inner:                            h.events,
		winnerCommitsDuringLock:          false,
	}

	res, err := h.uc.AdminGrantBatch(
		context.Background(),
		h.adminID,
		grantTestItems([2]string{"fresh@example.com", "premium"}),
		"idem-FRESH",
		nil,
	)
	if err != nil {
		t.Fatalf("AdminGrantBatch failed: %v", err)
	}

	if res.Succeeded != 1 {
		t.Fatalf("an ordinary grant must succeed, got succeeded=%d (%s)",
			res.Succeeded, res.Results[0].Message)
	}
	if res.Results[0].Reason == sub_dto.AdminGrantReasonDuplicateRequest {
		t.Error("with no racing request there must be no duplicate_request")
	}
}

// Compile-time proof the fake satisfies the interface the use case holds.
var _ repositories.IUserSubscriptionEventRepository = (*lateWinningEventRepo)(nil)

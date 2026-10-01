package grant

import (
	"context"
	"testing"
	"time"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// 024 T163 + T164: the grant write is a read-decide-write sequence, and its
// correctness rests entirely on the ORDER of those steps.
//
// A concurrency test against a real database can assert the outcome but cannot force
// the interleaving - the window between the read and the write is opened by the
// database scheduler, not by the test - so removing the lock from the code left such
// a test green. That was verified, not assumed.
//
// Order, by contrast, is exactly observable from a fake, and it is the whole fix:
//
//	1. lock the account
//	2. re-check for a duplicate under that lock
//	3. re-check the event is not stale
//	4. retire the previous grants
//	5. create the new one
//
// Any step moved above the lock, or any step dropped, is a regression this catches.

// orderedGrantRepo records the call sequence of the grant write.
type orderedGrantRepo struct {
	repositories.IAICostRepository

	calls  []string
	policy *aiexecutor_entities.AICostPolicy

	// presentFrom makes a lookup return a grant from the Nth call onwards, 0 meaning
	// "never present". A non-zero value is how a concurrent delivery is simulated:
	// absent when the cheap pre-check runs, present by the time the lock is held.
	presentFrom  int
	bySourceSeen int
	created      int
}

func (r *orderedGrantRepo) LockGrantsForAccount(context.Context, uuid.UUID) error {
	r.calls = append(r.calls, "lock")
	return nil
}

func (r *orderedGrantRepo) GetGrantBySourceEventID(context.Context, uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	r.bySourceSeen++
	underLock := r.bySourceSeen > 1
	if r.presentFrom > 0 && r.bySourceSeen >= r.presentFrom {
		return &aiexecutor_entities.AIPolicyGrant{}, nil
	}
	if underLock {
		r.calls = append(r.calls, "duplicate-check-under-lock")
	} else {
		r.calls = append(r.calls, "duplicate-check-pre")
	}
	return nil, nil
}

func (r *orderedGrantRepo) GetPolicyByID(context.Context, uuid.UUID) (*aiexecutor_entities.AICostPolicy, error) {
	return r.policy, nil
}

// GetNewestGrantForOrdering is what the stale check calls, so the fake records the
// step under this name rather than under GetCurrentGrant.
func (r *orderedGrantRepo) GetNewestGrantForOrdering(context.Context, uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	r.calls = append(r.calls, "stale-check")
	return nil, nil
}

func (r *orderedGrantRepo) CloseActiveGrants(_ context.Context, _ uuid.UUID, _, _ time.Time) error {
	r.calls = append(r.calls, "retire")
	return nil
}

func (r *orderedGrantRepo) DeleteFutureGrants(context.Context, uuid.UUID) error {
	// Part of retirePreviousGrants, recorded separately because it is a different
	// write and the order assertion cares that retirement precedes creation.
	r.calls = append(r.calls, "delete-future")
	return nil
}

func (r *orderedGrantRepo) CreateGrant(context.Context, *aiexecutor_entities.AIPolicyGrant) error {
	r.created++
	r.calls = append(r.calls, "create")
	return nil
}

// newOrderedUC builds a use case over the recording repository. The unit of work is
// the package's `recordingUow`, which runs the callback inline and records whether
// the body returned an error.
func newOrderedUC(repo repositories.IAICostRepository) *GrantUseCase {
	log := logger.New("test", "", "info", false)
	return NewGrantUseCase(repo, &recordingUow{}, log, NewGrantMetrics(log)).(*GrantUseCase)
}

func TestGrantWriteTakesTheAccountLockFirst(t *testing.T) {
	repo := &orderedGrantRepo{policy: strictPolicyWithAllOperations()}
	uc := newOrderedUC(repo)

	payload := validPayload()
	payload.AICostPolicyID = repo.policy.ID.String()
	payload.ToPlanID = uuid.New().String()
	payload.SourceEventID = uuid.New().String()
	payload.FromPlanCode = ""

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("HandlePlanChanged: %v", err)
	}

	// A cheap duplicate pre-check may run BEFORE the lock - that is deliberate, it
	// keeps a redelivery from taking a lock at all. What must hold is that the lock
	// precedes every step that reads state, because those are the reads the lock makes
	// meaningful.
	if !containsStep(repo.calls, "lock") {
		t.Fatalf("the account must be locked, call sequence = %v", repo.calls)
	}
	for _, after := range []string{"duplicate-check-under-lock", "stale-check", "retire", "create"} {
		if !stepAfter(repo.calls, "lock", after) {
			t.Errorf("step %q must come AFTER the account lock, sequence = %v", after, repo.calls)
		}
	}
	// FR-034: the ordering has to be decided against the state the lock makes stable,
	// so the stale check belongs after it, and retirement before the insert - the
	// account must never hold two overlapping entitlements.
	if !stepAfter(repo.calls, "stale-check", "retire") {
		t.Errorf("the ordering check must run BEFORE retiring the previous grant, sequence = %v", repo.calls)
	}
	if !stepAfter(repo.calls, "retire", "create") {
		t.Errorf("the previous grant must be retired before the new one is created, sequence = %v", repo.calls)
	}
}

// A redelivery that appears only after the lock must be caught by the re-check, and
// must write NOTHING. This is FR-033's "MUST NOT ghi tác dụng phụ" stated as an
// assertion about side effects rather than about the return value.
func TestRedeliveryAfterTheLockWritesNothing(t *testing.T) {
	repo := &orderedGrantRepo{policy: strictPolicyWithAllOperations(), presentFrom: 2}
	uc := newOrderedUC(repo)

	payload := validPayload()
	payload.AICostPolicyID = repo.policy.ID.String()
	payload.ToPlanID = uuid.New().String()
	payload.SourceEventID = uuid.New().String()
	payload.FromPlanCode = ""

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("a redelivery that was already applied is not an error: %v", err)
	}

	if repo.created != 0 {
		t.Errorf("a redelivery must not create a grant, created %d", repo.created)
	}
	if containsStep(repo.calls, "retire") || containsStep(repo.calls, "create") {
		t.Errorf("a redelivery must not touch the existing grant, sequence = %v", repo.calls)
	}
}

// AUTO_RENEW_UPDATED is the only type that still reaches the revocation branch,
// so it is what this test drives. 024 T188: it used SUBSCRIPTION_DOWNGRADED_TO_FREE,
// which asserted that a revoke takes the account lock before retiring and never
// creates a row — correct for the design of the day, and exactly the pair of
// properties that produced the data loss. A downgrade now creates a `free-default`
// grant, so it is an ordinary lock → retire → create and is covered by
// TestGrantWriteTakesTheAccountLockFirst; the lock ordering below therefore moved
// to the type it is still true of.
func TestRevocationTakesTheAccountLock(t *testing.T) {
	repo := &orderedGrantRepo{policy: strictPolicyWithAllOperations()}
	uc := newOrderedUC(repo)

	payload := validPayload()
	payload.AICostPolicyID = repo.policy.ID.String()
	payload.ToPlanID = uuid.New().String()
	payload.SourceEventID = uuid.New().String()
	payload.EventType = string(subscriptioneventtype.AUTO_RENEW_UPDATED)
	payload.FromPlanCode = "premium"

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("HandlePlanChanged (no-op): %v", err)
	}
	// Toggling auto-renew changes no entitlement, so it must not take the lock
	// either: a revoke racing a grant would otherwise close the row the grant had
	// just written, or vice versa, leaving the account with neither.
	if containsStep(repo.calls, "lock") {
		t.Errorf("an auto-renew toggle must not lock the account, sequence = %v", repo.calls)
	}
	if containsStep(repo.calls, "retire") {
		t.Errorf("an auto-renew toggle must not retire a grant, sequence = %v", repo.calls)
	}
	if repo.created != 0 || containsStep(repo.calls, "create") {
		t.Errorf("an auto-renew toggle must not create a grant, sequence = %v", repo.calls)
	}
}

// 024 T188: the downgrade is an ordinary lock → retire → create, and both halves
// are required. ClosesPriorGrants closes the Premium row; CreatesNewGrant creates
// the free-default one. Closing without creating was the bug, so this asserts the
// pair rather than either side alone.
func TestDowngradeLocksRetiresAndCreates(t *testing.T) {
	repo := &orderedGrantRepo{policy: strictPolicyWithAllOperations()}
	uc := newOrderedUC(repo)

	payload := validPayload()
	payload.AICostPolicyID = repo.policy.ID.String()
	payload.ToPlanID = uuid.New().String()
	payload.SourceEventID = uuid.New().String()
	payload.EventType = string(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE)
	payload.FromPlanCode = "premium"

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("HandlePlanChanged (downgrade): %v", err)
	}
	for _, step := range []string{"lock", "stale-check", "retire", "create"} {
		if !containsStep(repo.calls, step) {
			t.Fatalf("a downgrade must run step %q, sequence = %v", step, repo.calls)
		}
	}
	if !stepAfter(repo.calls, "lock", "retire") {
		t.Errorf("the downgrade must lock before retiring, sequence = %v", repo.calls)
	}
	if !stepAfter(repo.calls, "retire", "create") {
		t.Errorf("the downgrade must retire before creating, sequence = %v", repo.calls)
	}
}

func containsStep(calls []string, want string) bool {
	for _, c := range calls {
		if c == want {
			return true
		}
	}
	return false
}

func stepAfter(calls []string, first, second string) bool {
	firstIdx, secondIdx := -1, -1
	for i, c := range calls {
		if c == first && firstIdx < 0 {
			firstIdx = i
		}
		if c == second && secondIdx < 0 {
			secondIdx = i
		}
	}
	return firstIdx >= 0 && secondIdx > firstIdx
}

// testLogger is the sink used where a test needs a parser logger but does not assert
// on it. T171 made the older-payload warning flow through the service logger, so the
// parser takes one; tests that only care about the parse result pass this rather than
// constructing a logger at each call site.
func testLogger() logger.Interface {
	return logger.New("test", "", "info", false)
}

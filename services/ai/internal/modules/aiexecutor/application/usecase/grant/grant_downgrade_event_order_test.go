package grant

import (
	"context"
	"errors"
	"testing"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// ---- 024 T210: the unpriceable-downgrade path orders events the same way ----
//
// `retireOnUnpriceableDowngrade` (024 T204) opens a transaction, takes the
// per-account lock and retires the previous grant. It did NOT call
// `rejectStaleEvent`, while the priced path does - under the same lock, before it
// retires anything (`grant_uc.go:266-271`).
//
// The inconsistency is narrow but it is a contradiction on identical input: a
// downgrade that arrived late and cannot be priced would still take the Premium
// entitlement away, whereas the very same event, priced, would be refused as
// `stale_event`. Whether the customer loses access would depend on whether the
// free plan's `ai_cost_policy_id` resolves - a property of the plan catalogue,
// not of the event's ordering. One event, two conclusions, decided by an
// unrelated table.
//
// These two tests pin the rule from both sides. The first refuses the late event
// and writes nothing at all; the second proves the fix did not break the case
// T204 exists for, where a downgrade that IS current must still retire the tier
// the customer has given up.

// downgradePayload is `validPayload()` carrying the event type T204 scoped
// itself to.
func downgradePayload() aiexecutor_dto.PlanChangedEventPayload {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE)
	return payload
}

// grantWrittenAt builds the account's newest grant as the ordering check sees it
// when the policy snapshot is unreadable. The three-level floor then prefers
// created_at: 027 falls back to BaseEntity.CreatedAt rather than refusing, so that
// column is what makes the event stale or current here. The snapshot is left empty
// on purpose - it is the shape that used to lock this path up, and rejectStaleEvent
// must no longer care whether it is readable (FR-001, FR-019).
func grantWrittenAt(t *testing.T, createdAt time.Time) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	return &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New(), CreatedAt: createdAt},
		},
	}
}

// TestStaleDowngradeWithUnpriceablePolicyClosesNothing is the hole T210 closes.
//
// The account's newest grant is newer than the event being delivered, so this is
// a late downgrade. The priced path would refuse it as `stale_event` before it
// retired anything, and the priced path has to reach the same conclusion as the
// unpriceable one: the ordering of the event is a property of the event, and
// nothing about a missing cost policy may change the answer.
func TestStaleDowngradeWithUnpriceablePolicyClosesNothing(t *testing.T) {
	payload := downgradePayload()
	repo := &fakeGrantRepo{
		policy: nil,
		// One hour AFTER the event: the stored state already knows about a later
		// change, so this delivery is stale.
		currentGrant: grantWrittenAt(t, windowBase.Add(time.Hour)),
	}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	err := uc.HandlePlanChanged(context.Background(), payload)

	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("a stale downgrade must be refused, got %T: %v", err, err)
	}
	if rejection.Reason != ReasonStaleEvent {
		t.Fatalf("expected %q, got %q", ReasonStaleEvent, rejection.Reason)
	}
	if repo.closedCalls != 0 {
		t.Errorf("a stale downgrade must not close any grant, got %d closes", repo.closedCalls)
	}
	if repo.deletedCalls != 0 {
		t.Errorf("a stale downgrade must not delete any pending grant, got %d deletes", repo.deletedCalls)
	}
	if len(repo.createdGrants) != 0 {
		t.Errorf("a stale downgrade must not write a grant, got %d", len(repo.createdGrants))
	}
	if !uow.rolledBack {
		t.Error("the transaction must roll back, otherwise a write that raced the refusal could still land")
	}
}

// TestCurrentDowngradeWithUnpriceablePolicyStillRetires is the other side, and
// the reason the fix had to be narrow.
//
// A downgrade that is NOT stale is exactly what 024 T204 is about: the customer
// has given up the tier, the new one cannot be priced, and holding on to the old
// entitlement is its own defect. The added ordering check must not swallow this,
// and it must not turn the refusal into a retryable one - a refusal that is
// requeued spins on an event that can never be applied.
func TestCurrentDowngradeWithUnpriceablePolicyStillRetires(t *testing.T) {
	payload := downgradePayload()
	repo := &fakeGrantRepo{
		policy: nil,
		// One hour BEFORE the event: the delivered downgrade is the newest thing
		// that happened to this account.
		currentGrant: grantWrittenAt(t, windowBase.Add(-time.Hour)),
	}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	err := uc.HandlePlanChanged(context.Background(), payload)

	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("an unpriceable policy must still be refused, got %T: %v", err, err)
	}
	if rejection.Reason != ReasonAICostPolicyNotFound {
		t.Fatalf("expected %q, got %q", ReasonAICostPolicyNotFound, rejection.Reason)
	}
	if repo.closedCalls != 1 {
		t.Errorf("a current downgrade must still close the Premium grant, got %d closes", repo.closedCalls)
	}
	if repo.deletedCalls != 1 {
		t.Errorf("a current downgrade must still drop a pending grant, got %d deletes", repo.deletedCalls)
	}
	if uow.rolledBack {
		t.Error("the retirement must commit")
	}
}

// TestDowngradeEventTypeStillClosesPriorGrants pins `ClosesPriorGrants` for the
// type T210 touches. The added ordering check must not have changed what the
// downgrade retires: it still closes the Premium row, which is why
// `TestCurrentDowngradeWithUnpriceablePolicyStillRetires` above can assert one
// close at all.
func TestDowngradeEventTypeStillClosesPriorGrants(t *testing.T) {
	if !ClosesPriorGrants(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE) {
		t.Error("a downgrade must still close the grants it replaces")
	}
}

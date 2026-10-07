package grant

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// The project has no database test harness, so the handler is driven through
// hand-written fakes. That is sufficient here because the properties under test
// are about WHICH writes happen and in what order, not about SQL behaviour.
// ---------------------------------------------------------------------------

type fakeGrantRepo struct {
	repositories.IAICostRepository

	policy        *aiexecutor_entities.AICostPolicy
	existingGrant *aiexecutor_entities.AIPolicyGrant
	currentGrant  *aiexecutor_entities.AIPolicyGrant

	createdGrants []*aiexecutor_entities.AIPolicyGrant
	closedCalls   int
	deletedCalls  int
	truncateCalls int

	lookupErr  error
	closeErr   error
	createErr  error
	currentErr error
	// 024 T152: the duplicate check used to have no way to fail, which hid the
	// fact that its error was returned unclassified and dead-lettered.
	bySourceErr error
}

// 024 T163 + T164: the write path takes a per-account advisory lock inside the
// transaction. The fake has to answer it, otherwise any test that reaches the
// transaction panics on the embedded nil interface before reaching the behaviour it
// was written to check.
func (f *fakeGrantRepo) LockGrantsForAccount(context.Context, uuid.UUID) error { return nil }

func (f *fakeGrantRepo) GetGrantBySourceEventID(context.Context, uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	if f.bySourceErr != nil {
		return nil, f.bySourceErr
	}
	return f.existingGrant, nil
}

func (f *fakeGrantRepo) GetPolicyByID(context.Context, uuid.UUID) (*aiexecutor_entities.AICostPolicy, error) {
	return f.policy, f.lookupErr
}

func (f *fakeGrantRepo) GetCurrentGrant(context.Context, uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	return f.currentGrant, f.currentErr
}

// 024 T164: the ordering check reads the newest grant, so the fake has to answer the
// method it actually calls. The window a fake row "has" is irrelevant here — the
// ordering rule is exactly that a future row must be visible, which the integration
// test proves against a real database.
func (f *fakeGrantRepo) GetNewestGrantForOrdering(context.Context, uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error) {
	return f.currentGrant, f.currentErr
}

func (f *fakeGrantRepo) CreateGrant(_ context.Context, grant *aiexecutor_entities.AIPolicyGrant) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createdGrants = append(f.createdGrants, grant)
	return nil
}

func (f *fakeGrantRepo) CloseActiveGrants(context.Context, uuid.UUID, time.Time, time.Time) error {
	f.closedCalls++
	return f.closeErr
}

func (f *fakeGrantRepo) DeleteFutureGrants(context.Context, uuid.UUID) error {
	f.deletedCalls++
	return nil
}

func (f *fakeGrantRepo) TruncateGrantsBefore(context.Context, uuid.UUID, time.Time, time.Time) error {
	f.truncateCalls++
	return nil
}

type recordingUow struct {
	shared_repos.IUnitOfWork
	// rolledBack records that the transaction body returned an error, which is
	// what GORM does with everything the body wrote.
	rolledBack bool
	bodies     int
}

func (r *recordingUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	r.bodies++
	if err := fn(ctx); err != nil {
		r.rolledBack = true
		return err
	}
	return nil
}

func newHandler(repo repositories.IAICostRepository, uow shared_repos.IUnitOfWork) *GrantUseCase {
	log := logger.New("test", "", "info", false)
	return NewGrantUseCase(repo, uow, log, NewGrantMetrics(log)).(*GrantUseCase)
}

func strictPolicyWithAllOperations() *aiexecutor_entities.AICostPolicy {
	policy := &aiexecutor_entities.AICostPolicy{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.MustParse(validPolicyID)},
		},
		Code:             "premium-default",
		EnforcementMode:  aienforcementmode.Strict,
		PeriodDays:       30,
		HardCostMicroVND: int64Ptr(30000000000),
		IsActive:         true,
	}
	for _, op := range []aioperation.AIOperation{aioperation.Chat, aioperation.Outfit, aioperation.Summary, aioperation.Rewriter} {
		policy.Operations = append(policy.Operations, &aiexecutor_entities.AICostPolicyOperation{
			PolicyID:              policy.ID,
			Operation:             op,
			NormalRoute:           airoute.Paid,
			MaxPaidAttemptsPerDay: nil,
			IsEnabled:             true,
		})
	}
	return policy
}

func int64Ptr(v int64) *int64 { return &v }

// TestDuplicateEventIsSkippedWithoutSideEffects is FR-033. The outbox
// guarantees at-least-once delivery, so a redelivery is routine; treating it as
// a failure would burn the delivery limit on a message already handled.
func TestDuplicateEventIsSkippedWithoutSideEffects(t *testing.T) {
	repo := &fakeGrantRepo{
		policy: strictPolicyWithAllOperations(),
		existingGrant: &aiexecutor_entities.AIPolicyGrant{
			AuditableEntity: aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		},
	}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	if err := uc.HandlePlanChanged(context.Background(), validPayload()); err != nil {
		t.Fatalf("a duplicate must not be an error: %v", err)
	}
	if len(repo.createdGrants) != 0 {
		t.Fatal("a duplicate must not create a second grant")
	}
	if uow.bodies != 0 {
		t.Fatal("a duplicate must not open a transaction")
	}
}

// TestCloseAndCreateShareOneTransaction is FR-031 and the most important
// property in this file. Two independent writes leave a window in which the
// account holds no entitlement at all — a user who just paid, refused by the AI
// service.
func TestCloseAndCreateShareOneTransaction(t *testing.T) {
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	if err := uc.HandlePlanChanged(context.Background(), validPayload()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uow.bodies != 1 {
		t.Fatalf("expected exactly one transaction, got %d", uow.bodies)
	}
	if uow.rolledBack {
		t.Fatal("the transaction must commit")
	}
	if repo.closedCalls != 1 {
		t.Fatalf("the previous grant must be closed once, got %d", repo.closedCalls)
	}
	if len(repo.createdGrants) != 1 {
		t.Fatalf("expected one new grant, got %d", len(repo.createdGrants))
	}
}

// TestFailedCloseRollsBackTheWholeChange: a partial application is worse than
// none, because the account would keep a stale entitlement.
func TestFailedCloseRollsBackTheWholeChange(t *testing.T) {
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations(), closeErr: errors.New("connection reset")}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	err := uc.HandlePlanChanged(context.Background(), validPayload())
	if err == nil {
		t.Fatal("a failed close must surface as an error")
	}
	if !uow.rolledBack {
		t.Fatal("the transaction must roll back")
	}
	if len(repo.createdGrants) != 0 {
		t.Fatal("no grant may be created when the close failed")
	}
}

// TestAutoRenewToggleDoesNothing guards the exact bug this feature exists to
// remove: the old receiver treated AUTO_RENEW_UPDATED as a grant-creating
// change, so toggling auto-renew revoked and re-granted entitlements.
func TestAutoRenewToggleDoesNothing(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.AUTO_RENEW_UPDATED)
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.createdGrants) != 0 {
		t.Fatal("toggling auto-renew must not create a grant")
	}
	if repo.closedCalls != 0 {
		t.Fatal("toggling auto-renew must not close any grant")
	}
}

// TestDowngradeClosesTheOldGrantAndCreatesTheFreeOne is 024 T188.
//
// It used to assert the opposite — that a downgrade closes and creates nothing —
// because the free plan's quota was believed to be applied by the fallback
// resolver. A grant row is the single source of truth for an entitlement, paid or
// free, so the downgrade has to close the Premium row AND create the free-default
// one in the same transaction. Closing without creating left an account that had a
// plan and nothing to resolve against: check 1 of 07-verify_policy_grants.sql
// reported it on the dev stack until an operator ran the repair endpoint.
func TestDowngradeClosesTheOldGrantAndCreatesTheFreeOne(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE)
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.closedCalls != 1 {
		t.Fatalf("a downgrade must close the active Premium grant, got %d closes", repo.closedCalls)
	}
	if len(repo.createdGrants) != 1 {
		t.Fatalf("a downgrade must create the free-default grant, got %d", len(repo.createdGrants))
	}
	// The new row is a live entitlement, not a placeholder: it carries the free
	// plan's identity and starts at the moment of the change, so it is already
	// in force when the customer next calls the AI service.
	created := repo.createdGrants[0]
	if created.PlanCode != payload.ToPlanCode {
		t.Fatalf("the new grant must carry the target plan code, got %q", created.PlanCode)
	}
	if created.Status != aipolicygrantstatus.Active {
		t.Fatalf("the free grant must be active, got %q", created.Status)
	}
	if !created.EffectiveFrom.Equal(windowBase) {
		t.Fatalf("the free grant must start at the time of the change, got %v", created.EffectiveFrom)
	}
	if len(created.PolicySnapshot) == 0 {
		t.Fatal("the free grant must carry a policy snapshot, otherwise its limits are not applied")
	}
	// Closing the old row and creating the new one is one transaction: a crash
	// between them would leave the customer with no entitlement at all.
	if uow.bodies != 1 || uow.rolledBack {
		t.Fatalf("the close and the create must share one committed transaction, bodies=%d rolledBack=%v",
			uow.bodies, uow.rolledBack)
	}
}

// TestAutoRenewToggleIsStillANoOp is 024 T188's guard on the other side of the
// change: only the downgrade flipped, so an auto-renew toggle must still touch
// nothing at all. `TestAutoRenewToggleDoesNothing` above already asserts the two
// properties that matter — no row created, no row closed — so this one covers the
// remaining gap: it must not even open a transaction.
func TestAutoRenewToggleIsStillANoOp(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.AUTO_RENEW_UPDATED)
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.createdGrants) != 0 {
		t.Fatal("an auto-renew toggle must not create a grant")
	}
	if repo.closedCalls != 0 {
		t.Fatal("an auto-renew toggle must not close a grant")
	}
	if uow.bodies != 0 {
		t.Fatalf("an auto-renew toggle must not open a transaction, got %d", uow.bodies)
	}
}

// TestRenewalTruncatesInsteadOfClosing: a renewal buys one continuous stretch,
// so the prior grant is truncated at the boundary. Closing it would leave a gap.
func TestRenewalTruncatesInsteadOfClosing(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.EXTENDED)
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uc := newHandler(repo, &recordingUow{})

	if err := uc.HandlePlanChanged(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.truncateCalls != 1 {
		t.Fatalf("a renewal must truncate the prior grant, got %d truncations", repo.truncateCalls)
	}
	if repo.closedCalls != 0 {
		t.Fatal("a renewal must not close the prior grant")
	}
	if repo.deletedCalls != 0 {
		t.Fatal("a renewal must not delete a pending grant — the renewal may be what it was waiting for")
	}
}

// TestUnknownPolicyIsRefusedLoudly: a plan pointing at a missing policy is a
// configuration fault, and guessing one would hand out unpriced entitlements.
func TestUnknownPolicyIsRefusedLoudly(t *testing.T) {
	repo := &fakeGrantRepo{policy: nil}
	uc := newHandler(repo, &recordingUow{})

	err := uc.HandlePlanChanged(context.Background(), validPayload())
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("expected a RejectionError, got %T: %v", err, err)
	}
	if rejection.Reason != ReasonAICostPolicyNotFound {
		t.Fatalf("expected %q, got %q", ReasonAICostPolicyNotFound, rejection.Reason)
	}
	if len(repo.createdGrants) != 0 {
		t.Fatal("nothing may be written without a policy")
	}
}

// TestDowngradeWithMissingPolicyStillRetiresThePreviousGrant is 024 T204.
//
// A downgrade used to be routed to `applyRevocation`, which never read a policy,
// so nothing in the policy catalogue could stop it from retiring the Premium
// grant. T188 moved the downgrade onto the create path, and the policy lookup
// sits in FRONT of `retirePreviousGrants` there. The consequence is that a free
// plan whose `ai_cost_policy_id` cannot be resolved turns the refusal permanent
// AND leaves the customer holding the paid entitlement they just gave up.
//
// The refusal itself has to survive: no grant may be written without a policy to
// price it from, the reason must be named (FR-085) and recorded (FR-086). What
// must not survive is the tier the customer already left.
func TestDowngradeWithMissingPolicyStillRetiresThePreviousGrant(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE)
	repo := &fakeGrantRepo{policy: nil}
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
	if len(repo.createdGrants) != 0 {
		t.Fatal("nothing may be written without a policy to price it from")
	}
	if repo.closedCalls != 1 {
		t.Fatalf("a downgrade must still close the Premium grant when the policy is unreadable, got %d closes", repo.closedCalls)
	}
	if repo.deletedCalls != 1 {
		t.Fatalf("a downgrade must still drop a pending grant when the policy is unreadable, got %d deletes", repo.deletedCalls)
	}
	if uow.rolledBack {
		t.Fatal("the retirement must commit")
	}
}

// TestUnpriceableUpgradeKeepsTheCurrentGrant is the other side of T204, and the
// reason the fix is scoped to one event type rather than to every refusal.
//
// An upgrade that cannot be priced has cost the customer nothing: they still hold
// the tier they had. Closing that row would take away an entitlement they are
// still entitled to and leave the fallback, whose limits are lower. So the
// retirement is for downgrades only, and this test is what keeps it that way.
func TestUnpriceableUpgradeKeepsTheCurrentGrant(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.UPGRADED)
	repo := &fakeGrantRepo{policy: nil}
	uow := &recordingUow{}
	uc := newHandler(repo, uow)

	err := uc.HandlePlanChanged(context.Background(), payload)
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("an unpriceable policy must still be refused, got %T: %v", err, err)
	}
	if repo.closedCalls != 0 {
		t.Fatalf("an unpriceable upgrade must not take away the tier the customer still holds, got %d closes", repo.closedCalls)
	}
	if repo.deletedCalls != 0 {
		t.Fatalf("an unpriceable upgrade must not touch pending rows, got %d deletes", repo.deletedCalls)
	}
	if uow.bodies != 0 {
		t.Fatalf("an unpriceable upgrade must not open a transaction, got %d", uow.bodies)
	}
}

// TestStaleEventIsRefused stops an out-of-order delivery from resurrecting a
// revoked entitlement.
//
// 027: the ordering floor is the prior row's OWN created_at, not occurred_at from
// the policy snapshot. So the stored record is stamped through BaseEntity.CreatedAt
// and the snapshot is deliberately left empty - the very shape that used to make
// this check refuse permanently.
func TestStaleEventIsRefused(t *testing.T) {
	storedAt := windowBase.Add(time.Hour)
	prior := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New(), CreatedAt: storedAt},
		},
	}

	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations(), currentGrant: prior}
	uc := newHandler(repo, &recordingUow{})

	rejectionErr := uc.HandlePlanChanged(context.Background(), validPayload())
	var rejection *RejectionError
	if !errors.As(rejectionErr, &rejection) {
		t.Fatalf("expected a RejectionError, got %T: %v", rejectionErr, rejectionErr)
	}
	if rejection.Reason != ReasonStaleEvent {
		t.Fatalf("expected %q, got %q", ReasonStaleEvent, rejection.Reason)
	}
	if len(repo.createdGrants) != 0 {
		t.Fatal("a stale event must not write a grant")
	}
}

func TestNewerEventIsAccepted(t *testing.T) {
	prior := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New(), CreatedAt: windowBase.Add(-time.Hour)},
		},
	}

	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations(), currentGrant: prior}
	uc := newHandler(repo, &recordingUow{})

	if err := uc.HandlePlanChanged(context.Background(), validPayload()); err != nil {
		t.Fatalf("a genuinely newer event must be accepted: %v", err)
	}
	if len(repo.createdGrants) != 1 {
		t.Fatal("the new grant must be created")
	}
}

// TestGrantCarriesTheFullPlanIdentity: plan_id is NOT NULL and its foreign key
// was dropped, so nothing downstream can recover a wrong value.
func TestGrantCarriesTheFullPlanIdentity(t *testing.T) {
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uc := newHandler(repo, &recordingUow{})

	if err := uc.HandlePlanChanged(context.Background(), validPayload()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	grant := repo.createdGrants[0]
	if grant.PlanID != uuid.MustParse(validPlanID) {
		t.Fatalf("plan id was not carried: %s", grant.PlanID)
	}
	if grant.PolicyID != uuid.MustParse(validPolicyID) {
		t.Fatalf("policy id was not carried: %s", grant.PolicyID)
	}
	if grant.SourceEventID == nil || *grant.SourceEventID != uuid.MustParse(validEventID) {
		t.Fatalf("the duplicate key was not stored: %+v", grant.SourceEventID)
	}
	if grant.Status == "" {
		t.Fatal("the status is NOT NULL and must be set")
	}
}

// TestSnapshotCarriesAllFourOperations is SC-009. A snapshot missing an
// operation cannot satisfy the entitlement read, so the grant would be written
// already broken.
func TestSnapshotCarriesAllFourOperations(t *testing.T) {
	repo := &fakeGrantRepo{policy: strictPolicyWithAllOperations()}
	uc := newHandler(repo, &recordingUow{})

	if err := uc.HandlePlanChanged(context.Background(), validPayload()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw := repo.createdGrants[0].PolicySnapshot
	if len(raw) == 0 {
		t.Fatal("the snapshot must never be empty")
	}
	var decoded PolicySnapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("the snapshot must round-trip as JSON: %v", err)
	}
	for _, op := range entitlementOperations {
		if _, ok := decoded.Entitlements[string(op)]; !ok {
			t.Fatalf("the snapshot is missing operation %q: %s", op, raw)
		}
	}
	if decoded.Envelope.SourceEventID != validEventID {
		t.Fatalf("the envelope must carry the source event id, got %q", decoded.Envelope.SourceEventID)
	}
	if decoded.Envelope.EventVersion != 2 {
		t.Fatalf("the envelope must carry the contract version, got %d", decoded.Envelope.EventVersion)
	}
}

// TestRejectionCounterSeparatesCauses is FR-087: "an event was refused" and "an
// event was never sent" must not look the same on a dashboard.
func TestRejectionCounterSeparatesCauses(t *testing.T) {
	metrics := NewGrantMetrics(logger.New("test", "", "info", false))
	metrics.RecordRejection(ReasonStaleEvent, "22222222-2222-2222-2222-222222222222")
	metrics.RecordRejection(ReasonStaleEvent, "22222222-2222-2222-2222-222222222222")
	metrics.RecordRejection(ReasonAICostPolicyNotFound, "22222222-2222-2222-2222-222222222222")
	metrics.RecordRejection(ReasonDuplicateSourceEvent, "22222222-2222-2222-2222-222222222222")

	snapshot := metrics.Snapshot()
	if snapshot[ReasonStaleEvent] != 2 {
		t.Fatalf("stale count wrong: %+v", snapshot)
	}
	if snapshot[ReasonAICostPolicyNotFound] != 1 {
		t.Fatalf("catalog count wrong: %+v", snapshot)
	}
	if snapshot[ReasonDuplicateSourceEvent] != 1 {
		t.Fatalf("duplicate count wrong: %+v", snapshot)
	}
}

var _ = aiexecutor_dto.PlanChangedEventPayload{}

package subscription

import (
	"context"
	"errors"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// ---- 024 T196: provisioning happens EAGERLY, at account creation ----
//
// Default provisioning used to run only on the first read of a subscription,
// so SC-006 ("a valid grant within 60 seconds of creating the account, with no
// read in between") could not be met by the three account-creation doors. They
// now call `EnsureDefaultSubscription` inside the transaction that inserted the
// user row.
//
// That creates the one risk this file exists to rule out: two provisioning doors
// for one account. The lazy door must stay for accounts created any other way,
// and both doors run the same body, so running them in sequence has to produce
// ONE subscription row and ONE history row. Every assertion below is about that.

// eagerSubRepo models the parts of the persistence layer the guarantees rest on:
// `user_id` is the primary key and `ProvisionDefault` is `ON CONFLICT (user_id)
// DO NOTHING`, so a second insert attempt cannot produce a second row.
type eagerSubRepo struct {
	fakeGrantSubRepo
	// provisionCalls counts every ProvisionDefault attempt, INCLUDING the ones the
	// database refuses. That is the number a duplicate would show up in.
	provisionCalls int
}

func newEagerSubRepo() *eagerSubRepo {
	return &eagerSubRepo{
		fakeGrantSubRepo: fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{}},
	}
}

func (r *eagerSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	r.provisionCalls++
	// ON CONFLICT (user_id) DO NOTHING: the second caller keeps the existing row.
	if _, exists := r.subs[sub.UserID]; exists {
		return nil
	}
	// `Version` is stored exactly as the caller left it, because
	// `provisionDefaultOn` never sets it and `user_subscriptions.version` is
	// `bigint NOT NULL DEFAULT 0`. This fake used to invent `Version = 1` here,
	// which made the event-key assertions below pin a value production cannot
	// produce - and worse, it hid the fact that the key is version-independent in
	// practice: every provisioning of a given account computes `:0`, which is
	// precisely why the UNIQUE index on `event_key` deduplicates the retry.
	copied := *sub
	r.subs[sub.UserID] = &copied
	return nil
}

func (r *eagerSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return r.subs[userID], nil
}

// eagerEventRepo models `user_subscription_events.event_key` being UNIQUE and
// `CreateIfAbsent` refusing the loser rather than failing it.
type eagerEventRepo struct {
	byKey   map[string]*entities.UserSubscriptionEvent
	keys    []string
	creates int
	// lastCtx is the transaction context the insert ran on. FR-015 requires the
	// history row and the outbox row to land in the same transaction as the
	// subscription row, so the context has to be the caller's.
	lastCtx context.Context
}

func newEagerEventRepo() *eagerEventRepo {
	return &eagerEventRepo{byKey: map[string]*entities.UserSubscriptionEvent{}}
}

func (r *eagerEventRepo) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	_, err := r.CreateIfAbsent(ctx, event)
	return err
}

func (r *eagerEventRepo) CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (bool, error) {
	r.lastCtx = ctx
	if _, exists := r.byKey[event.EventKey]; exists {
		return false, nil
	}
	r.creates++
	r.keys = append(r.keys, event.EventKey)
	// GORM assigns the primary key on the caller's own struct during INSERT, and
	// the payload is built from that struct afterwards - it carries the id as
	// sourceEventId so the receiver can de-duplicate. Assigning it to a copy here
	// would produce an event the receiver refuses by design.
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	copied := *event
	r.byKey[event.EventKey] = &copied
	return true, nil
}

func (r *eagerEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	return r.byKey[key], nil
}

// eagerUow refuses a nested Execute. `EnsureDefaultSubscription` is called from
// inside the identity use case's unit of work, so a second `uow.Execute` would
// open a transaction on a different connection that cannot see the uncommitted
// user row - and `user_subscriptions.user_id` is a foreign key onto it. Failing
// loudly here is how a regression would be caught without a database.
type eagerUow struct {
	nestedCalled bool
}

var errEagerNestedUow = errors.New("nested unit of work: eager provisioning must reuse the caller's transaction")

func (u *eagerUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if inProvisionTx(ctx) {
		u.nestedCalled = true
		return errEagerNestedUow
	}
	return fn(context.WithValue(ctx, provisionTxKey{}, true))
}

type eagerHarness struct {
	support   *SubscriptionStateSupport
	subRepo   *eagerSubRepo
	eventRepo *eagerEventRepo
	publisher *recordingPublisher
	uow       *eagerUow
	userID    uuid.UUID
	plan      *entities.SubscriptionPlan
}

func newEagerHarness(t *testing.T) *eagerHarness {
	t.Helper()

	plan := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "free",
		Name:            "Free",
		IsActive:        true,
		PlanKind:        plankind.DefaultFree,
	}
	subRepo := newEagerSubRepo()
	eventRepo := newEagerEventRepo()
	publisher := &recordingPublisher{}
	uow := &eagerUow{}

	return &eagerHarness{
		subRepo:   subRepo,
		eventRepo: eventRepo,
		publisher: publisher,
		uow:       uow,
		userID:    uuid.New(),
		plan:      plan,
		support: NewSubscriptionStateSupport(
			subRepo,
			&provisionPlanRepo{
				fakeGrantPlanRepo: fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{"free": plan}},
				plan:              plan,
			},
			&provisionQuotaRepo{},
			NewPlanChangePublisher(eventRepo, publisher, noopPlanPeriodRecorder{}),
			uow,
		),
	}
}

// The core requirement: provisioning writes on the caller's transaction and does
// NOT open one of its own.
func TestEnsureDefaultSubscriptionRunsOnTheCallersTransaction(t *testing.T) {
	h := newEagerHarness(t)

	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("EnsureDefaultSubscription failed: %v", err)
	}
	if h.uow.nestedCalled {
		t.Error("must not open a nested transaction - the FK onto identity.users(id) needs the caller's")
	}
	if len(h.publisher.events) != 1 {
		t.Fatalf("expected exactly 1 published event, got %d", len(h.publisher.events))
	}
	if h.subRepo.provisionCalls != 1 {
		t.Errorf("expected exactly 1 ProvisionDefault call, got %d", h.subRepo.provisionCalls)
	}
	if h.subRepo.subs[h.userID] == nil {
		t.Fatal("the account must have a subscription row after provisioning")
	}
}

// FR-015: the history row must be written inside the caller's transaction, so a
// rollback takes the subscription row with it.
//
// The caller shape is the one the identity use cases use: the unit of work is
// opened by the CREATE, and provisioning runs inside its closure. Calling
// EnsureDefaultSubscription on a bare context - as the tests below do - is the
// degenerate case where there is no transaction to join, so this test drives it
// the way production does.
func TestEnsureDefaultSubscriptionWritesHistoryInsideTheCallersTransaction(t *testing.T) {
	h := newEagerHarness(t)

	err := h.uow.Execute(context.Background(), func(txCtx context.Context) error {
		return h.support.EnsureDefaultSubscription(txCtx, h.userID)
	})
	if err != nil {
		t.Fatalf("EnsureDefaultSubscription failed: %v", err)
	}
	if h.uow.nestedCalled {
		t.Error("must not open a nested transaction inside the caller's unit of work")
	}
	if !inProvisionTx(h.eventRepo.lastCtx) {
		t.Error("the history/outbox insert must run on the transaction the caller opened (FR-015)")
	}
}

// A publish failure must abort the caller's transaction, so the account is not
// left holding a plan the AI service never heard about. This is the same
// failure T126 fixed on the lazy door, and the eager door reaches it too.
func TestEnsureDefaultSubscriptionPropagatesThePublishFailure(t *testing.T) {
	h := newEagerHarness(t)
	failing := NewPlanChangePublisher(h.eventRepo, failingEventPublisher{}, noopPlanPeriodRecorder{})

	support := NewSubscriptionStateSupport(
		h.subRepo,
		&provisionPlanRepo{
			fakeGrantPlanRepo: fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{"free": h.plan}},
			plan:              h.plan,
		},
		&provisionQuotaRepo{},
		failing,
		h.uow,
	)

	err := h.uow.Execute(context.Background(), func(txCtx context.Context) error {
		return support.EnsureDefaultSubscription(txCtx, h.userID)
	})
	if err == nil {
		t.Fatal("must return an error when publishing fails, so the caller's transaction rolls back")
	}
	if !errors.Is(err, errProvisionPublish) {
		t.Errorf("must return the publish error, got = %v", err)
	}
}

// THE assertion this task exists for: eager first, lazy second, still ONE record.
//
// Before it there was one door; now there are two, and a customer created
// through one of them will reach the other. Two subscription rows would mean two
// entitlements and an unreconcilable bill, and two history rows would mean the AI
// service is told about a change that did not happen.
func TestLazyProvisioningAfterEagerCreatesNoSecondRecord(t *testing.T) {
	h := newEagerHarness(t)

	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("eager provisioning failed: %v", err)
	}

	// The lazy door, exactly as the first subscription read runs it.
	sub, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID)
	if err != nil {
		t.Fatalf("lazy provisioning failed: %v", err)
	}
	if sub == nil {
		t.Fatal("the lazy path must return the subscription the eager path created")
	}

	if h.subRepo.provisionCalls != 1 {
		t.Errorf("ProvisionDefault ran %d times, want 1 - the lazy path wrote a second row", h.subRepo.provisionCalls)
	}
	if len(h.subRepo.subs) != 1 {
		t.Errorf("expected exactly 1 subscription row, got %d", len(h.subRepo.subs))
	}
	if h.eventRepo.creates != 1 {
		t.Errorf("expected exactly 1 history row, got %d (keys: %v)", h.eventRepo.creates, h.eventRepo.keys)
	}
	if len(h.publisher.events) != 1 {
		t.Errorf("expected exactly 1 published event, got %d - the AI service would hear about the same grant twice", len(h.publisher.events))
	}

	// The lazy path must also not have opened a transaction: the row already
	// exists, which is its fast path.
	if h.uow.nestedCalled {
		t.Error("the lazy path must not open a transaction when the subscription already exists")
	}
}

// The reverse order matters too: an account provisioned lazily first (rows that
// predate this feature) must not gain a second record when something calls the
// eager door on it later.
func TestEagerProvisioningAfterLazyCreatesNoSecondRecord(t *testing.T) {
	h := newEagerHarness(t)

	if _, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("lazy provisioning failed: %v", err)
	}
	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("eager provisioning failed: %v", err)
	}

	if h.subRepo.provisionCalls != 1 {
		t.Errorf("ProvisionDefault ran %d times, want 1", h.subRepo.provisionCalls)
	}
	if len(h.subRepo.subs) != 1 {
		t.Errorf("expected exactly 1 subscription row, got %d", len(h.subRepo.subs))
	}
	if h.eventRepo.creates != 1 {
		t.Errorf("expected exactly 1 history row, got %d", h.eventRepo.creates)
	}
	if len(h.publisher.events) != 1 {
		t.Errorf("expected exactly 1 published event, got %d", len(h.publisher.events))
	}
}

// Retrying the creation request must not double-provision either. This is the
// case the event_key UNIQUE index was added for, and it is reachable in
// production: the admin endpoint is behind no idempotency key of its own.
func TestRepeatedEagerProvisioningCreatesNoSecondRecord(t *testing.T) {
	h := newEagerHarness(t)

	for i := 0; i < 3; i++ {
		if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
			t.Fatalf("attempt %d failed: %v", i, err)
		}
	}

	if len(h.subRepo.subs) != 1 {
		t.Errorf("expected exactly 1 subscription row after 3 attempts, got %d", len(h.subRepo.subs))
	}
	if h.eventRepo.creates != 1 {
		t.Errorf("expected exactly 1 history row after 3 attempts, got %d", h.eventRepo.creates)
	}
	if len(h.publisher.events) != 1 {
		t.Errorf("expected exactly 1 published event after 3 attempts, got %d", len(h.publisher.events))
	}
}

// An account that already has a subscription must be left alone. Without this
// check the eager door would overwrite a plan an operator had just granted.
func TestEnsureDefaultSubscriptionLeavesAnExistingSubscriptionAlone(t *testing.T) {
	h := newEagerHarness(t)
	h.subRepo.subs[h.userID] = &entities.UserSubscription{
		UserID:                 h.userID,
		CurrentPlanCode:        "premium-default",
		CurrentBenefitSnapshot: entities.JSONDocument(`{"aiChatDailyQuota":40}`),
		Version:                7,
	}

	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("EnsureDefaultSubscription failed: %v", err)
	}

	if h.subRepo.provisionCalls != 0 {
		t.Errorf("must not write when the account already has a subscription, ProvisionDefault ran %d times", h.subRepo.provisionCalls)
	}
	if got := h.subRepo.subs[h.userID].CurrentPlanCode; got != "premium-default" {
		t.Errorf("the existing plan must be preserved, got %q", got)
	}
	if h.eventRepo.creates != 0 {
		t.Errorf("must not write a history row for an account that changed nothing, got %d", h.eventRepo.creates)
	}
}

// FR-070 / FR-071: the benefit snapshot is recorded at provisioning time. `{}`
// records nothing, so an account reconciled later has nothing to reconcile
// against.
func TestEnsureDefaultSubscriptionRecordsThePlanBenefits(t *testing.T) {
	h := newEagerHarness(t)
	h.plan.MaxWardrobeItems = 120
	h.plan.MaxOutfits = 30
	h.plan.AiOutfitDailyQuota = 7
	h.plan.AiChatDailyQuota = 40

	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("EnsureDefaultSubscription failed: %v", err)
	}

	sub := h.subRepo.subs[h.userID]
	if sub == nil {
		t.Fatal("expected a provisioned subscription")
	}
	if string(sub.CurrentBenefitSnapshot) != string(BuildPlanBenefitSnapshot(h.plan)) {
		t.Errorf("the snapshot must be the plan's benefits, got %s", sub.CurrentBenefitSnapshot)
	}
}

// The event key must be derived from the account and the committed version, or a
// retry computes a different key and the UNIQUE index stops deduplicating.
//
// The version is 0 because that is what the body stores: `provisionDefaultOn`
// never assigns `Version`, and the column defaults to 0. Asserting `:1` here
// would have pinned a value production never emits, and the assertion would still
// have been satisfied because the fake was inventing it.
func TestEnsureDefaultSubscriptionKeysTheEventByAccountAndVersion(t *testing.T) {
	h := newEagerHarness(t)

	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("EnsureDefaultSubscription failed: %v", err)
	}

	want := "DEFAULT_PROVISION:" + h.userID.String() + ":0"
	if len(h.eventRepo.keys) != 1 || h.eventRepo.keys[0] != want {
		t.Errorf("expected the event key %q, got %v", want, h.eventRepo.keys)
	}
	if got := h.subRepo.subs[h.userID].Version; got != 0 {
		t.Errorf("the stored version must be the schema default 0, got %d", got)
	}
}

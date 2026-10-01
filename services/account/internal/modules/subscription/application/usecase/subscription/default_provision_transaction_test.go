package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/shopspring/decimal"
	sharedevent "smart-wardrobe-be/shared/application/event"

	"github.com/google/uuid"
)

// ---- 024 T126: default provisioning is ONE transaction ----
//
// Constitution §III requires the outbox record to be written inside the same
// transaction as the business operation. Default provisioning writes three things
// - the subscription row, the history row and the outbox row - and before this
// task each fell back to the base database handle and committed on its own. The
// failure that motivated it: an account ended up holding a plan the AI service
// had never heard about, and nothing anywhere reported an error.
//
// These tests model the only thing the assertion needs: whether a given
// repository call happened INSIDE the transaction the unit of work opened. A call
// made at nesting depth 0 is a write on the base handle, which is the bug.

type provisionTxKey struct{}

func inProvisionTx(ctx context.Context) bool {
	v, _ := ctx.Value(provisionTxKey{}).(bool)
	return v
}

// provisionTxUow tracks nesting depth per repository call and discards writes when
// the closure fails, so a test can prove a row did NOT survive a rollback.
type provisionTxUow struct {
	depth int
	// maxDepth is the deepest nesting observed across every call the closure made.
	maxDepth     int
	closed       bool
	rolledBack   bool
	committed    bool
	nestedCalled bool
}

func (u *provisionTxUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if u.depth > 0 {
		// A nested Execute starts a SECOND transaction, which the constitution
		// forbids. Failing loudly beats silently running the inner one on its own
		// handle - the exact failure mode this task removes.
		u.nestedCalled = true
		return errors.New("nested unit of work: provisioning must reuse the outer transaction")
	}
	u.depth++
	err := fn(context.WithValue(ctx, provisionTxKey{}, true))
	u.depth--
	u.closed = true
	if err != nil {
		u.rolledBack = true
		return err
	}
	u.committed = true
	return nil
}

// observe records the nesting depth a repository call was made at.
func (u *provisionTxUow) observe() {
	if u.depth > u.maxDepth {
		u.maxDepth = u.depth
	}
}

// provisionSubRepo reuses the full IUserSubscriptionRepository fake from the grant
// tests and adds transaction-depth bookkeeping plus a rollback that discards the
// provisioned row.
type provisionSubRepo struct {
	fakeGrantSubRepo
	uow *provisionTxUow
	// staged is the row ProvisionDefault wrote inside the open transaction: it
	// becomes visible only if that transaction commits.
	staged *entities.UserSubscription
	// stagedReads records, per re-read, whether the call saw the transaction.
	stagedReads []bool
}

func (r *provisionSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	r.uow.observe()
	if inProvisionTx(ctx) {
		// Inside a transaction the row only exists if the transaction commits, so
		// stage it. The fake unit of work drops staged writes on rollback.
		r.staged = sub
		return nil
	}
	r.subs[sub.UserID] = sub
	return nil
}

func (r *provisionSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	r.uow.observe()
	if sub, ok := r.subs[userID]; ok {
		return sub, nil
	}
	// ProvisionDefault above may have staged the row inside the open transaction;
	// an uncommitted row IS visible to the transaction that wrote it.
	if r.staged != nil && r.staged.UserID == userID {
		r.stagedReads = append(r.stagedReads, inProvisionTx(ctx))
		return r.staged, nil
	}
	return nil, nil
}

func (r *provisionSubRepo) commit() {
	if r.staged != nil {
		r.subs[r.staged.UserID] = r.staged
	}
}

func (r *provisionSubRepo) rollback() {
	r.staged = nil
}

// provisionPlanRepo serves a default plan. The grant fake returns nil, which would
// make the state support stop at ErrDefaultPlanConfigNotFound.
type provisionPlanRepo struct {
	fakeGrantPlanRepo
	plan *entities.SubscriptionPlan
}

func (r *provisionPlanRepo) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return r.plan, nil
}

// provisionQuotaRepo satisfies IUserDailyQuotaRepository. Provisioning never
// touches it; any unexpected call panics on the nil embedded interface, which
// fails the test instead of silently doing nothing.
type provisionQuotaRepo struct {
	fakeGrantQuotaRepo
}

// failingEventPublisher refuses the outbox write, standing in for the failure that
// used to leave an orphaned subscription behind.
type failingEventPublisher struct{}

var errProvisionPublish = errors.New("outbox insert failed")

func (failingEventPublisher) Publish(ctx context.Context, topic string, payload any) error {
	return errProvisionPublish
}

// recordingPublisher succeeds, so tests can assert the event was published.
type recordingPublisher struct {
	events []any
}

func (p *recordingPublisher) Publish(ctx context.Context, topic string, payload any) error {
	p.events = append(p.events, payload)
	return nil
}

var _ sharedevent.IEventPublisher = (*recordingPublisher)(nil)
var _ sharedevent.IEventPublisher = (*failingEventPublisher)(nil)

type provisionHarness struct {
	support   *SubscriptionStateSupport
	uow       *provisionTxUow
	subRepo   *provisionSubRepo
	publisher *recordingPublisher
	userID    uuid.UUID
	plan      *entities.SubscriptionPlan
}

func newProvisionHarness(t *testing.T, publisher sharedevent.IEventPublisher) *provisionHarness {
	t.Helper()

	userID := uuid.New()
	plan := &entities.SubscriptionPlan{
		// 024 T155: read from a database this always has an id, and it is carried to
		// the AI receiver as toPlanId. Without it the payload is refused by design.
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "free",
		Name:            "Free",
		IsActive:        true,
		Price:           decimal.Zero,
		PlanKind:        plankind.DefaultFree,
	}
	uow := &provisionTxUow{}
	subRepo := &provisionSubRepo{
		fakeGrantSubRepo: fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{}},
		uow:              uow,
	}
	recorder, _ := publisher.(*recordingPublisher)

	h := &provisionHarness{
		uow:       uow,
		subRepo:   subRepo,
		publisher: recorder,
		userID:    userID,
		plan:      plan,
		support: NewSubscriptionStateSupport(
			subRepo,
			&provisionPlanRepo{
				fakeGrantPlanRepo: fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{"free": plan}},
				plan:              plan,
			},
			&provisionQuotaRepo{},
			NewPlanChangePublisher(&fakeRenewalEventRepo{}, publisher),
			uow,
		),
	}
	return h
}

// The core assertion: every write of the provisioning path runs inside the
// transaction the unit of work opened, and the transaction commits.
func TestDefaultProvisioningWritesEverythingInsideOneTransaction(t *testing.T) {
	h := newProvisionHarness(t, &recordingPublisher{})

	sub, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID)
	if err != nil {
		t.Fatalf("GetOrCreateUserSubscription failed: %v", err)
	}
	if sub == nil {
		t.Fatal("must return the freshly provisioned subscription")
	}
	if !h.uow.closed || !h.uow.committed {
		t.Fatalf("must open and commit one transaction, closed=%v commit=%v", h.uow.closed, h.uow.committed)
	}
	if h.uow.rolledBack {
		t.Fatal("must not roll back when everything succeeds")
	}
	if h.uow.nestedCalled {
		t.Error("must not open a nested transaction")
	}
	if h.uow.maxDepth < 1 {
		t.Error("at least one write must run inside the transaction")
	}
	if h.subRepo.staged == nil {
		t.Error("ProvisionDefault must run INSIDE the transaction (Constitution §III)")
	}
	if len(h.subRepo.stagedReads) == 0 {
		t.Fatal("must re-read after the write")
	}
	for i, inTx := range h.subRepo.stagedReads {
		if !inTx {
			t.Errorf("re-read %d must run inside the transaction, otherwise the version in the event key can diverge from the row actually committed", i)
		}
	}
}

// The failure that motivated T126: when publishing fails, the subscription row
// must NOT survive. Before the task the row was already committed by the time the
// publish was attempted, which is how an account ended up with a plan nobody knew
// about and no error anywhere.
func TestDefaultProvisioningRollsBackTheSubscriptionWhenPublishingFails(t *testing.T) {
	h := newProvisionHarness(t, failingEventPublisher{})

	sub, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID)
	if err == nil {
		t.Fatal("must return an error when publishing fails")
	}
	if !errors.Is(err, errProvisionPublish) {
		t.Errorf("must return the publish error, got = %v", err)
	}
	if sub != nil {
		t.Error("must not return a subscription after the transaction rolled back")
	}
	if !h.uow.closed {
		t.Fatal("must close the transaction on error")
	}
	if !h.uow.rolledBack {
		t.Error("must roll back every change when publishing fails")
	}
	if h.uow.committed {
		t.Error("must not commit when publishing fails")
	}
	h.subRepo.rollback()
	if _, ok := h.subRepo.subs[h.userID]; ok {
		t.Error("the rolled-back subscription row still exists - which is exactly the defect T126 removed")
	}
}

// The fast path stays OUTSIDE the transaction: an existing subscription is the
// overwhelmingly common case, and holding a transaction open for a read would
// serialise every subscription read in the system.
func TestDefaultProvisioningSkipsTheTransactionWhenTheSubscriptionAlreadyExists(t *testing.T) {
	h := newProvisionHarness(t, &recordingPublisher{})
	h.subRepo.subs[h.userID] = &entities.UserSubscription{
		UserID:             h.userID,
		SubscriptionPlanID: h.plan.ID,
		CurrentPlanCode:    "free",
	}

	sub, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID)
	if err != nil {
		t.Fatalf("GetOrCreateUserSubscription failed: %v", err)
	}
	if sub == nil {
		t.Fatal("must return the existing subscription")
	}
	if h.uow.closed {
		t.Error("the fast read path must not open a transaction")
	}
	if h.subRepo.staged != nil {
		t.Error("must not write again when the subscription already exists")
	}
}

// 024 T170 / FR-070 + FR-071: the single-account provisioning path must record the
// plan's benefits, exactly as the batch path does.
//
// It used to write `{}`. Same default plan, two different snapshots depending on
// which door the account came through, and `{}` is not a weak record - it is no
// record. FR-070/FR-071 ask for the plan's benefits to be captured at the moment of
// the change so an account can be reconciled back to what it was granted; an account
// provisioned here had nothing to reconcile against.
//
// This is the COMMON path, not an edge case: most accounts are provisioned the first
// time something reads their subscription, and the bulk path only runs for
// administrators granting plans in bulk.
func TestDefaultProvisioningRecordsThePlanBenefits(t *testing.T) {
	h := newProvisionHarness(t, &recordingPublisher{})

	// Give the plan non-default benefits, so an empty snapshot cannot pass by
	// coincidence: every field in the snapshot would be zero for a bare fixture.
	h.plan.MaxWardrobeItems = 120
	h.plan.MaxOutfits = 30
	h.plan.AiOutfitDailyQuota = 7
	h.plan.AiChatDailyQuota = 40

	sub, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID)
	if err != nil {
		t.Fatalf("GetOrCreateUserSubscription loi: %v", err)
	}
	if sub == nil {
		t.Fatal("phai tra ve subscription vua cap")
	}

	var recorded struct {
		MaxWardrobeItems   int `json:"maxWardrobeItems"`
		MaxOutfits         int `json:"maxOutfits"`
		AiOutfitDailyQuota int `json:"aiOutfitDailyQuota"`
		AiChatDailyQuota   int `json:"aiChatDailyQuota"`
	}
	if err := json.Unmarshal([]byte(sub.CurrentBenefitSnapshot), &recorded); err != nil {
		t.Fatalf("anh chup khong phai JSON: %v (raw = %s)", err, sub.CurrentBenefitSnapshot)
	}

	if recorded.MaxWardrobeItems != 120 || recorded.MaxOutfits != 30 ||
		recorded.AiOutfitDailyQuota != 7 || recorded.AiChatDailyQuota != 40 {
		t.Errorf("anh chup phai ghi dung quyen loi cua goi, co = %+v", recorded)
	}
}

// The two paths must not drift again. This is the cross-check that failed: the batch
// path was always correct, so comparing the two outputs against the same plan is what
// catches the single-account path regressing on its own.
func TestDefaultProvisionsBothPathsAgreeOnTheSnapshot(t *testing.T) {
	h := newProvisionHarness(t, &recordingPublisher{})

	h.plan.MaxWardrobeItems = 120
	h.plan.MaxOutfits = 30
	h.plan.AiOutfitDailyQuota = 7
	h.plan.AiChatDailyQuota = 40

	sub, err := h.support.GetOrCreateUserSubscription(context.Background(), h.userID)
	if err != nil {
		t.Fatalf("GetOrCreateUserSubscription loi: %v", err)
	}

	batch := BuildPlanBenefitSnapshot(h.plan)
	if string(sub.CurrentBenefitSnapshot) != string(batch) {
		t.Errorf("hai duong cap goi mac dinh phai ghi cung mot anh chup\n  don le   = %s\n  hang loat = %s",
			sub.CurrentBenefitSnapshot, batch)
	}
}

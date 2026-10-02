package purchase

import (
	"context"
	"encoding/json"
	"testing"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 025 T114 — FR-095: proof that the plan-change event is published on the
// purchase path.
//
// The purchase path is the one a real paying customer takes, and FR-095 calls
// its failure mode out explicitly: a path that publishes nothing "does not
// report an error anywhere". So this file asserts a COUNT — one purchase of one
// paid plan produces exactly one plan-change history row and exactly one
// published subscription.plan.changed — and never the absence of an error.
//
// The webhook settlement in `webhook/purchase_workflow.go` covers the gateway
// side of the same path, but only in `webhook_integration_test.go`, which skips
// itself unless TEST_DATABASE_URL is set. An assertion that only exists in a
// skipped test is not coverage, so the wallet route is proved here with fakes
// that always run.
//
// Nothing here sends email. The notification contract is a fake and the account
// brief deliberately carries no address, so `wallet.go`'s success-email branch
// is not entered at all — zero outbound mail, as the constitution's cap on
// verification runs requires.

// ---- Fakes ----

// purchaseWalletRepo holds a single wallet and counts the balance writes.
type purchaseWalletRepo struct {
	wallet  *entities.UserWallet
	updates int
}

func (f *purchaseWalletRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error) {
	return f.wallet, nil
}
func (f *purchaseWalletRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error) {
	return f.wallet, nil
}
func (f *purchaseWalletRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserWallet, error) {
	return nil, nil
}
func (f *purchaseWalletRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserWallet, error) {
	return nil, nil
}
func (f *purchaseWalletRepo) GetAll(ctx context.Context) ([]*entities.UserWallet, error) {
	return nil, nil
}
func (f *purchaseWalletRepo) Create(ctx context.Context, e *entities.UserWallet) error { return nil }
func (f *purchaseWalletRepo) Delete(ctx context.Context, id uuid.UUID) error           { return nil }
func (f *purchaseWalletRepo) Update(ctx context.Context, e *entities.UserWallet) error {
	f.updates++
	return nil
}
func (f *purchaseWalletRepo) BatchUpdate(ctx context.Context, es []*entities.UserWallet) error {
	return nil
}
func (f *purchaseWalletRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// purchaseStatementRepo counts ledger rows written by the purchase.
type purchaseStatementRepo struct {
	creates int
}

func (f *purchaseStatementRepo) Create(ctx context.Context, e *entities.WalletStatement) error {
	f.creates++
	return nil
}
func (f *purchaseStatementRepo) GetByUserID(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.WalletStatement, error) {
	return nil, nil
}
func (f *purchaseStatementRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *purchaseStatementRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.WalletStatement, error) {
	return nil, nil
}
func (f *purchaseStatementRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WalletStatement, error) {
	return nil, nil
}
func (f *purchaseStatementRepo) GetAll(ctx context.Context) ([]*entities.WalletStatement, error) {
	return nil, nil
}
func (f *purchaseStatementRepo) Update(ctx context.Context, e *entities.WalletStatement) error {
	return nil
}
func (f *purchaseStatementRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *purchaseStatementRepo) BatchUpdate(ctx context.Context, es []*entities.WalletStatement) error {
	return nil
}
func (f *purchaseStatementRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// purchaseEventRepo is the plan-change history store. `creates` is the number of
// history rows this single purchase wrote — the thing FR-095 asks to be counted,
// and the thing a silent path would leave at zero.
type purchaseEventRepo struct {
	creates int
	rows    []*entities.UserSubscriptionEvent
}

func (f *purchaseEventRepo) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	f.creates++
	f.rows = append(f.rows, event)
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return nil
}

func (f *purchaseEventRepo) CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (bool, error) {
	if err := f.Create(ctx, event); err != nil {
		return false, err
	}
	return true, nil
}

func (f *purchaseEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	return nil, nil
}

// purchaseEventPublisher records what would have gone to the broker.
type purchaseEventPublisher struct {
	events []any
}

func (p *purchaseEventPublisher) Publish(_ context.Context, _ string, payload any) error {
	p.events = append(p.events, payload)
	return nil
}

// purchaseBrief answers lookups with an account that has NO email address, which
// is what keeps `wallet.go` out of its success-email branch. A test fixture with
// a real address would be one stray send away from the metered relay.
type purchaseBrief struct{}

func (f *purchaseBrief) GetByID(ctx context.Context, userID uuid.UUID) (*identity_dto.UserBriefContractDTO, error) {
	return &identity_dto.UserBriefContractDTO{ID: userID}, nil
}
func (f *purchaseBrief) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}
func (f *purchaseBrief) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}
func (f *purchaseBrief) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}

// purchaseNotif records any notification attempt so the test can prove none was
// made.
type purchaseNotif struct {
	sent []string
}

func (f *purchaseNotif) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	f.sent = append(f.sent, string(template))
	return nil
}
func (f *purchaseNotif) SendByUserID(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, userID uuid.UUID) error {
	f.sent = append(f.sent, string(template))
	return nil
}

// ---- Helpers ----

func purchaseFreePlan() *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "free",
		Name:            "Free",
		IsActive:        true,
		Price:           decimal.Zero,
		TierRank:        0,
		PlanKind:        plankind.DefaultFree,
	}
}

// purchaseFinitePlan is `paidPlan()` from checkout_test.go plus the plan kind.
// The checkout fixtures never needed it — only a wallet purchase does, because
// the entity refuses to activate a plan whose kind is neither finite nor
// lifetime.
func purchaseFinitePlan() *entities.SubscriptionPlan {
	plan := paidPlan()
	plan.PlanKind = plankind.Finite
	return plan
}

type purchaseHarness struct {
	uc         *SubscriptionPurchaseUseCase
	plan       *entities.SubscriptionPlan
	sub        *entities.UserSubscription
	events     *purchaseEventRepo
	publisher  *purchaseEventPublisher
	statements *purchaseStatementRepo
	wallet     *purchaseWalletRepo
	notif      *purchaseNotif
}

// newPurchaseHarness wires a wallet purchase for one account sitting on the free
// plan. The real PlanChangePublisher is constructed over fakes for the same
// reason the renewal harness does it: the publish step is only observable if the
// step actually runs.
func newPurchaseHarness(t *testing.T) *purchaseHarness {
	t.Helper()
	cfg := checkoutTestConfig()
	free := purchaseFreePlan()
	plan := purchaseFinitePlan()
	userID := uuid.New()
	sub := &entities.UserSubscription{
		UserID:                 userID,
		SubscriptionPlanID:     free.ID,
		SubscriptionPlan:       free,
		CurrentPlanCode:        free.Slug,
		CurrentTierRank:        free.TierRank,
		CurrentPlanKind:        plankind.DefaultFree,
		CurrentBenefitSnapshot: entities.JSONDocument(`{}`),
	}
	h := &purchaseHarness{
		plan:       plan,
		sub:        sub,
		events:     &purchaseEventRepo{},
		publisher:  &purchaseEventPublisher{},
		statements: &purchaseStatementRepo{},
		wallet:     &purchaseWalletRepo{wallet: &entities.UserWallet{UserID: userID, Balance: decimal.NewFromInt(500000), Currency: currency.VND}},
		notif:      &purchaseNotif{},
	}
	h.uc = &SubscriptionPurchaseUseCase{
		walletRepo:          h.wallet,
		depositTxRepo:       &fakeDepositTxRepo{},
		statementRepo:       h.statements,
		planRepo:            &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{plan.Slug: plan}, defaultPlan: free},
		userSubRepo:         &fakeSubRepo{sub: sub},
		eventRepo:           h.events,
		planChangePublisher: subscription.NewPlanChangePublisher(h.events, h.publisher),
		uow:                 &fakeUow{},
		cfg:                 &cfg,
		notifContract:       h.notif,
		userBriefContract:   &purchaseBrief{},
	}
	return h
}

// publishedPlanChangeType reads `eventType` out of a captured payload. The
// payload struct is unexported in the mapper package by design (constitution §II:
// the JSON is the contract), so going through JSON also asserts on what the
// receiver would receive rather than on a struct the publisher never serialised.
func publishedPlanChangeType(t *testing.T, payload any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal published plan-change payload: %v", err)
	}
	var decoded struct {
		EventType string `json:"eventType"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode published plan-change payload: %v", err)
	}
	return decoded.EventType
}

// ---- Tests ----

// TestPurchasePlanWithWalletPublishesExactlyOnePlanChangeEvent is FR-095's
// purchase path: one wallet purchase of one paid plan must write exactly one
// plan-change history row and publish exactly one subscription.plan.changed.
func TestPurchasePlanWithWalletPublishesExactlyOnePlanChangeEvent(t *testing.T) {
	h := newPurchaseHarness(t)

	if err := h.uc.PurchasePlanWithWallet(context.Background(), h.sub.UserID, h.plan.Slug); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The purchase really happened, so the counts below are not vacuously true:
	// a no-op would also write zero rows, and this test is about the announce,
	// not about whether the plan moved.
	if h.sub.CurrentPlanCode != h.plan.Slug {
		t.Fatalf("expected subscription on %q, got %q", h.plan.Slug, h.sub.CurrentPlanCode)
	}
	if h.statements.creates != 1 {
		t.Fatalf("expected one purchase debit statement, got %d", h.statements.creates)
	}
	if h.wallet.updates != 1 {
		t.Fatalf("expected the wallet debited exactly once, got %d writes", h.wallet.updates)
	}

	if h.events.creates != 1 {
		t.Fatalf("FR-095: one purchase must write exactly one plan-change history row, got %d", h.events.creates)
	}
	if h.events.rows[0].EventType != string(subscriptioneventtype.ACTIVATED) {
		t.Fatalf("expected history row type %q, got %q", subscriptioneventtype.ACTIVATED, h.events.rows[0].EventType)
	}
	if len(h.publisher.events) != 1 {
		t.Fatalf("FR-095: one purchase must publish exactly one subscription.plan.changed, got %d", len(h.publisher.events))
	}
	if got := publishedPlanChangeType(t, h.publisher.events[0]); got != string(subscriptioneventtype.ACTIVATED) {
		t.Fatalf("expected published event type %q, got %q", subscriptioneventtype.ACTIVATED, got)
	}

	// Constitution: a verification run must not reach the real relay. The
	// fixture has no email address, so the purchase-success mail must never have
	// been attempted - and the fake means nothing could have left the process
	// even if it had.
	if len(h.notif.sent) != 0 {
		t.Fatalf("expected zero notifications from this fixture, got %v", h.notif.sent)
	}
}

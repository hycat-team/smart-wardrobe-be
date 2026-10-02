package subscription

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	_ "smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/subscriptionrenewalstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// ---- Fakes ----

type fakeRenewalUow struct{}

func (f *fakeRenewalUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type fakeRenewalSubRepo struct {
	lockedSub *entities.UserSubscription
	updates   int
}

func (f *fakeRenewalSubRepo) GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error) {
	return f.lockedSub, nil
}
func (f *fakeRenewalSubRepo) Update(ctx context.Context, e *entities.UserSubscription) error {
	f.updates++
	return nil
}
func (f *fakeRenewalSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.lockedSub, nil
}
func (f *fakeRenewalSubRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.lockedSub, nil
}
func (f *fakeRenewalSubRepo) GetByUserIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) GetAll(ctx context.Context) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) Create(ctx context.Context, e *entities.UserSubscription) error {
	return nil
}
func (f *fakeRenewalSubRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeRenewalSubRepo) GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	return nil
}
func (f *fakeRenewalSubRepo) GetDistributionByPlanCode(ctx context.Context) ([]repositories.SubscriptionDistributionDTO, error) {
	return nil, nil
}
func (f *fakeRenewalSubRepo) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	return nil, 0, nil
}
func (f *fakeRenewalSubRepo) BatchUpdate(ctx context.Context, es []*entities.UserSubscription) error {
	return nil
}
func (f *fakeRenewalSubRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeRenewalWalletRepo struct {
	wallet  *entities.UserWallet
	updates int
}

func (f *fakeRenewalWalletRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error) {
	return f.wallet, nil
}
func (f *fakeRenewalWalletRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error) {
	return f.wallet, nil
}
func (f *fakeRenewalWalletRepo) Update(ctx context.Context, e *entities.UserWallet) error {
	f.updates++
	return nil
}
func (f *fakeRenewalWalletRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserWallet, error) {
	return nil, nil
}
func (f *fakeRenewalWalletRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserWallet, error) {
	return nil, nil
}
func (f *fakeRenewalWalletRepo) GetAll(ctx context.Context) ([]*entities.UserWallet, error) {
	return nil, nil
}
func (f *fakeRenewalWalletRepo) Create(ctx context.Context, e *entities.UserWallet) error { return nil }
func (f *fakeRenewalWalletRepo) Delete(ctx context.Context, id uuid.UUID) error           { return nil }
func (f *fakeRenewalWalletRepo) BatchUpdate(ctx context.Context, es []*entities.UserWallet) error {
	return nil
}
func (f *fakeRenewalWalletRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeRenewalStatementRepo struct {
	creates int
}

func (f *fakeRenewalStatementRepo) Create(ctx context.Context, e *entities.WalletStatement) error {
	f.creates++
	return nil
}
func (f *fakeRenewalStatementRepo) GetByUserID(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.WalletStatement, error) {
	return nil, nil
}
func (f *fakeRenewalStatementRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeRenewalStatementRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.WalletStatement, error) {
	return nil, nil
}
func (f *fakeRenewalStatementRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WalletStatement, error) {
	return nil, nil
}
func (f *fakeRenewalStatementRepo) GetAll(ctx context.Context) ([]*entities.WalletStatement, error) {
	return nil, nil
}
func (f *fakeRenewalStatementRepo) Update(ctx context.Context, e *entities.WalletStatement) error {
	return nil
}
func (f *fakeRenewalStatementRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeRenewalStatementRepo) BatchUpdate(ctx context.Context, es []*entities.WalletStatement) error {
	return nil
}
func (f *fakeRenewalStatementRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeRenewalAttemptRepo struct {
	existing *entities.SubscriptionRenewalAttempt
	updates  int
}

func (f *fakeRenewalAttemptRepo) GetByKey(ctx context.Context, key string) (*entities.SubscriptionRenewalAttempt, error) {
	return f.existing, nil
}
func (f *fakeRenewalAttemptRepo) Create(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error {
	return nil
}
func (f *fakeRenewalAttemptRepo) Update(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error {
	f.updates++
	return nil
}
func (f *fakeRenewalAttemptRepo) GetRenewalStats(ctx context.Context) (*repositories.RenewalStatsDTO, error) {
	return nil, nil
}

type fakeRenewalEventRepo struct {
	creates int
}

func (f *fakeRenewalEventRepo) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	f.creates++
	// 024 T155: GORM assigns the primary key on insert and the receiver needs it as
	// sourceEventId. A fake that skips this produces events the receiver would refuse.
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return nil
}
func (f *fakeRenewalEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	return nil, nil
}

// CreateIfAbsent satisfies the interface added by 024 T148. This fake reports
// every call as an insert because it holds no rows, which matches a fresh account.
//
// 024 T155: the id assignment matters here too - this is the method
// PublishPlanChange actually calls, so skipping it produced events with no
// sourceEventId, which the AI receiver refuses by design.
func (f *fakeRenewalEventRepo) CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (bool, error) {
	f.creates++
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return true, nil
}

type fakeRenewalPlanRepo struct {
	planByID map[uuid.UUID]*entities.SubscriptionPlan
}

func (f *fakeRenewalPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlan, error) {
	return f.planByID[id], nil
}
func (f *fakeRenewalPlanRepo) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeRenewalPlanRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeRenewalPlanRepo) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeRenewalPlanRepo) GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return nil, nil
}

// GetBySlugsAnyStatus satisfies the interface added by 024 T150. The renewal path
// loads plans by id, never by slug set, so nothing in these tests calls it.
func (f *fakeRenewalPlanRepo) GetBySlugsAnyStatus(ctx context.Context, slugs []string) (map[string]*entities.SubscriptionPlan, error) {
	return map[string]*entities.SubscriptionPlan{}, nil
}
func (f *fakeRenewalPlanRepo) ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeRenewalPlanRepo) GetAll(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeRenewalPlanRepo) Create(ctx context.Context, e *entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeRenewalPlanRepo) Update(ctx context.Context, e *entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeRenewalPlanRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeRenewalPlanRepo) BatchUpdate(ctx context.Context, es []*entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeRenewalPlanRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeRenewalNotif struct {
	sent []string
}

func (f *fakeRenewalNotif) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	f.sent = append(f.sent, string(template))
	return nil
}
func (f *fakeRenewalNotif) SendByUserID(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, userID uuid.UUID) error {
	f.sent = append(f.sent, string(template))
	return nil
}

// ---- Helpers ----

func renewalTestPlan() *entities.SubscriptionPlan {
	days := 30
	return &entities.SubscriptionPlan{
		// 024 T155: a plan read from the database always has an id, and the
		// subscription.plan.changed payload carries it as toPlanId. The AI receiver
		// refuses a message without one, so a fixture without it is a fixture the
		// receiver would reject - and any test that published through it was passing
		// on an event that could never have been applied.
		AuditableEntity:    entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:               "premium",
		Name:               "Premium",
		IsActive:           true,
		Price:              decimal.NewFromInt(99000),
		TierRank:           1,
		PlanKind:           plankind.Finite,
		DurationDays:       &days,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
	}
}

type renewalHarness struct {
	uc         *SubscriptionUseCase
	subRepo    *fakeRenewalSubRepo
	walletRepo *fakeRenewalWalletRepo
	attempt    *fakeRenewalAttemptRepo
	statements *fakeRenewalStatementRepo
	events     *fakeRenewalEventRepo
	// 025 T114: kept on the harness so a test can inspect what actually reached
	// the broker. FR-095 asks for proof that the plan-change EVENT is published on
	// the renewal and expiry paths, and the row counter alone only proves the
	// history write - a regression that kept the insert and dropped the publish
	// would still pass a counter-only assertion.
	publisher  *fakeRenewalPublisher
	notif      *fakeRenewalNotif
	freePlanID uuid.UUID
	plan       *entities.SubscriptionPlan
	now        time.Time
}

func newRenewalHarness(t *testing.T, autoRenew bool, walletBalance int64) *renewalHarness {
	t.Helper()
	plan := renewalTestPlan()
	freePlanID := uuid.New()
	now := time.Now()
	expired := now.Add(-time.Hour)
	sub := &entities.UserSubscription{
		UserID:                 uuid.New(),
		SubscriptionPlanID:     plan.ID,
		SubscriptionPlan:       plan,
		CurrentPlanCode:        plan.Slug,
		CurrentTierRank:        plan.TierRank,
		CurrentPlanKind:        plankind.Finite,
		CurrentBenefitSnapshot: entities.JSONDocument(`{}`),
		IsAutoRenewEnabled:     autoRenew,
		ExpiresAt:              &expired,
	}
	wallet := &entities.UserWallet{UserID: sub.UserID, Balance: decimal.NewFromInt(walletBalance), Currency: currency.VND}
	h := &renewalHarness{
		subRepo:    &fakeRenewalSubRepo{lockedSub: sub},
		walletRepo: &fakeRenewalWalletRepo{wallet: wallet},
		attempt:    &fakeRenewalAttemptRepo{},
		statements: &fakeRenewalStatementRepo{},
		events:     &fakeRenewalEventRepo{},
		notif:      &fakeRenewalNotif{},
		freePlanID: freePlanID,
		plan:       plan,
		now:        now,
	}
	// 024 T051: the expiry path now publishes through the single helper, so the
	// harness wires the real publisher over the fakes. This is what makes the
	// test able to catch a regression where the publish step is dropped.
	publisher := &fakeRenewalPublisher{}
	h.publisher = publisher
	publisherHelper := NewPlanChangePublisher(h.events, publisher)
	h.uc = &SubscriptionUseCase{
		uow:           &fakeRenewalUow{},
		userSubRepo:   h.subRepo,
		walletRepo:    h.walletRepo,
		statementRepo: h.statements,
		// 024 T155: the map key IS the plan's id, so the value must carry it too.
		// Without it the payload's toPlanId is the nil UUID and the AI receiver
		// refuses the message - which is what used to happen here, silently, on every
		// renewal test.
		planRepo: &fakeRenewalPlanRepo{planByID: map[uuid.UUID]*entities.SubscriptionPlan{
			freePlanID: {
				AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: freePlanID}},
				Slug:            "free",
				IsActive:        true,
			},
		}},
		eventRepo:           h.events,
		renewalAttemptRepo:  h.attempt,
		notifContract:       h.notif,
		planChangePublisher: publisherHelper,
		log:                 zap.NewNop(),
	}
	return h
}

// fakeRenewalPublisher records published events without a broker.
type fakeRenewalPublisher struct {
	events []any
}

func (f *fakeRenewalPublisher) Publish(_ context.Context, _ string, payload any) error {
	f.events = append(f.events, payload)
	return nil
}

// ---- Tests ----

func TestExecuteScheduledRenewalDowngradesWhenAutoRenewOff(t *testing.T) {
	h := newRenewalHarness(t, false, 500000)

	result := RenewalExecutionResult{}
	err := h.uc.executeScheduledRenewal(context.Background(), h.subRepo.lockedSub.UserID, h.freePlanID, h.now, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusDowngraded {
		t.Fatalf("expected downgraded status, got %q", result.Status)
	}
	if h.subRepo.updates != 1 {
		t.Fatalf("expected subscription downgraded in repo, got %d updates", h.subRepo.updates)
	}
	if len(h.notif.sent) != 1 {
		t.Fatalf("expected renewal failure notification, got %d", len(h.notif.sent))
	}
}

func TestExecuteScheduledRenewalDowngradesOnInsufficientWallet(t *testing.T) {
	h := newRenewalHarness(t, true, 1000) // wallet far below plan price

	result := RenewalExecutionResult{}
	err := h.uc.executeScheduledRenewal(context.Background(), h.subRepo.lockedSub.UserID, h.freePlanID, h.now, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusDowngraded {
		t.Fatalf("expected downgraded status, got %q", result.Status)
	}
	if h.subRepo.updates != 1 {
		t.Fatalf("expected subscription downgraded in repo, got %d updates", h.subRepo.updates)
	}
}

func TestExecuteScheduledRenewalRenewsPaidPlan(t *testing.T) {
	h := newRenewalHarness(t, true, 500000)

	result := RenewalExecutionResult{}
	err := h.uc.executeScheduledRenewal(context.Background(), h.subRepo.lockedSub.UserID, h.freePlanID, h.now, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusRenewed {
		t.Fatalf("expected renewed status, got %q", result.Status)
	}
	if h.statements.creates != 1 {
		t.Fatalf("expected renewal debit statement, got %d", h.statements.creates)
	}
	if h.subRepo.updates != 1 {
		t.Fatalf("expected subscription extension persisted, got %d updates", h.subRepo.updates)
	}
	if h.notif.sent == nil || len(h.notif.sent) != 1 {
		t.Fatalf("expected renewal success notification, got %d", len(h.notif.sent))
	}
	_ = depositstatus.Pending
	_ = walletstatementtype.SubscriptionRenewal
	_ = subscriptionrenewalstatus.Succeeded
}

func TestExecuteScheduledRenewalSkipsWhenSubscriptionDisappears(t *testing.T) {
	h := newRenewalHarness(t, true, 500000)
	h.subRepo.lockedSub = nil

	result := RenewalExecutionResult{}
	err := h.uc.executeScheduledRenewal(context.Background(), uuid.New(), h.freePlanID, h.now, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusSkipped {
		t.Fatalf("expected skipped status, got %q", result.Status)
	}
	if h.subRepo.updates != 0 {
		t.Fatalf("expected no repo writes, got %d updates", h.subRepo.updates)
	}
}

// ---- 025 T114 — FR-095: proof that the plan-change event is published on the
// renewal and expiry-downgrade paths.
//
// The two tests above already cover that both paths CHANGE the subscription.
// What they never asked is whether the change announced itself. FR-095 names
// exactly that as the failure mode: a path that publishes nothing "does not
// report an error anywhere" - the debit is recorded, the downgrade is applied,
// every return value is nil, and the AI service keeps honouring a term that has
// ended.
//
// So the assertion has to be a COUNT on a single operation, never the absence
// of an error. "No error" is exactly what a silent path returns.

// publishedPlanChangeType reads `eventType` out of a captured payload.
//
// The payload type lives in the mapper package and is deliberately unexported
// (constitution §II: the JSON is the contract, the struct is not shared), so
// there is no field to read from here. Going through JSON also means this
// asserts on what the receiver would actually receive rather than on an
// in-memory struct that the publisher has not yet serialised.
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

// TestExecuteScheduledRenewalPublishesExactlyOnePlanChangeEvent covers the
// renewal path (gia hạn) of FR-095's six: one scheduled renewal of one
// subscription must write exactly one plan-change history row and publish
// exactly one subscription.plan.changed.
func TestExecuteScheduledRenewalPublishesExactlyOnePlanChangeEvent(t *testing.T) {
	h := newRenewalHarness(t, true, 500000)

	result := RenewalExecutionResult{}
	if err := h.uc.executeScheduledRenewal(context.Background(), h.subRepo.lockedSub.UserID, h.freePlanID, h.now, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusRenewed {
		t.Fatalf("expected renewed status, got %q", result.Status)
	}
	if h.events.creates != 1 {
		t.Fatalf("FR-095: one renewal must write exactly one plan-change history row, got %d", h.events.creates)
	}
	if len(h.publisher.events) != 1 {
		t.Fatalf("FR-095: one renewal must publish exactly one subscription.plan.changed, got %d", len(h.publisher.events))
	}
	if got := publishedPlanChangeType(t, h.publisher.events[0]); got != string(subscriptioneventtype.EXTENDED) {
		t.Fatalf("expected event type %q on the renewal publish, got %q", subscriptioneventtype.EXTENDED, got)
	}
}

// TestExecuteScheduledRenewalDowngradePublishesExactlyOnePlanChangeEvent covers
// the expiry-downgrade path (hạ về gói miễn phí khi hết hạn) — the one FR-095
// singles out as most consequential, because it is the transition that REMOVES a
// paying customer's entitlements. A downgrade nobody hears about leaves the AI
// service still enforcing a plan that ended.
func TestExecuteScheduledRenewalDowngradePublishesExactlyOnePlanChangeEvent(t *testing.T) {
	h := newRenewalHarness(t, false, 500000)

	result := RenewalExecutionResult{}
	if err := h.uc.executeScheduledRenewal(context.Background(), h.subRepo.lockedSub.UserID, h.freePlanID, h.now, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusDowngraded {
		t.Fatalf("expected downgraded status, got %q", result.Status)
	}
	if h.events.creates != 1 {
		t.Fatalf("FR-095: one downgrade to free must write exactly one plan-change history row, got %d", h.events.creates)
	}
	if len(h.publisher.events) != 1 {
		t.Fatalf("FR-095: one downgrade to free must publish exactly one subscription.plan.changed, got %d", len(h.publisher.events))
	}
	if got := publishedPlanChangeType(t, h.publisher.events[0]); got != string(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE) {
		t.Fatalf("expected event type %q on the downgrade publish, got %q", subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, got)
	}
}

// TestExecuteScheduledRenewalDowngradeOnInsufficientFundsPublishesExactlyOnePlanChangeEvent
// covers the OTHER way the same file reaches `downgradeToFree`: the wallet
// cannot pay. FR-095 asks for the downgrade path, and there are two ways in, so
// covering only the auto-renew switch would leave a caller free to regress one
// branch of it.
func TestExecuteScheduledRenewalDowngradeOnInsufficientFundsPublishesExactlyOnePlanChangeEvent(t *testing.T) {
	h := newRenewalHarness(t, true, 1000) // far below the plan price

	result := RenewalExecutionResult{}
	if err := h.uc.executeScheduledRenewal(context.Background(), h.subRepo.lockedSub.UserID, h.freePlanID, h.now, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != renewalStatusDowngraded {
		t.Fatalf("expected downgraded status, got %q", result.Status)
	}
	if h.events.creates != 1 {
		t.Fatalf("FR-095: one insufficient-funds downgrade must write exactly one plan-change history row, got %d", h.events.creates)
	}
	if len(h.publisher.events) != 1 {
		t.Fatalf("FR-095: one insufficient-funds downgrade must publish exactly one subscription.plan.changed, got %d", len(h.publisher.events))
	}
}

func (f *fakeRenewalSubRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}

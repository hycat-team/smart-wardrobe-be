package purchase

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---- Fakes (same-package tests construct only the fields the flow touches) ----

type fakeUow struct{}

func (f *fakeUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type fakePlanRepo struct {
	plansBySlug map[string]*entities.SubscriptionPlan
	// 025 T114: the canonical free plan, which `GetDefaultPlan` used to answer
	// with a hard-coded nil. Checkout never asks for it, so it stays unset in
	// every test below; the wallet purchase in `wallet.go` does ask, because it
	// has to resolve what plan the account is currently on before changing it.
	// Optional precisely so adding it changed nothing for those tests.
	defaultPlan *entities.SubscriptionPlan
}

func (f *fakePlanRepo) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return f.plansBySlug[slug], nil
}
func (f *fakePlanRepo) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return f.defaultPlan, nil
}
func (f *fakePlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlan, error) {
	if f.defaultPlan != nil && f.defaultPlan.ID == id {
		return f.defaultPlan, nil
	}
	for _, plan := range f.plansBySlug {
		if plan != nil && plan.ID == id {
			return plan, nil
		}
	}
	return nil, nil
}
func (f *fakePlanRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakePlanRepo) GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return f.plansBySlug[slug], nil
}

// GetBySlugsAnyStatus satisfies the interface added by 024 T150. Checkout never
// resolves a set of slugs, so this exists for the compiler rather than for a test;
// it answers the way the real repository does so a future caller is not misled.
func (f *fakePlanRepo) GetBySlugsAnyStatus(ctx context.Context, slugs []string) (map[string]*entities.SubscriptionPlan, error) {
	found := make(map[string]*entities.SubscriptionPlan, len(slugs))
	for _, slug := range slugs {
		if plan, ok := f.plansBySlug[slug]; ok && plan != nil {
			found[slug] = plan
		}
	}
	return found, nil
}
func (f *fakePlanRepo) ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakePlanRepo) GetAll(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakePlanRepo) Create(ctx context.Context, e *entities.SubscriptionPlan) error { return nil }
func (f *fakePlanRepo) Update(ctx context.Context, e *entities.SubscriptionPlan) error { return nil }
func (f *fakePlanRepo) Delete(ctx context.Context, id uuid.UUID) error                 { return nil }
func (f *fakePlanRepo) BatchUpdate(ctx context.Context, es []*entities.SubscriptionPlan) error {
	return nil
}
func (f *fakePlanRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeDepositTxRepo struct {
	active       *entities.DepositTransaction
	created      []*entities.DepositTransaction
	lockedByCode map[int64]*entities.DepositTransaction
	hasPending   bool
}

func (f *fakeDepositTxRepo) GetActiveDirectPurchase(ctx context.Context, userID uuid.UUID) (*entities.DepositTransaction, error) {
	return f.active, nil
}
func (f *fakeDepositTxRepo) HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error) {
	if f.hasPending {
		return true, nil
	}
	for _, tx := range f.lockedByCode {
		switch tx.Status {
		case depositstatus.Pending, depositstatus.Creating, depositstatus.ReconciliationRequired, depositstatus.Reconciling:
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeDepositTxRepo) Create(ctx context.Context, e *entities.DepositTransaction) error {
	f.created = append(f.created, e)
	if f.lockedByCode == nil {
		f.lockedByCode = map[int64]*entities.DepositTransaction{}
	}
	f.lockedByCode[e.OrderCode] = e
	return nil
}
func (f *fakeDepositTxRepo) GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	if tx, ok := f.lockedByCode[orderCode]; ok {
		return tx, nil
	}
	if f.active != nil && f.active.OrderCode == orderCode {
		return f.active, nil
	}
	return nil, nil
}
func (f *fakeDepositTxRepo) Update(ctx context.Context, e *entities.DepositTransaction) error {
	if f.lockedByCode == nil {
		f.lockedByCode = map[int64]*entities.DepositTransaction{}
	}
	for code, v := range f.lockedByCode {
		if v == e && code != e.OrderCode {
			delete(f.lockedByCode, code)
		}
	}
	f.lockedByCode[e.OrderCode] = e
	return nil
}
func (f *fakeDepositTxRepo) GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	if tx, ok := f.lockedByCode[orderCode]; ok {
		return tx, nil
	}
	if f.active != nil && f.active.OrderCode == orderCode {
		return f.active, nil
	}
	return nil, nil
}
func (f *fakeDepositTxRepo) GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTxRepo) ListAdmin(ctx context.Context, filter repositories.AdminDepositTransactionFilter, page, limit int) ([]*entities.DepositTransaction, int64, error) {
	return nil, 0, nil
}
func (f *fakeDepositTxRepo) ClaimReconciliationCandidates(ctx context.Context, limit int, lease time.Duration) ([]*repositories.ClaimedDepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTxRepo) UpdateWithToken(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
	return 1, nil
}
func (f *fakeDepositTxRepo) GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return nil, nil
}
func (f *fakeDepositTxRepo) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return nil, nil
}
func (f *fakeDepositTxRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTxRepo) GetAll(ctx context.Context) ([]*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTxRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeDepositTxRepo) BatchUpdate(ctx context.Context, es []*entities.DepositTransaction) error {
	return nil
}
func (f *fakeDepositTxRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeSubRepo struct {
	sub *entities.UserSubscription
}

func (f *fakeSubRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.sub, nil
}
func (f *fakeSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.sub, nil
}
func (f *fakeSubRepo) GetByUserIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) BulkCreate(ctx context.Context, subs []*entities.UserSubscription) error {
	return nil
}
func (f *fakeSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	return nil
}
func (f *fakeSubRepo) GetDistributionByPlanCode(ctx context.Context) ([]repositories.SubscriptionDistributionDTO, error) {
	return nil, nil
}
func (f *fakeSubRepo) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	return nil, 0, nil
}
func (f *fakeSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) GetAll(ctx context.Context) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeSubRepo) Create(ctx context.Context, e *entities.UserSubscription) error { return nil }
func (f *fakeSubRepo) Update(ctx context.Context, e *entities.UserSubscription) error { return nil }
func (f *fakeSubRepo) Delete(ctx context.Context, id uuid.UUID) error                 { return nil }
func (f *fakeSubRepo) BatchUpdate(ctx context.Context, es []*entities.UserSubscription) error {
	return nil
}
func (f *fakeSubRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeGateway struct {
	result    *payment.CheckoutSessionResult
	results   []*payment.CheckoutSessionResult
	calls     int
	requests  []*payment.CheckoutSessionReq
	cancelled []int64
	cancelErr error
}

func duplicateOrderResult() *payment.CheckoutSessionResult {
	return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, Retryable: true, ErrorCode: payment.ErrCodeDuplicateOrder}
}

func (f *fakeGateway) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutSessionReq) (*payment.CheckoutSessionResult, error) {
	f.calls++
	f.requests = append(f.requests, req)
	if len(f.results) > 0 {
		r := f.results[0]
		f.results = f.results[1:]
		if r.Outcome == payment.OutcomeSucceeded {
			return r, nil
		}
		return r, errors.New("payos returned business error code " + r.ErrorCode)
	}
	return f.result, nil
}
func (f *fakeGateway) VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) error {
	return nil
}
func (f *fakeGateway) GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
	return nil, nil
}
func (f *fakeGateway) CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error {
	f.cancelled = append(f.cancelled, orderCode)
	return f.cancelErr
}

// ---- Helpers ----

func checkoutTestConfig() config.Config {
	cfg := config.Config{}
	cfg.Database.TimeZone = "Asia/Ho_Chi_Minh"
	cfg.PayOS.ExpiredMinutes = 15
	cfg.PayOS.ReturnUrl = "https://app.example.com/return"
	cfg.PayOS.CancelUrl = "https://app.example.com/cancel"
	return cfg
}

func paidPlan() *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		AuditableEntity:    entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:               "premium",
		Name:               "Premium",
		IsActive:           true,
		Price:              decimal.NewFromInt(99000),
		TierRank:           1,
		DurationDays:       durationDaysPtr(30),
		MaxWardrobeItems:   100,
		MaxOutfits:         50,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
	}
}

func strPtr(s string) *string { return &s }

func durationDaysPtr(d int) *int { return &d }

func newCheckoutUseCase(cfg config.Config, planRepo *fakePlanRepo, txRepo *fakeDepositTxRepo, subRepo *fakeSubRepo, gateway *fakeGateway) *SubscriptionPurchaseUseCase {
	return &SubscriptionPurchaseUseCase{
		planRepo:       planRepo,
		depositTxRepo:  txRepo,
		userSubRepo:    subRepo,
		paymentGateway: gateway,
		uow:            &fakeUow{},
		cfg:            &cfg,
	}
}

// ---- Tests ----

func TestCreateDirectPurchaseSuccessCreatesPendingTransaction(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	txRepo := &fakeDepositTxRepo{}
	gateway := &fakeGateway{result: &payment.CheckoutSessionResult{
		Outcome:       payment.OutcomeSucceeded,
		CheckoutURL:   "https://pay.example.com/link",
		PaymentLinkID: "plink-1",
	}}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, gateway)

	res, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PaymentUrl != "https://pay.example.com/link" {
		t.Fatalf("expected checkout url, got %q", res.PaymentUrl)
	}
	if res.PaymentStatus != depositstatus.Pending {
		t.Fatalf("expected status pending, got %v", res.PaymentStatus)
	}
	if len(txRepo.created) != 1 {
		t.Fatalf("expected exactly one transaction created, got %d", len(txRepo.created))
	}
	if txRepo.created[0].PlanCode == nil || *txRepo.created[0].PlanCode != "premium" {
		t.Fatalf("expected persisted transaction linked to plan premium, got %+v", txRepo.created[0].PlanCode)
	}
	if !txRepo.created[0].Amount.Equal(plan.Price) {
		t.Fatalf("expected persisted amount to match plan price, got %s", txRepo.created[0].Amount)
	}
}

func TestCreateDirectPurchaseReusesActivePendingTransaction(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	future := time.Now().Add(10 * time.Minute)
	existing := &entities.DepositTransaction{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		OrderCode:       123456,
		Status:          depositstatus.Pending,
		PaymentUrl:      strPtr("https://pay.example.com/existing"),
		ExpiresAt:       &future,
	}
	txRepo := &fakeDepositTxRepo{active: existing}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, &fakeGateway{})

	res, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderCode != 123456 || res.PaymentUrl != "https://pay.example.com/existing" {
		t.Fatalf("expected existing pending link reused, got %+v", res)
	}
	if len(txRepo.created) != 0 {
		t.Fatalf("expected no new transaction, got %d", len(txRepo.created))
	}
}

func TestCreateDirectPurchaseLazyExpiresOutdatedTransaction(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	past := time.Now().Add(-10 * time.Minute)
	existing := &entities.DepositTransaction{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		OrderCode:       123456,
		Status:          depositstatus.Pending,
		PaymentUrl:      strPtr("https://pay.example.com/expired-link"),
		ExpiresAt:       &past,
	}
	txRepo := &fakeDepositTxRepo{active: existing}
	gateway := &fakeGateway{result: &payment.CheckoutSessionResult{
		Outcome:       payment.OutcomeSucceeded,
		CheckoutURL:   "https://pay.example.com/new-fresh-link",
		PaymentLinkID: "plink-new",
	}}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, gateway)

	res, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if existing.Status != depositstatus.Expired {
		t.Fatalf("expected old transaction to be marked Expired, got %v", existing.Status)
	}
	if res.PaymentUrl != "https://pay.example.com/new-fresh-link" {
		t.Fatalf("expected new fresh link created, got %q", res.PaymentUrl)
	}
	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 new transaction created, got %d", len(txRepo.created))
	}
}

func TestCreateDirectPurchaseRejectsFreePlan(t *testing.T) {
	cfg := checkoutTestConfig()
	freePlan := paidPlan()
	freePlan.Price = decimal.Zero
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"free": freePlan}}, &fakeDepositTxRepo{}, &fakeSubRepo{}, &fakeGateway{})

	_, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "free"})
	if err == nil {
		t.Fatal("expected free plan rejection, got nil error")
	}
}

func TestCreateDirectPurchaseRejectsUnknownPlan(t *testing.T) {
	cfg := checkoutTestConfig()
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{}}, &fakeDepositTxRepo{}, &fakeSubRepo{}, &fakeGateway{})

	_, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "ghost"})
	if err == nil {
		t.Fatal("expected unknown plan rejection, got nil error")
	}
}

func TestCreateDirectPurchaseRejectsWhenPendingExists(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	txRepo := &fakeDepositTxRepo{hasPending: true}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, &fakeGateway{})

	_, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err == nil {
		t.Fatal("expected pending payment rejection, got nil error")
	}
}

func TestCreateDirectPurchaseRetriesOnDuplicateOrderCode(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	txRepo := &fakeDepositTxRepo{}
	gateway := &fakeGateway{results: []*payment.CheckoutSessionResult{
		duplicateOrderResult(),
		{Outcome: payment.OutcomeSucceeded, CheckoutURL: "https://pay.example.com/retry-link", PaymentLinkID: "plink-retry"},
	}}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, gateway)

	res, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err != nil {
		t.Fatalf("expected retry to recover from 231, got error: %v", err)
	}
	if res.PaymentStatus != depositstatus.Pending || res.PaymentUrl != "https://pay.example.com/retry-link" {
		t.Fatalf("expected pending retry link, got %+v", res)
	}
	if gateway.calls != 2 {
		t.Fatalf("expected 2 gateway calls, got %d", gateway.calls)
	}
	first, second := gateway.requests[0].OrderCode, gateway.requests[1].OrderCode
	if first == second {
		t.Fatalf("expected rotated order code on retry, got same %d", first)
	}
	for _, code := range []int64{first, second} {
		if code < 1_000_000_000_000 {
			t.Fatalf("expected time-ordered order code, got %d", code)
		}
	}
	if len(txRepo.created) != 1 {
		t.Fatalf("expected exactly one transaction row (update-in-place), got %d", len(txRepo.created))
	}
	if res.OrderCode != second || txRepo.created[0].OrderCode != second {
		t.Fatalf("expected persisted order code %d, got res %+v created %+v", second, res, txRepo.created[0])
	}
}

func TestCreateDirectPurchaseConflictsWhenDuplicatePersists(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	txRepo := &fakeDepositTxRepo{}
	gateway := &fakeGateway{results: []*payment.CheckoutSessionResult{
		duplicateOrderResult(),
		duplicateOrderResult(),
		duplicateOrderResult(),
	}}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, gateway)

	_, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err == nil {
		t.Fatal("expected conflict after exhausted 231 retries, got nil error")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("expected HTTP 409 conflict, got %v", err)
	}
	if gateway.calls != 3 {
		t.Fatalf("expected 3 gateway calls, got %d", gateway.calls)
	}
	if txRepo.created[0].Status != depositstatus.CreationFailed {
		t.Fatalf("expected exhausted duplicate marked CreationFailed, got %v", txRepo.created[0].Status)
	}
}

func shortLivedPendingTransaction() *entities.DepositTransaction {
	soon := time.Now().Add(3 * time.Minute)
	return &entities.DepositTransaction{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		OrderCode:       654321,
		Status:          depositstatus.Pending,
		PaymentUrl:      strPtr("https://pay.example.com/old-link"),
		PaymentLinkID:   strPtr("plink-old"),
		ExpiresAt:       &soon,
	}
}

func TestCreateDirectPurchaseRotatesShortLivedLink(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	old := shortLivedPendingTransaction()
	txRepo := &fakeDepositTxRepo{active: old}
	gateway := &fakeGateway{result: &payment.CheckoutSessionResult{
		Outcome:       payment.OutcomeSucceeded,
		CheckoutURL:   "https://pay.example.com/fresh-link",
		PaymentLinkID: "plink-fresh",
	}}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, gateway)

	res, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err != nil {
		t.Fatalf("expected rotation to fresh link, got error: %v", err)
	}
	if res.PaymentUrl != "https://pay.example.com/fresh-link" {
		t.Fatalf("expected fresh link, got %q", res.PaymentUrl)
	}
	if res.OrderCode == 654321 {
		t.Fatalf("expected new order code, got reused %d", res.OrderCode)
	}
	if len(gateway.cancelled) != 1 || gateway.cancelled[0] != 654321 {
		t.Fatalf("expected old link 654321 cancelled, got %v", gateway.cancelled)
	}
	if old.Status != depositstatus.Expired {
		t.Fatalf("expected old transaction Expired, got %v", old.Status)
	}
	if len(txRepo.created) != 1 {
		t.Fatalf("expected exactly one new transaction, got %d", len(txRepo.created))
	}
}

func TestCreateDirectPurchaseFailsSafeWhenCancelFails(t *testing.T) {
	cfg := checkoutTestConfig()
	plan := paidPlan()
	old := shortLivedPendingTransaction()
	txRepo := &fakeDepositTxRepo{active: old}
	gateway := &fakeGateway{
		result: &payment.CheckoutSessionResult{
			Outcome:     payment.OutcomeSucceeded,
			CheckoutURL: "https://pay.example.com/should-not-happen",
		},
		cancelErr: errors.New("payos cancel boom"),
	}
	uc := newCheckoutUseCase(cfg, &fakePlanRepo{plansBySlug: map[string]*entities.SubscriptionPlan{"premium": plan}}, txRepo, &fakeSubRepo{}, gateway)

	res, err := uc.CreateDirectPurchase(context.Background(), uuid.New(), &dto.DirectPurchaseReq{PlanSlug: "premium"})
	if err != nil {
		t.Fatalf("expected fail-safe old link, got error: %v", err)
	}
	if res.OrderCode != 654321 || res.PaymentUrl != "https://pay.example.com/old-link" {
		t.Fatalf("expected old link returned untouched, got %+v", res)
	}
	if len(txRepo.created) != 0 {
		t.Fatalf("expected no new transaction when cancel fails, got %d", len(txRepo.created))
	}
	if gateway.calls != 0 {
		t.Fatalf("expected no checkout call when cancel fails, got %d", gateway.calls)
	}
}

func (f *fakeDepositTxRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.DepositTransaction, error) {
	return nil, nil
}

func (f *fakeSubRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}

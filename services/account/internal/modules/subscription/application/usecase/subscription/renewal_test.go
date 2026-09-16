package subscription

import (
	"context"
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
func (f *fakeRenewalSubRepo) BulkCreate(ctx context.Context, subs []*entities.UserSubscription) error {
	return nil
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
	return nil
}
func (f *fakeRenewalEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	return nil, nil
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
	h.uc = &SubscriptionUseCase{
		uow:                &fakeRenewalUow{},
		userSubRepo:        h.subRepo,
		walletRepo:         h.walletRepo,
		statementRepo:      h.statements,
		planRepo:           &fakeRenewalPlanRepo{planByID: map[uuid.UUID]*entities.SubscriptionPlan{freePlanID: {Slug: "free", IsActive: true}}},
		eventRepo:          h.events,
		renewalAttemptRepo: h.attempt,
		notifContract:      h.notif,
		log:                zap.NewNop(),
	}
	return h
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

func (f *fakeRenewalSubRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}

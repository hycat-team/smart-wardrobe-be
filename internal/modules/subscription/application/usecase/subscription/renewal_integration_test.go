package subscription

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/internal/shared/infrastructure/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TestIntegrationScheduledRenewalRenewsPaidPlan proves the renewal workflow
// against a real Postgres: wallet debit, subscription extension, statement and
// renewal attempt rows must all be persisted (spec 019 FR-004/FR-014).
func TestIntegrationScheduledRenewalRenewsPaidPlan(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	walletRepo := persistence.NewUserWalletRepository(dbConn)
	statementRepo := persistence.NewWalletStatementRepository(dbConn)
	eventRepo := persistence.NewUserSubscriptionEventRepository(dbConn)
	attemptRepo := persistence.NewSubscriptionRenewalAttemptRepository(dbConn)

	ctx := context.Background()
	now := time.Now()
	plan := &entities.SubscriptionPlan{Slug: "premium-renewal", Name: "Premium Renewal", IsActive: true, Price: decimal.NewFromInt(99000), TierRank: 1, PlanKind: plankind.Finite, DurationDays: durationDaysPtr2(30), AiOutfitDailyQuota: 10, AiChatDailyQuota: 20}
	if err := planRepo.Create(ctx, plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	userID := uuid.New()
	expired := now.Add(-time.Hour)
	sub := &entities.UserSubscription{UserID: userID, SubscriptionPlanID: plan.ID, CurrentPlanCode: plan.Slug, CurrentTierRank: plan.TierRank, CurrentPlanKind: plankind.Finite, IsAutoRenewEnabled: true, ExpiresAt: &expired, StartedAt: now}
	if err := subRepo.Create(ctx, sub); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	wallet := &entities.UserWallet{UserID: userID, Balance: decimal.NewFromInt(500000), Currency: currency.VND}
	if err := walletRepo.Create(ctx, wallet); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	uc := &SubscriptionUseCase{
		uow:                db.NewGormUnitOfWork(dbConn),
		userSubRepo:        subRepo,
		walletRepo:         walletRepo,
		statementRepo:      statementRepo,
		planRepo:           planRepo,
		eventRepo:          eventRepo,
		renewalAttemptRepo: attemptRepo,
		notifContract:      &integrationRenewalNotif{},
		log:                zap.NewNop(),
	}

	result := RenewalExecutionResult{}
	if err := uc.executeScheduledRenewal(ctx, userID, uuid.New(), now, &result); err != nil {
		t.Fatalf("renewal failed: %v", err)
	}
	if result.Status != renewalStatusRenewed {
		t.Fatalf("expected renewed status, got %q", result.Status)
	}

	updated, err := subRepo.GetByUserID(ctx, userID)
	if err != nil || updated == nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if updated.ExpiresAt == nil || !updated.ExpiresAt.After(now) {
		t.Fatalf("expected subscription extended past now, got %v", updated.ExpiresAt)
	}
	walletAfter, err := walletRepo.GetByUserID(ctx, userID)
	if err != nil || walletAfter == nil {
		t.Fatalf("reload wallet: %v", err)
	}
	if !walletAfter.Balance.Equal(decimal.NewFromInt(401000)) {
		t.Fatalf("expected wallet debited to 401000, got %s", walletAfter.Balance)
	}
}

// TestIntegrationScheduledRenewalDowngradesWithoutFunds proves the fallback
// path persists a downgrade to the default free plan when the wallet cannot
// cover the renewal price.
func TestIntegrationScheduledRenewalDowngradesWithoutFunds(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	walletRepo := persistence.NewUserWalletRepository(dbConn)
	statementRepo := persistence.NewWalletStatementRepository(dbConn)
	eventRepo := persistence.NewUserSubscriptionEventRepository(dbConn)
	attemptRepo := persistence.NewSubscriptionRenewalAttemptRepository(dbConn)

	ctx := context.Background()
	now := time.Now()
	freePlan := &entities.SubscriptionPlan{Slug: "free-downgrade", Name: "Free", IsActive: true, Price: decimal.Zero, PlanKind: plankind.DefaultFree}
	if err := planRepo.Create(ctx, freePlan); err != nil {
		t.Fatalf("seed free plan: %v", err)
	}
	plan := &entities.SubscriptionPlan{Slug: "premium-broke", Name: "Premium", IsActive: true, Price: decimal.NewFromInt(99000), TierRank: 1, PlanKind: plankind.Finite, DurationDays: durationDaysPtr2(30)}
	if err := planRepo.Create(ctx, plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	userID := uuid.New()
	expired := now.Add(-time.Hour)
	sub := &entities.UserSubscription{UserID: userID, SubscriptionPlanID: plan.ID, CurrentPlanCode: plan.Slug, CurrentTierRank: plan.TierRank, CurrentPlanKind: plankind.Finite, IsAutoRenewEnabled: true, ExpiresAt: &expired, StartedAt: now}
	if err := subRepo.Create(ctx, sub); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	if err := walletRepo.Create(ctx, &entities.UserWallet{UserID: userID, Balance: decimal.NewFromInt(500), Currency: currency.VND}); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	uc := &SubscriptionUseCase{
		uow:                db.NewGormUnitOfWork(dbConn),
		userSubRepo:        subRepo,
		walletRepo:         walletRepo,
		statementRepo:      statementRepo,
		planRepo:           planRepo,
		eventRepo:          eventRepo,
		renewalAttemptRepo: attemptRepo,
		notifContract:      &integrationRenewalNotif{},
		log:                zap.NewNop(),
	}

	result := RenewalExecutionResult{}
	if err := uc.executeScheduledRenewal(ctx, userID, freePlan.ID, now, &result); err != nil {
		t.Fatalf("renewal downgrade failed: %v", err)
	}
	if result.Status != renewalStatusDowngraded {
		t.Fatalf("expected downgraded status, got %q", result.Status)
	}
	updated, err := subRepo.GetByUserID(ctx, userID)
	if err != nil || updated == nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if updated.CurrentPlanKind != plankind.DefaultFree {
		t.Fatalf("expected subscription downgraded to free plan, got %v", updated.CurrentPlanKind)
	}
}

func openSubscriptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping subscription integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	if err := dbConn.AutoMigrate(
		&entities.SubscriptionPlan{},
		&entities.UserSubscription{},
		&entities.UserDailyQuota{},
		&entities.UserWallet{},
		&entities.DepositTransaction{},
		&entities.WalletStatement{},
		&entities.UserSubscriptionEvent{},
		&entities.SubscriptionRenewalAttempt{},
		&entities.ProviderPaymentEvent{},
		&entities.ProviderWebhookInbox{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return dbConn
}

func cleanupSubscriptionTables(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	for _, model := range []any{
		&entities.SubscriptionRenewalAttempt{},
		&entities.UserSubscriptionEvent{},
		&entities.WalletStatement{},
		&entities.DepositTransaction{},
		&entities.ProviderPaymentEvent{},
		&entities.ProviderWebhookInbox{},
		&entities.UserDailyQuota{},
		&entities.UserSubscription{},
		&entities.UserWallet{},
		&entities.SubscriptionPlan{},
	} {
		if err := dbConn.Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("cleanup table: %v", err)
		}
	}
}

type integrationRenewalNotif struct{}

func (f *integrationRenewalNotif) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	return nil
}
func (f *integrationRenewalNotif) SendByUserID(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, userID uuid.UUID) error {
	return nil
}

func durationDaysPtr2(d int) *int { return &d }

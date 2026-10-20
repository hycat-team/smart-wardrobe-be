package subscription

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/shared/infrastructure/db"

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
	plan := &entities.SubscriptionPlan{Slug: renewalPremiumPlanSlug, Name: "Premium Renewal", IsActive: true, Price: decimal.NewFromInt(99000), TierRank: 1, PlanKind: plankind.Finite, DurationDays: durationDaysPtr2(30), AiOutfitDailyQuota: 10, AiChatDailyQuota: 20}
	if err := planRepo.Create(ctx, plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	userID := renewalUserID
	ensureSubscriptionUser(t, dbConn, userID, "renewing")
	ensureSubscriptionUser(t, dbConn, userID, "renewing")
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
		// See the note on the other renewal test: the publisher is a real
		// dependency of this path since 024 T048, and leaving it nil panicked only
		// when the integration suite ran against a real database.
		planChangePublisher: NewPlanChangePublisher(eventRepo, &recordingPublisher{}, noopPlanPeriodRecorder{}),
		log:                 zap.NewNop(),
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
	// Reuse the seeded default free plan: ux_single_active_default_free_plan
	// refuses a second active one, which the AutoMigrate schema never expressed.
	freePlan, err := planRepo.GetDefaultPlan(ctx)
	if err != nil || freePlan == nil {
		t.Fatalf("default free plan missing from test database (apply database/init/03-seed.sql): %v", err)
	}
	plan := &entities.SubscriptionPlan{Slug: renewalBrokePlanSlug, Name: "Premium", IsActive: true, Price: decimal.NewFromInt(99000), TierRank: 1, PlanKind: plankind.Finite, DurationDays: durationDaysPtr2(30)}
	if err := planRepo.Create(ctx, plan); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	userID := renewalBrokeUserID
	ensureSubscriptionUser(t, dbConn, userID, "broke")
	ensureSubscriptionUser(t, dbConn, userID, "broke")
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
		// The renewal path publishes its plan change through the shared publisher
		// (024 T048), so it has to be wired here. It was missing, and the test still
		// passed in CI because `TEST_DATABASE_URL` is normally unset - the nil
		// dependency only ever surfaced as a panic for whoever ran the integration
		// suite locally against a real database.
		planChangePublisher: NewPlanChangePublisher(eventRepo, &recordingPublisher{}, noopPlanPeriodRecorder{}),
		log:                 zap.NewNop(),
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

// Fixture identifiers are fixed so a crashed run's leftovers can be found and
// removed without touching rows that are not ours (026 D-06, same rule as
// 025/D-02 for the webhook harness).
var (
	renewalUserID      = uuid.MustParse("a0000000-0000-4000-8000-000000000032")
	renewalBrokeUserID = uuid.MustParse("a0000000-0000-4000-8000-000000000033")
)

const (
	renewalPremiumPlanSlug = "premium-renewal"
	renewalFreePlanSlug    = "free-downgrade"
	renewalBrokePlanSlug   = "premium-broke"
)

// subscriptionTestSearchPath mirrors configs/config.yaml. Repositories use
// unqualified table names; without the setting every query fails with
// "relation does not exist" even though the schema is correct.
const subscriptionTestSearchPath = "public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,community"

func subscriptionTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if strings.Contains(dsn, "search_path=") {
		return dsn
	}
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		q := u.Query()
		q.Set("search_path", subscriptionTestSearchPath)
		u.RawQuery = q.Encode()
		return u.String()
	}
	return dsn + " search_path=" + subscriptionTestSearchPath
}

// openSubscriptionTestDB requires the schema the repository builds — init scripts
// then the per-service migrations — instead of authoring it with AutoMigrate.
//
// AutoMigrate used to run here, and because GORM emits unqualified table names it
// created `public.subscription_plans`, `public.user_subscriptions`… next to the
// real `subscription.*` tables. With `public` first on the search_path those
// copies SHADOWED the migrated schema for every package that ran afterwards in
// the same database: reads landed on empty `public` copies and later suites failed
// for a reason unrelated to what they tested. Requiring the migrated schema is the
// same fix 025/D-01 applied to the webhook harness.
func openSubscriptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping subscription integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(subscriptionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// One pool per `gorm.Open`; close it when the test ends, or repeated runs and
	// parallel packages exhaust Postgres connections.
	t.Cleanup(func() {
		if sqlDB, dbErr := dbConn.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	var schemaReady bool
	if err := dbConn.Raw(`
		SELECT to_regclass('subscription.subscription_plans') IS NOT NULL
		   AND to_regclass('subscription.user_subscriptions') IS NOT NULL
		   AND to_regclass('shared.outbox_records') IS NOT NULL
		   AND to_regclass('identity.users') IS NOT NULL
		   AND to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&schemaReady).Error; err != nil {
		t.Fatalf("probe test schema: %v", err)
	}
	if !schemaReady {
		t.Fatalf("TEST_DATABASE_URL is not a migration-built database: apply database/init/*.sql then " +
			"`make migrate-up svc=account DB_NAME=<that database>` first (see openSubscriptionTestDB)")
	}
	return dbConn
}

// ensureSubscriptionUser inserts the fixture account these tests renew.
// `user_subscriptions.user_id` is a FOREIGN KEY onto `identity.users(id)` in the
// migrated schema, so a bare uuid.New() would be refused.
func ensureSubscriptionUser(t *testing.T, dbConn *gorm.DB, id uuid.UUID, label string) {
	t.Helper()
	// The handle carries the id so a second run against the same database cannot
	// collide on `idx_users_email_lower`: these fixtures are keyed by a fresh uuid
	// each run, so a fixed email would make the harness single-use.
	handle := "renewal-int-" + label + "-" + id.String()
	if err := dbConn.Exec(`
		INSERT INTO identity.users (id, username, email, password_hash, role_slug)
		VALUES (?, ?, ?, 'fixture-not-a-real-hash', 'user')
		ON CONFLICT (id) DO NOTHING`, id, handle, handle+"@example.test").Error; err != nil {
		t.Fatalf("ensure fixture user %s: %v", id, err)
	}
}

// cleanupSubscriptionTables removes only rows this file creates: keyed on the two
// fixture user ids and the three fixture plan slugs. The old implementation used
// `WHERE 1 = 1`, which emptied whole tables — including the seeded default free
// plan other suites depend on.
func cleanupSubscriptionTables(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	statements := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM subscription.subscription_renewal_attempts
			WHERE user_id IN (?, ?)
			   OR expected_plan_id IN (
			        SELECT id FROM subscription.subscription_plans WHERE slug IN (?, ?, ?))`,
			[]any{renewalUserID, renewalBrokeUserID, renewalPremiumPlanSlug, renewalFreePlanSlug, renewalBrokePlanSlug}},
		{`DELETE FROM subscription.user_subscription_events WHERE user_id IN (?, ?)`,
			[]any{renewalUserID, renewalBrokeUserID}},
		{`DELETE FROM subscription.wallet_statements WHERE user_id IN (?, ?)`,
			[]any{renewalUserID, renewalBrokeUserID}},
		{`DELETE FROM subscription.deposit_transactions WHERE user_id IN (?, ?)`,
			[]any{renewalUserID, renewalBrokeUserID}},
		{`DELETE FROM subscription.user_daily_quotas WHERE user_id IN (?, ?)`,
			[]any{renewalUserID, renewalBrokeUserID}},
		{`DELETE FROM subscription.user_subscriptions
			WHERE user_id IN (?, ?)
			   OR subscription_plan_id IN (
			        SELECT id FROM subscription.subscription_plans WHERE slug IN (?, ?, ?))`,
			[]any{renewalUserID, renewalBrokeUserID, renewalPremiumPlanSlug, renewalFreePlanSlug, renewalBrokePlanSlug}},
		{`DELETE FROM subscription.user_wallets WHERE user_id IN (?, ?)`,
			[]any{renewalUserID, renewalBrokeUserID}},
		{`DELETE FROM subscription.subscription_plans WHERE slug IN (?, ?, ?)`,
			[]any{renewalPremiumPlanSlug, renewalFreePlanSlug, renewalBrokePlanSlug}},
	}
	for _, stmt := range statements {
		if err := dbConn.Exec(stmt.sql, stmt.args...).Error; err != nil {
			t.Fatalf("cleanup %q: %v", stmt.sql, err)
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

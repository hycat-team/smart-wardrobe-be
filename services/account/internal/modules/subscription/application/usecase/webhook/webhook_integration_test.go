package webhook

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	ucInterfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/webhookprocessingstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	"smart-wardrobe-be/shared/infrastructure/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ---- Fakes ----

type integrationGateway struct{}

func (f *integrationGateway) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutSessionReq) (*payment.CheckoutSessionResult, error) {
	return nil, nil
}
func (f *integrationGateway) VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) error {
	return nil
}
func (f *integrationGateway) GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
	return nil, nil
}
func (f *integrationGateway) CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error {
	return nil
}

type integrationNotif struct{}

func (f *integrationNotif) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	return nil
}
func (f *integrationNotif) SendByUserID(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, userID uuid.UUID) error {
	return nil
}

type integrationBrief struct{}

func (f *integrationBrief) GetByID(ctx context.Context, userID uuid.UUID) (*identity_dto.UserBriefContractDTO, error) {
	return &identity_dto.UserBriefContractDTO{ID: userID, Email: "user@example.com"}, nil
}
func (f *integrationBrief) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}
func (f *integrationBrief) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}
func (f *integrationBrief) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}

// ---- Helpers ----

// webhookTestUserID is fixed on purpose. 025 D-02 requires cleanup to be scoped
// to this file's own rows; a random id per run would make a crashed run's
// leftovers impossible to find without touching rows that are not ours.
var webhookTestUserID = uuid.MustParse("a0000000-0000-4000-8000-000000000030")

const (
	webhookPremiumPlanSlug  = "premium-webhook"
	webhookUnknownOrderCode = int64(123)
)

// webhookTestSearchPath mirrors configs/config.yaml `database.search_path`.
// Migrations place the tables in module schemas while the repositories use
// unqualified names, exactly like production, where the server-side search_path
// resolves them (FR-046). A bare TEST_DATABASE_URL has no such setting, so the
// harness supplies it; without it every query fails with "relation does not
// exist" even though the schema is correct.
const webhookTestSearchPath = "public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,community"

// webhookTestDSN adds the search_path to TEST_DATABASE_URL when the caller did
// not already set one (a caller that sets it explicitly keeps control).
func webhookTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if strings.Contains(dsn, "search_path=") {
		return dsn
	}
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		q := u.Query()
		q.Set("search_path", webhookTestSearchPath)
		u.RawQuery = q.Encode()
		return u.String()
	}
	return dsn + " search_path=" + webhookTestSearchPath
}

// openWebhookTestDB requires the schema to be the one this repository builds
// (025 D-01). It used to call GORM AutoMigrate, which derived tables from the Go
// structs and therefore produced a schema that only RESEMBLED production: it
// created `uni_subscription_plans_slug` where the migration creates
// `subscription_plans_slug_key`, and it cannot express the CHECK constraints or
// partial indexes the migrations carry. Evidence gathered on that schema was
// evidence about a different database.
//
// Instead of rebuilding the schema here, this harness REQUIRES a database built
// the documented way — `database/init/*.sql` (the project's base schema) followed
// by the Account migrations:
//
//	psql "$TEST_DATABASE_URL" -f database/init/00-init.sql   (then 01/02/03)
//	make migrate-up svc=account DB_NAME=<the same database>
//
// The guard below turns a wrong database into one actionable failure instead of
// a cascade of confusing ones. It is deliberately not AutoMigrate: the point of
// the fix is that the test can no longer author its own schema.
func openWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping payment webhook integration test")
	}
	gin.SetMode(gin.TestMode)
	dbConn, err := gorm.Open(postgres.Open(webhookTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	var schemaReady bool
	if err := dbConn.Raw(`
		SELECT to_regclass('subscription.subscription_plans') IS NOT NULL
		   AND to_regclass('subscription.deposit_transactions') IS NOT NULL
		   AND to_regclass('identity.users') IS NOT NULL
		   AND to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&schemaReady).Error; err != nil {
		t.Fatalf("probe test schema: %v", err)
	}
	if !schemaReady {
		t.Fatalf("TEST_DATABASE_URL is not a migration-built database: apply database/init/*.sql then " +
			"`make migrate-up svc=account DB_NAME=<that database>` first (see openWebhookTestDB)")
	}

	var constraintName string
	if err := dbConn.Raw(`
		SELECT conname FROM pg_constraint
		WHERE conrelid = 'subscription.subscription_plans'::regclass
		  AND contype = 'u'
		  AND conname = 'subscription_plans_slug_key'`).Scan(&constraintName).Error; err != nil {
		t.Fatalf("probe plan slug constraint: %v", err)
	}
	if constraintName == "" {
		t.Fatalf("test schema carries the AutoMigrate shape (`uni_subscription_plans_slug`), not the " +
			"migration shape (`subscription_plans_slug_key`); recreate the database from database/init + migrations")
	}
	return dbConn
}

// ensureWebhookUser inserts the fixed fixture user. The migration-built schema
// carries the production FKs, so user_subscriptions.user_id needs a real
// identity.users row (AutoMigrate never enforced that).
func ensureWebhookUser(t *testing.T, dbConn *gorm.DB, id uuid.UUID, username, email string) {
	t.Helper()
	if err := dbConn.Exec(`
		INSERT INTO identity.users (id, username, email, password_hash, role_slug)
		VALUES (?, ?, ?, 'fixture-not-a-real-hash', 'user')
		ON CONFLICT (id) DO NOTHING`, id, username, email).Error; err != nil {
		t.Fatalf("ensure fixture user %s: %v", id, err)
	}
}

// cleanupWebhookTables removes only rows this file creates (025 D-02). The old
// implementation deleted with `WHERE 1 = 1`, which emptied whole tables: that is
// why running it needed an explicit consent gate, and that gate is what let the
// Brevo incident go unnoticed. Every statement below is keyed on a fixture value
// (the fixed user id, the fixture plan slugs, or the fixture order codes), so a
// shared database keeps every row that is not ours.
func cleanupWebhookTables(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	const (
		sqlDeleteEvents = `DELETE FROM subscription.user_subscription_events
			WHERE user_id = ?
			   OR source_deposit_transaction_id IN (
			        SELECT id FROM subscription.deposit_transactions WHERE order_code = ?)`
		sqlDeleteStatements = `DELETE FROM subscription.wallet_statements WHERE user_id = ?`
		sqlDeleteDeposits   = `DELETE FROM subscription.deposit_transactions WHERE order_code = ?`
		sqlDeletePayEvents  = `DELETE FROM subscription.provider_payment_events WHERE order_code IN (?, ?)`
		sqlDeleteInbox      = `DELETE FROM subscription.provider_webhook_inbox WHERE order_code IN (?, ?)`
		sqlDeleteQuotas     = `DELETE FROM subscription.user_daily_quotas WHERE user_id = ?`
		sqlDeleteRenewals   = `DELETE FROM subscription.subscription_renewal_attempts
			WHERE user_id = ?
			   OR expected_plan_id IN (
			        SELECT id FROM subscription.subscription_plans WHERE slug = ?)`
		sqlDeleteSubs = `DELETE FROM subscription.user_subscriptions
			WHERE user_id = ?
			   OR subscription_plan_id IN (
			        SELECT id FROM subscription.subscription_plans WHERE slug = ?)`
		sqlDeleteWallets = `DELETE FROM subscription.user_wallets WHERE user_id = ?`
		sqlDeletePlans   = `DELETE FROM subscription.subscription_plans WHERE slug = ?`
	)

	stmts := []struct {
		sql  string
		args []any
	}{
		{sqlDeleteEvents, []any{webhookTestUserID, webhookOrderCode}},
		{sqlDeleteStatements, []any{webhookTestUserID}},
		{sqlDeleteDeposits, []any{webhookOrderCode}},
		{sqlDeletePayEvents, []any{webhookOrderCode, webhookUnknownOrderCode}},
		{sqlDeleteInbox, []any{webhookOrderCode, webhookUnknownOrderCode}},
		{sqlDeleteQuotas, []any{webhookTestUserID}},
		{sqlDeleteRenewals, []any{webhookTestUserID, webhookPremiumPlanSlug}},
		{sqlDeleteSubs, []any{webhookTestUserID, webhookPremiumPlanSlug}},
		{sqlDeleteWallets, []any{webhookTestUserID}},
		{sqlDeletePlans, []any{webhookPremiumPlanSlug}},
	}
	for _, stmt := range stmts {
		if err := dbConn.Exec(stmt.sql, stmt.args...).Error; err != nil {
			t.Fatalf("cleanup %q: %v", stmt.sql, err)
		}
	}
}

const webhookOrderCode = int64(777001)

func webhookPtr[T any](v T) *T { return &v }

func seedWebhookScenario(t *testing.T, dbConn *gorm.DB) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	txRepo := persistence.NewDepositTransactionRepository(dbConn)

	// The default free plan is seeded by database/init/03-seed.sql. Reuse it: the
	// migration-built schema carries `ux_single_active_default_free_plan`, a
	// partial unique index the AutoMigrate schema never expressed, so a second
	// active free plan cannot exist. That is the constraint doing its job.
	freePlan, err := planRepo.GetDefaultPlan(ctx)
	if err != nil || freePlan == nil {
		t.Fatalf("default free plan missing from test database (apply database/init/03-seed.sql): %v", err)
	}
	premiumPlan := &entities.SubscriptionPlan{Slug: webhookPremiumPlanSlug, Name: "Premium Webhook", IsActive: true, Price: decimal.NewFromInt(99000), TierRank: 1, PlanKind: plankind.Finite, DurationDays: webhookPtr(30), AiOutfitDailyQuota: 10, AiChatDailyQuota: 20}
	if err := planRepo.Create(ctx, premiumPlan); err != nil {
		t.Fatalf("seed premium plan: %v", err)
	}
	userID := webhookTestUserID
	ensureWebhookUser(t, dbConn, userID, "webhook-int-user", "webhook-int-user@example.com")
	sub := &entities.UserSubscription{UserID: userID, SubscriptionPlanID: freePlan.ID, CurrentPlanCode: freePlan.Slug, CurrentPlanKind: plankind.DefaultFree, StartedAt: time.Now().UTC()}
	if err := subRepo.Create(ctx, sub); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(15 * time.Minute)
	tx := &entities.DepositTransaction{
		UserID:                userID,
		Amount:                decimal.NewFromInt(99000),
		ExpectedAmount:        decimal.NewFromInt(99000),
		Currency:              currency.VND,
		Status:                depositstatus.Pending,
		TransactionType:       deposittransactiontype.DirectPurchase,
		SubscriptionPlanID:    &premiumPlan.ID,
		PlanCode:              webhookPtr(webhookPremiumPlanSlug),
		PlanName:              webhookPtr("Premium Webhook"),
		TierRank:              webhookPtr(1),
		PlanKind:              webhookPtr(plankind.Finite),
		PurchasedDurationDays: webhookPtr(30),
		BenefitSnapshot:       entities.JSONDocument(`{}`),
		ExpiresAt:             &expiresAt,
		OrderCode:             webhookOrderCode,
	}
	if err := txRepo.Create(ctx, tx); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
	// `DepositTransaction.Status` carries `gorm:"...;default:3"`, and GORM omits a
	// field from the INSERT when its value is the type's zero value and the column
	// declares a default. `depositstatus.Pending` IS zero, so the status set above
	// never reached the database and the row landed as `Creating` instead.
	//
	// That made this fixture lie about its own starting state, and the test below -
	// which asserts an unsuccessful webhook changes nothing - failed against a
	// status it never actually seeded. Production code always creates transactions as
	// `Creating`, a non-zero value, so it is unaffected; only a fixture that wants
	// `Pending` has to say so out of band.
	//
	// The update is deliberate rather than a `Select("*")` on the create: it makes
	// the intent "this row starts as Pending" explicit at the point of reading.
	if err := dbConn.Model(&entities.DepositTransaction{}).
		Where("order_code = ?", webhookOrderCode).
		Update("status", depositstatus.Pending).Error; err != nil {
		t.Fatalf("force seeded transaction to Pending: %v", err)
	}
	return userID
}

func buildWebhookPayload(orderCode int64, amount string) []byte {
	return []byte(`{"code":"00","desc":"success","success":true,"data":{"orderCode":` + strconv.FormatInt(orderCode, 10) +
		`,"amount":"` + amount + `","description":"premium","accountNumber":"123","reference":"ref-` +
		strconv.FormatInt(orderCode, 10) + `","transactionDateTime":"2026-09-06T10:00:00+07:00","currency":"VND","paymentLinkId":"plink-` +
		strconv.FormatInt(orderCode, 10) + `","code":"00","desc":"success"},"signature":"ignored-by-fake"}`)
}

// newWebhookUseCase wires the use case over the real repositories AND hands back
// the recording publisher, so a test can count what the publish step emitted.
//
// The publisher used to be created and dropped inside this helper, which meant no
// test in this file could observe it. Returning it is what lets T116 assert a
// COUNT of published messages rather than the absence of an error.
func newWebhookUseCase(dbConn *gorm.DB) (ucInterfaces.IPaymentWebhookUseCase, *integrationPublisher) {
	cfg := config.Config{}
	cfg.Database.TimeZone = "Asia/Ho_Chi_Minh"
	// 024 T051: the settlement path now publishes through the single helper, so
	// the integration fixture wires the real publisher against the real
	// repositories rather than a stub — that is what makes this test able to
	// catch a regression in the publish step.
	publisher := &integrationPublisher{}
	planChangePublisher := subscription.NewPlanChangePublisher(
		persistence.NewUserSubscriptionEventRepository(dbConn),
		publisher,
	)
	return NewPaymentWebhookUseCase(
		&cfg,
		persistence.NewUserWalletRepository(dbConn),
		persistence.NewDepositTransactionRepository(dbConn),
		persistence.NewWalletStatementRepository(dbConn),
		persistence.NewSubscriptionPlanRepository(dbConn),
		persistence.NewUserSubscriptionRepository(dbConn),
		&integrationGateway{},
		db.NewGormUnitOfWork(dbConn),
		persistence.NewPaymentEventRepository(dbConn),
		persistence.NewWebhookInboxRepository(dbConn),
		persistence.NewUserSubscriptionEventRepository(dbConn),
		&integrationNotif{},
		&integrationBrief{},
		zap.NewNop(),
		nil,
		planChangePublisher,
	), publisher
}

// integrationPublisher records published events without touching a broker.
//
// 025 T116: the topic is recorded alongside the payload. Asserting only the
// payload count would pass even if the message went out on the wrong topic,
// which is a different failure from the one FR-095 names.
type integrationPublisher struct {
	events []publishedWebhookEvent
}

// publishedWebhookEvent pairs the topic a message went to with its payload.
type publishedWebhookEvent struct {
	topic   string
	payload any
}

func (p *integrationPublisher) Publish(_ context.Context, topic string, payload any) error {
	p.events = append(p.events, publishedWebhookEvent{topic: topic, payload: payload})
	return nil
}

// publishedWebhookEventType reads `eventType` out of a captured payload. The
// payload struct is unexported in the mapper package by design (constitution §II:
// the JSON is the contract), so going through JSON asserts on what the receiver
// would actually receive.
func publishedWebhookEventType(t *testing.T, payload any) string {
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

// webhookPlanChangeHistory reads back the plan-change history this file's
// cleanup emptied, ordered so a failure message names the row it found.
func webhookPlanChangeHistory(t *testing.T, dbConn *gorm.DB) []entities.UserSubscriptionEvent {
	t.Helper()
	var rows []entities.UserSubscriptionEvent
	if err := dbConn.Order("occurred_at asc").Find(&rows).Error; err != nil {
		t.Fatalf("read plan-change history: %v", err)
	}
	return rows
}

// ---- Tests ----

func TestIntegrationProcessWebhookAppliesDirectPurchaseIdempotently(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	userID := seedWebhookScenario(t, dbConn)
	uc, _ := newWebhookUseCase(dbConn)
	ctx := context.Background()

	payload := buildWebhookPayload(webhookOrderCode, "99000")
	if err := uc.ProcessWebhook(ctx, payload, "sig"); err != nil {
		t.Fatalf("first webhook processing failed: %v", err)
	}

	// Transaction marked success with gateway reference.
	txRepo := persistence.NewDepositTransactionRepository(dbConn)
	tx, err := txRepo.GetByOrderCode(ctx, webhookOrderCode)
	if err != nil || tx == nil {
		t.Fatalf("reload transaction: %v", err)
	}
	if tx.Status != depositstatus.Success {
		t.Fatalf("expected transaction success, got %v", tx.Status)
	}

	// Subscription upgraded to the purchased plan.
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	sub, err := subRepo.GetByUserID(ctx, userID)
	if err != nil || sub == nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if sub.CurrentPlanCode != webhookPremiumPlanSlug {
		t.Fatalf("expected subscription upgraded to %q, got %q", webhookPremiumPlanSlug, sub.CurrentPlanCode)
	}

	// Idempotency: replaying the same webhook must not double-apply.
	if err := uc.ProcessWebhook(ctx, payload, "sig"); err != nil {
		t.Fatalf("replayed webhook must be a no-op success, got %v", err)
	}
	var eventCount, statementCount int64
	dbConn.Model(&entities.ProviderPaymentEvent{}).Count(&eventCount)
	dbConn.Model(&entities.WalletStatement{}).Count(&statementCount)
	if eventCount != 1 {
		t.Fatalf("expected exactly one provider payment event, got %d", eventCount)
	}
	if statementCount != 0 {
		t.Fatalf("direct purchase upgrade must not credit wallet, got %d statements", statementCount)
	}
}

func TestIntegrationProcessWebhookRejectsAmountMismatch(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	seedWebhookScenario(t, dbConn)
	uc, _ := newWebhookUseCase(dbConn)

	err := uc.ProcessWebhook(context.Background(), buildWebhookPayload(webhookOrderCode, "99001"), "sig")
	if err == nil {
		t.Fatal("expected amount mismatch rejection, got nil")
	}
}

func TestIntegrationProcessWebhookSkipsUnsuccessfulEvents(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	seedWebhookScenario(t, dbConn)
	uc, _ := newWebhookUseCase(dbConn)

	raw := []byte(`{"code":"01","desc":"failed","success":false,"data":{"orderCode":` + strconv.FormatInt(webhookOrderCode, 10) + `,"amount":"99000","reference":"ref-fail","currency":"VND","code":"01","desc":"failed"},"signature":"x"}`)
	if err := uc.ProcessWebhook(context.Background(), raw, "sig"); err != nil {
		t.Fatalf("unsuccessful webhook must be acknowledged, got %v", err)
	}
	txRepo := persistence.NewDepositTransactionRepository(dbConn)
	tx, err := txRepo.GetByOrderCode(context.Background(), webhookOrderCode)
	if err != nil || tx == nil {
		t.Fatalf("reload transaction: %v", err)
	}
	if tx.Status != depositstatus.Pending {
		t.Fatalf("expected transaction untouched for failed webhook, got %v", tx.Status)
	}
}

func TestIntegrationProcessWebhookAcksUnknownOrderCode(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	uc, _ := newWebhookUseCase(dbConn)
	ctx := context.Background()

	// Mirrors the payOS webhook-URL test sample: orderCode 123 is never ours.
	if err := uc.ProcessWebhook(ctx, buildWebhookPayload(webhookUnknownOrderCode, "3000"), "sig"); err != nil {
		t.Fatalf("webhook for an unknown orderCode must be acknowledged, got %v", err)
	}

	var inbox entities.ProviderWebhookInbox
	if err := dbConn.Where("order_code = ?", webhookUnknownOrderCode).First(&inbox).Error; err != nil {
		t.Fatalf("expected an inbox row for the unknown orderCode: %v", err)
	}
	if inbox.ProcessingStatus != string(webhookprocessingstatus.InvestigationRequired) {
		t.Fatalf("expected inbox status %q, got %q", webhookprocessingstatus.InvestigationRequired, inbox.ProcessingStatus)
	}
}

// TestIntegrationPayOSSettlementPublishesExactlyOnePlanChangeEvent is 025 T116's
// contribution to FR-095: proof that the plan-change event is published on the
// GATEWAY branch of the purchase path, i.e. `purchase_workflow.go:120`.
//
// The purchase path already had one proof — `purchase/plan_change_event_test.go`
// covers the WALLET branch at `purchase/wallet.go:144` with fakes that always run
// in `make test-race`. The gateway branch is a separate call site in a separate
// function reached only when a real payment settles, and it had no coverage at
// all. FR-095 names the failure mode this leaves open: a branch that publishes
// nothing "does not report an error anywhere". `ProcessWebhook` would still return
// nil, the subscription would still be upgraded, the customer would still have
// paid — and the AI service would never hear about the plan they bought.
//
// So the assertions here are COUNTS, never the absence of an error: exactly one
// plan-change history row and exactly one `subscription.plan.changed` on the
// topic. The row is read back out of the real `user_subscription_events` table
// rather than counted in a fake, because the history row is what the outbox
// event's `sourceEventId` points at and a row that was never persisted would
// leave the receiver holding a dangling id.
//
// The plan deliberately ends up on the happy publish branch and only that branch:
// the seeded subscription has a nil `ExpiresAt` and a nil `FallbackPlanID`, so the
// lifetime-fallback publish at `purchase_workflow.go:81` is not entered, and the
// seeded plan tier is above the current one, so the wallet-credit branch at `:90`
// is not entered either. One publish call site, therefore "exactly one" is a
// meaningful number rather than an accident of two branches firing.
//
// This sends no email. `integrationNotif` is a fake with no transport, and the
// use case is built from `config.Config{}` rather than the loaded yaml, so even
// though the settlement does enter its payment-success notification branch the
// message cannot leave the process. Zero outbound mail, as the constitution's cap
// on a verification run requires.
func TestIntegrationPayOSSettlementPublishesExactlyOnePlanChangeEvent(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	userID := seedWebhookScenario(t, dbConn)
	uc, publisher := newWebhookUseCase(dbConn)
	ctx := context.Background()

	if err := uc.ProcessWebhook(ctx, buildWebhookPayload(webhookOrderCode, "99000"), "sig"); err != nil {
		t.Fatalf("webhook settlement failed: %v", err)
	}

	// The settlement really upgraded the plan. Without this the counts below would
	// be satisfied by a webhook that did nothing at all, which is a different test.
	sub, err := persistence.NewUserSubscriptionRepository(dbConn).GetByUserID(ctx, userID)
	if err != nil || sub == nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if sub.CurrentPlanCode != webhookPremiumPlanSlug {
		t.Fatalf("expected subscription upgraded to %q, got %q", webhookPremiumPlanSlug, sub.CurrentPlanCode)
	}

	// FR-095, history half: exactly one row, carrying the transition this
	// settlement actually performed and pointing at the transaction that paid for it.
	rows := webhookPlanChangeHistory(t, dbConn)
	if len(rows) != 1 {
		t.Fatalf("FR-095: a settled PayOS purchase must write exactly one plan-change history row, got %d", len(rows))
	}
	if rows[0].EventType != string(subscriptioneventtype.ACTIVATED) {
		t.Fatalf("expected history row type %q, got %q", subscriptioneventtype.ACTIVATED, rows[0].EventType)
	}
	if rows[0].UserID != userID {
		t.Fatalf("expected history row for user %s, got %s", userID, rows[0].UserID)
	}
	if rows[0].SourceDepositTransactionID == nil {
		t.Fatal("expected the history row to name the deposit transaction it came from")
	}
	tx, err := persistence.NewDepositTransactionRepository(dbConn).GetByOrderCode(ctx, webhookOrderCode)
	if err != nil || tx == nil {
		t.Fatalf("reload transaction: %v", err)
	}
	if *rows[0].SourceDepositTransactionID != tx.ID {
		t.Fatalf("expected history row to reference transaction %s, got %s", tx.ID, *rows[0].SourceDepositTransactionID)
	}

	// FR-095, event half: exactly one message, on the right topic, describing the
	// same transition. A count of zero here is the failure FR-095 exists to catch,
	// and it would otherwise be invisible.
	if len(publisher.events) != 1 {
		t.Fatalf("FR-095: a settled PayOS purchase must publish exactly one subscription.plan.changed, got %d", len(publisher.events))
	}
	if publisher.events[0].topic != eventconstants.TopicSubscriptionPlanChanged {
		t.Fatalf("expected topic %q, got %q", eventconstants.TopicSubscriptionPlanChanged, publisher.events[0].topic)
	}
	if got := publishedWebhookEventType(t, publisher.events[0].payload); got != string(subscriptioneventtype.ACTIVATED) {
		t.Fatalf("expected published event type %q, got %q", subscriptioneventtype.ACTIVATED, got)
	}

	// Exactly one has to mean exactly one even when the gateway retries. A replay
	// that produced a second row or a second message would double-grant downstream,
	// so the count is re-checked rather than assumed.
	if err := uc.ProcessWebhook(ctx, buildWebhookPayload(webhookOrderCode, "99000"), "sig"); err != nil {
		t.Fatalf("replayed webhook must be a no-op success, got %v", err)
	}
	if rows := webhookPlanChangeHistory(t, dbConn); len(rows) != 1 {
		t.Fatalf("FR-095: a replayed settlement must not add history, got %d rows", len(rows))
	}
	if len(publisher.events) != 1 {
		t.Fatalf("FR-095: a replayed settlement must not publish again, got %d messages", len(publisher.events))
	}
}

// TestIntegrationCleanupLeavesForeignRowsUntouched is the regression guard for
// 025 D-02: cleanup must never delete a row this file did not create. The old
// `WHERE 1 = 1` sweep would have removed the sentinel below, which is exactly
// how a shared database could be emptied by a test run.
func TestIntegrationCleanupLeavesForeignRowsUntouched(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)

	otherUser := uuid.MustParse("a0000000-0000-4000-8000-000000000031")
	ensureWebhookUser(t, dbConn, otherUser, "webhook-sentinel", "webhook-sentinel@example.com")

	// Reuse the seeded default free plan: a second active free plan is rejected by
	// ux_single_active_default_free_plan (see seedWebhookScenario).
	freePlan, err := persistence.NewSubscriptionPlanRepository(dbConn).GetDefaultPlan(context.Background())
	if err != nil || freePlan == nil {
		t.Fatalf("default free plan missing from test database (apply database/init/03-seed.sql): %v", err)
	}
	sentinelSub := &entities.UserSubscription{
		UserID: otherUser, SubscriptionPlanID: freePlan.ID,
		CurrentPlanCode: freePlan.Slug, CurrentPlanKind: plankind.DefaultFree,
		StartedAt: time.Now().UTC(),
	}
	if err := persistence.NewUserSubscriptionRepository(dbConn).Create(context.Background(), sentinelSub); err != nil {
		t.Fatalf("seed sentinel subscription: %v", err)
	}

	cleanupWebhookTables(t, dbConn)

	var count int64
	if err := dbConn.Model(&entities.UserSubscription{}).
		Where("user_id = ?", otherUser).Count(&count).Error; err != nil {
		t.Fatalf("count sentinel subscription: %v", err)
	}
	if count != 1 {
		t.Fatalf("cleanup deleted a row it did not create: sentinel subscription count = %d, want 1", count)
	}

	if err := dbConn.Exec("DELETE FROM subscription.user_subscriptions WHERE user_id = ?", otherUser).Error; err != nil {
		t.Fatalf("teardown sentinel subscription: %v", err)
	}
}

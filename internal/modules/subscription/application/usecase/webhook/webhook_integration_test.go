package webhook

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"smart-wardrobe-be/internal/config"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	ucInterfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/internal/shared/infrastructure/db"

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

// ---- Helpers ----

func openWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping payment webhook integration test")
	}
	gin.SetMode(gin.TestMode)
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

func cleanupWebhookTables(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	for _, model := range []any{
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

const webhookOrderCode = int64(777001)

func webhookPtr[T any](v T) *T { return &v }

func seedWebhookScenario(t *testing.T, dbConn *gorm.DB) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	txRepo := persistence.NewDepositTransactionRepository(dbConn)

	freePlan := &entities.SubscriptionPlan{Slug: "free-webhook", Name: "Free", IsActive: true, Price: decimal.Zero, PlanKind: plankind.DefaultFree}
	if err := planRepo.Create(ctx, freePlan); err != nil {
		t.Fatalf("seed free plan: %v", err)
	}
	premiumPlan := &entities.SubscriptionPlan{Slug: "premium-webhook", Name: "Premium Webhook", IsActive: true, Price: decimal.NewFromInt(99000), TierRank: 1, PlanKind: plankind.Finite, DurationDays: webhookPtr(30), AiOutfitDailyQuota: 10, AiChatDailyQuota: 20}
	if err := planRepo.Create(ctx, premiumPlan); err != nil {
		t.Fatalf("seed premium plan: %v", err)
	}
	userID := uuid.New()
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
		PlanCode:              webhookPtr("premium-webhook"),
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
	return userID
}

func buildWebhookPayload(orderCode int64, amount string) []byte {
	return []byte(`{"code":"00","desc":"success","success":true,"data":{"orderCode":` + strconv.FormatInt(orderCode, 10) +
		`,"amount":"` + amount + `","description":"premium","accountNumber":"123","reference":"ref-` +
		strconv.FormatInt(orderCode, 10) + `","transactionDateTime":"2026-09-06T10:00:00+07:00","currency":"VND","paymentLinkId":"plink-` +
		strconv.FormatInt(orderCode, 10) + `","code":"00","desc":"success"},"signature":"ignored-by-fake"}`)
}

func newWebhookUseCase(dbConn *gorm.DB) ucInterfaces.IPaymentWebhookUseCase {
	cfg := config.Config{}
	cfg.Database.TimeZone = "Asia/Ho_Chi_Minh"
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
	)
}

// ---- Tests ----

func TestIntegrationProcessWebhookAppliesDirectPurchaseIdempotently(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	userID := seedWebhookScenario(t, dbConn)
	uc := newWebhookUseCase(dbConn)
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
		t.Fatalf("expected transaction success, got %q", tx.Status)
	}

	// Subscription upgraded to the purchased plan.
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	sub, err := subRepo.GetByUserID(ctx, userID)
	if err != nil || sub == nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-webhook" {
		t.Fatalf("expected subscription upgraded to premium-webhook, got %q", sub.CurrentPlanCode)
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
	uc := newWebhookUseCase(dbConn)

	err := uc.ProcessWebhook(context.Background(), buildWebhookPayload(webhookOrderCode, "99001"), "sig")
	if err == nil {
		t.Fatal("expected amount mismatch rejection, got nil")
	}
}

func TestIntegrationProcessWebhookSkipsUnsuccessfulEvents(t *testing.T) {
	dbConn := openWebhookTestDB(t)
	cleanupWebhookTables(t, dbConn)
	seedWebhookScenario(t, dbConn)
	uc := newWebhookUseCase(dbConn)

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
		t.Fatalf("expected transaction untouched for failed webhook, got %q", tx.Status)
	}
}

package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubDepositRepo struct {
	tx      *entities.DepositTransaction
	updated int
}

func (s *stubDepositRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (s *stubDepositRepo) GetAll(ctx context.Context) ([]*entities.DepositTransaction, error) {
	return nil, nil
}
func (s *stubDepositRepo) Create(ctx context.Context, entity *entities.DepositTransaction) error {
	return nil
}
func (s *stubDepositRepo) Update(ctx context.Context, entity *entities.DepositTransaction) error {
	s.updated++
	return nil
}
func (s *stubDepositRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (s *stubDepositRepo) GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (s *stubDepositRepo) GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	return s.tx, nil
}
func (s *stubDepositRepo) GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	return s.tx, nil
}
func (s *stubDepositRepo) ListAdmin(ctx context.Context, filter repositories.AdminDepositTransactionFilter, page, limit int) ([]*entities.DepositTransaction, int64, error) {
	return nil, 0, nil
}
func (s *stubDepositRepo) HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error) {
	return false, nil
}
func (s *stubDepositRepo) GetActiveDirectPurchase(ctx context.Context, userID uuid.UUID) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (s *stubDepositRepo) ClaimReconciliationCandidates(ctx context.Context, limit int, lease time.Duration) ([]*repositories.ClaimedDepositTransaction, error) {
	return nil, nil
}
func (s *stubDepositRepo) UpdateWithToken(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
	return 1, nil
}
func (s *stubDepositRepo) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return nil, nil
}
func (s *stubDepositRepo) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return nil, nil
}
func (s *stubDepositRepo) BatchUpdate(ctx context.Context, items []*entities.DepositTransaction) error {
	return nil
}
func (s *stubDepositRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (s *stubDepositRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.DepositTransaction, error) {
	return nil, nil
}

type stubGateway struct {
	info  *payment.PaymentLinkInfo
	err   error
	calls int
}

func (g *stubGateway) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutSessionReq) (*payment.CheckoutSessionResult, error) {
	return nil, nil
}
func (g *stubGateway) VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) error {
	return nil
}
func (g *stubGateway) GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
	g.calls++
	return g.info, g.err
}
func (g *stubGateway) CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error {
	return nil
}

type stubUnitOfWork struct{}

func (s *stubUnitOfWork) Execute(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

func newTestInbox() *entities.ProviderWebhookInbox {
	return &entities.ProviderWebhookInbox{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
	}
}

func TestMarkCancelledUpdatesActiveTransaction(t *testing.T) {
	tx := &entities.DepositTransaction{OrderCode: 999, Status: depositstatus.Pending}
	repo := &stubDepositRepo{tx: tx}
	cancelledAt := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	uc := &PaymentWebhookUseCase{depositTxRepo: repo, uow: &stubUnitOfWork{}, log: zap.NewNop()}

	info := &payment.PaymentLinkInfo{
		OrderCode:          999,
		Status:             payment.ProviderCancelled,
		CancelledAt:        &cancelledAt,
		CancellationReason: "user cancelled",
	}
	if err := uc.MarkCancelled(context.Background(), info); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Status != depositstatus.Cancelled {
		t.Fatalf("expected status Cancelled, got %v", tx.Status)
	}
	if tx.CancelledAt == nil || !tx.CancelledAt.Equal(cancelledAt) {
		t.Fatalf("expected cancelledAt %v, got %v", cancelledAt, tx.CancelledAt)
	}
	if tx.FailureReason == nil || *tx.FailureReason != "user cancelled" {
		t.Fatalf("expected failure reason from provider, got %v", tx.FailureReason)
	}
	if tx.ProviderStatus == nil || *tx.ProviderStatus != string(payment.ProviderCancelled) {
		t.Fatalf("expected provider status CANCELLED, got %v", tx.ProviderStatus)
	}
	if tx.NextReconciliationAt != nil {
		t.Fatal("expected next reconciliation to be cleared")
	}
	if repo.updated != 1 {
		t.Fatalf("expected one update, got %d", repo.updated)
	}
}

func TestMarkCancelledNeverDowngradesSettledTransaction(t *testing.T) {
	tx := &entities.DepositTransaction{OrderCode: 999, Status: depositstatus.Success}
	repo := &stubDepositRepo{tx: tx}
	uc := &PaymentWebhookUseCase{depositTxRepo: repo, uow: &stubUnitOfWork{}, log: zap.NewNop()}

	if err := uc.MarkCancelled(context.Background(), &payment.PaymentLinkInfo{OrderCode: 999}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Status != depositstatus.Success {
		t.Fatalf("expected settled status to be preserved, got %v", tx.Status)
	}
	if repo.updated != 0 {
		t.Fatalf("expected no update, got %d", repo.updated)
	}
}

func TestMarkCancelledUsesDefaultReasonWhenProviderOmitsIt(t *testing.T) {
	tx := &entities.DepositTransaction{OrderCode: 999, Status: depositstatus.Pending}
	repo := &stubDepositRepo{tx: tx}
	uc := &PaymentWebhookUseCase{depositTxRepo: repo, uow: &stubUnitOfWork{}, log: zap.NewNop()}

	if err := uc.MarkCancelled(context.Background(), &payment.PaymentLinkInfo{OrderCode: 999}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.FailureReason == nil || *tx.FailureReason != defaultCancellationReason {
		t.Fatalf("expected default reason %q, got %v", defaultCancellationReason, tx.FailureReason)
	}
}

func TestResolveUnsuccessfulWebhookMarksCancelled(t *testing.T) {
	tx := &entities.DepositTransaction{OrderCode: 999, Status: depositstatus.Pending}
	repo := &stubDepositRepo{tx: tx}
	gateway := &stubGateway{info: &payment.PaymentLinkInfo{OrderCode: 999, Status: payment.ProviderCancelled}}
	inboxRepo := &captureInboxRepo{}
	uc := &PaymentWebhookUseCase{
		depositTxRepo:  repo,
		paymentGateway: gateway,
		inboxRepo:      inboxRepo,
		uow:            &stubUnitOfWork{},
		log:            zap.NewNop(),
	}
	payload := &WebhookPayload{Code: "01", Data: WebhookPayloadData{OrderCode: 999, Code: "01"}}

	if err := uc.resolveUnsuccessfulWebhook(context.Background(), payload, newTestInbox()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Status != depositstatus.Cancelled {
		t.Fatalf("expected status Cancelled, got %v", tx.Status)
	}
	if inboxRepo.processed != 1 {
		t.Fatalf("expected the inbox row to be processed, got %d", inboxRepo.processed)
	}
}

func TestResolveUnsuccessfulWebhookAcknowledgesPendingStatus(t *testing.T) {
	tx := &entities.DepositTransaction{OrderCode: 999, Status: depositstatus.Pending}
	repo := &stubDepositRepo{tx: tx}
	gateway := &stubGateway{info: &payment.PaymentLinkInfo{OrderCode: 999, Status: payment.ProviderPending}}
	inboxRepo := &captureInboxRepo{}
	uc := &PaymentWebhookUseCase{
		depositTxRepo:  repo,
		paymentGateway: gateway,
		inboxRepo:      inboxRepo,
		uow:            &stubUnitOfWork{},
		log:            zap.NewNop(),
	}
	payload := &WebhookPayload{Code: "01", Data: WebhookPayloadData{OrderCode: 999, Code: "01"}}

	if err := uc.resolveUnsuccessfulWebhook(context.Background(), payload, newTestInbox()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Status != depositstatus.Pending {
		t.Fatalf("expected status to stay Pending, got %v", tx.Status)
	}
	if inboxRepo.investigated != 1 {
		t.Fatalf("expected the inbox row to be marked for investigation, got %d", inboxRepo.investigated)
	}
}

func TestResolveUnsuccessfulWebhookReturnsLookupError(t *testing.T) {
	tx := &entities.DepositTransaction{OrderCode: 999, Status: depositstatus.Pending}
	repo := &stubDepositRepo{tx: tx}
	gateway := &stubGateway{err: errors.New("network error")}
	inboxRepo := &captureInboxRepo{}
	uc := &PaymentWebhookUseCase{
		depositTxRepo:  repo,
		paymentGateway: gateway,
		inboxRepo:      inboxRepo,
		uow:            &stubUnitOfWork{},
		log:            zap.NewNop(),
	}
	payload := &WebhookPayload{Code: "01", Data: WebhookPayloadData{OrderCode: 999, Code: "01"}}

	if err := uc.resolveUnsuccessfulWebhook(context.Background(), payload, newTestInbox()); err == nil {
		t.Fatal("expected the provider lookup error to be returned so payOS retries")
	}
	if inboxRepo.retried != 1 {
		t.Fatalf("expected the inbox row to be marked for retry, got %d", inboxRepo.retried)
	}
}

func TestResolveUnsuccessfulWebhookAcknowledgesUnknownOrder(t *testing.T) {
	repo := &stubDepositRepo{}
	gateway := &stubGateway{info: &payment.PaymentLinkInfo{Status: payment.ProviderCancelled}}
	inboxRepo := &captureInboxRepo{}
	uc := &PaymentWebhookUseCase{
		depositTxRepo:  repo,
		paymentGateway: gateway,
		inboxRepo:      inboxRepo,
		uow:            &stubUnitOfWork{},
		log:            zap.NewNop(),
	}
	payload := &WebhookPayload{Code: "01", Data: WebhookPayloadData{OrderCode: 999, Code: "01"}}

	if err := uc.resolveUnsuccessfulWebhook(context.Background(), payload, newTestInbox()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateway.calls != 0 {
		t.Fatalf("expected no provider lookup for an unknown order, got %d", gateway.calls)
	}
	if inboxRepo.processed != 1 {
		t.Fatalf("expected the inbox row to be processed, got %d", inboxRepo.processed)
	}
}

func TestIsTerminalDepositStatus(t *testing.T) {
	terminal := []depositstatus.DepositStatus{
		depositstatus.Success,
		depositstatus.FailedLegacy,
		depositstatus.CreationFailed,
		depositstatus.Cancelled,
		depositstatus.Expired,
		depositstatus.Refunded,
	}
	for _, status := range terminal {
		if !isTerminalDepositStatus(status) {
			t.Fatalf("expected %v to be terminal", status)
		}
	}
	active := []depositstatus.DepositStatus{
		depositstatus.Pending,
		depositstatus.Creating,
		depositstatus.ReconciliationRequired,
		depositstatus.Reconciling,
		depositstatus.InvestigationRequired,
	}
	for _, status := range active {
		if isTerminalDepositStatus(status) {
			t.Fatalf("expected %v not to be terminal", status)
		}
	}
}

package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Fake repositories and services for testing
type fakeDepositTransactionRepository struct {
	updateWithTokenFunc func(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error)
	getByOrderCodeFunc  func(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
}

func (f *fakeDepositTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTransactionRepository) GetAll(ctx context.Context) ([]*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTransactionRepository) Create(ctx context.Context, entity *entities.DepositTransaction) error {
	return nil
}
func (f *fakeDepositTransactionRepository) Update(ctx context.Context, entity *entities.DepositTransaction) error {
	return nil
}
func (f *fakeDepositTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (f *fakeDepositTransactionRepository) GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTransactionRepository) GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	if f.getByOrderCodeFunc != nil {
		return f.getByOrderCodeFunc(ctx, orderCode)
	}
	return nil, nil
}
func (f *fakeDepositTransactionRepository) GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTransactionRepository) HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeDepositTransactionRepository) GetActiveDirectPurchase(ctx context.Context, userID uuid.UUID) (*entities.DepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTransactionRepository) ClaimReconciliationCandidates(ctx context.Context, limit int, lease time.Duration) ([]*repositories.ClaimedDepositTransaction, error) {
	return nil, nil
}
func (f *fakeDepositTransactionRepository) UpdateWithToken(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
	if f.updateWithTokenFunc != nil {
		return f.updateWithTokenFunc(ctx, orderCode, token, updates)
	}
	return 1, nil
}

type fakePaymentGatewayService struct {
	getPaymentLinkInfoFunc func(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error)
	cancelPaymentLinkFunc  func(ctx context.Context, orderCode int64, reason string) error
}

func (f *fakePaymentGatewayService) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutSessionReq) (*payment.CheckoutSessionResult, error) {
	return nil, nil
}
func (f *fakePaymentGatewayService) VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) (map[string]any, error) {
	return nil, nil
}
func (f *fakePaymentGatewayService) GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
	if f.getPaymentLinkInfoFunc != nil {
		return f.getPaymentLinkInfoFunc(ctx, orderCode)
	}
	return nil, nil
}
func (f *fakePaymentGatewayService) CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error {
	if f.cancelPaymentLinkFunc != nil {
		return f.cancelPaymentLinkFunc(ctx, orderCode, reason)
	}
	return nil
}

type fakePaymentWebhookUseCase struct {
	completeVerifiedPaymentFunc func(ctx context.Context, info *payment.PaymentLinkInfo) error
}

func (f *fakePaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	return nil
}
func (f *fakePaymentWebhookUseCase) CompleteVerifiedPayment(ctx context.Context, info *payment.PaymentLinkInfo) error {
	if f.completeVerifiedPaymentFunc != nil {
		return f.completeVerifiedPaymentFunc(ctx, info)
	}
	return nil
}

type fakeUnitOfWork struct{}

func (f *fakeUnitOfWork) Execute(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

type fakeLogger struct{}

func (f *fakeLogger) Info(message string, fields ...zap.Field)  {}
func (f *fakeLogger) Error(message string, fields ...zap.Field) {}
func (f *fakeLogger) Warn(message string, fields ...zap.Field)  {}
func (f *fakeLogger) Debug(message string, fields ...zap.Field) {}
func (f *fakeLogger) Fatal(message string, fields ...zap.Field) {}

func TestPaymentReconciliationWorker(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		PayOS: config.PayOS{
			ReconciliationMaxAttempts: 5,
			ReconciliationMaxAgeHours: 24,
		},
	}

	t.Run("ProviderPaid - completes payment", func(t *testing.T) {
		completedCalled := false
		webhook := &fakePaymentWebhookUseCase{
			completeVerifiedPaymentFunc: func(ctx context.Context, info *payment.PaymentLinkInfo) error {
				completedCalled = true
				return nil
			},
		}

		gateway := &fakePaymentGatewayService{
			getPaymentLinkInfoFunc: func(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
				return &payment.PaymentLinkInfo{
					OrderCode:     orderCode,
					Status:        payment.ProviderPaid,
					PaymentLinkID: "link-id",
					CheckoutURL:   "checkout-url",
				}, nil
			},
		}

		repo := &fakeDepositTransactionRepository{}

		w := &PaymentReconciliationWorker{
			repo:       repo,
			gateway:    gateway,
			completion: webhook,
			uow:        &fakeUnitOfWork{},
			log:        &fakeLogger{},
			cfg:        cfg,
		}

		claimed := &repositories.ClaimedDepositTransaction{
			ProcessingToken: uuid.New(),
			Transaction: &entities.DepositTransaction{
				OrderCode: 12345,
			},
		}

		w.reconcile(ctx, claimed)

		if !completedCalled {
			t.Fatal("expected CompleteVerifiedPayment to be called")
		}
	})

	t.Run("ProviderPending - expired link gets cancelled", func(t *testing.T) {
		cancelCalled := false
		gateway := &fakePaymentGatewayService{
			getPaymentLinkInfoFunc: func(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
				return &payment.PaymentLinkInfo{
					OrderCode:     orderCode,
					Status:        payment.ProviderPending,
					PaymentLinkID: "link-id",
				}, nil
			},
			cancelPaymentLinkFunc: func(ctx context.Context, orderCode int64, reason string) error {
				cancelCalled = true
				return nil
			},
		}

		finishedStatus := depositstatus.DepositStatus(0)
		repo := &fakeDepositTransactionRepository{
			updateWithTokenFunc: func(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
				if s, ok := updates["status"]; ok {
					finishedStatus = s.(depositstatus.DepositStatus)
				}
				return 1, nil
			},
		}

		w := &PaymentReconciliationWorker{
			repo:       repo,
			gateway:    gateway,
			completion: &fakePaymentWebhookUseCase{},
			uow:        &fakeUnitOfWork{},
			log:        &fakeLogger{},
			cfg:        cfg,
		}

		past := time.Now().Add(-10 * time.Minute)
		claimed := &repositories.ClaimedDepositTransaction{
			ProcessingToken: uuid.New(),
			Transaction: &entities.DepositTransaction{
				OrderCode: 12345,
				ExpiresAt: &past,
			},
		}

		w.reconcile(ctx, claimed)

		if !cancelCalled {
			t.Fatal("expected CancelPaymentLink to be called for expired transaction")
		}
		if finishedStatus != depositstatus.Expired {
			t.Fatalf("expected transaction status to be set to Expired, got %v", finishedStatus)
		}
	})

	t.Run("ProviderPending - recoverable pending recovers status", func(t *testing.T) {
		gateway := &fakePaymentGatewayService{
			getPaymentLinkInfoFunc: func(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
				return &payment.PaymentLinkInfo{
					OrderCode:     orderCode,
					Status:        payment.ProviderPending,
					PaymentLinkID: "link-id",
					CheckoutURL:   "checkout-url",
				}, nil
			},
		}

		var updatedStatus depositstatus.DepositStatus
		repo := &fakeDepositTransactionRepository{
			updateWithTokenFunc: func(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
				if s, ok := updates["status"]; ok {
					updatedStatus = s.(depositstatus.DepositStatus)
				}
				return 1, nil
			},
		}

		w := &PaymentReconciliationWorker{
			repo:       repo,
			gateway:    gateway,
			completion: &fakePaymentWebhookUseCase{},
			uow:        &fakeUnitOfWork{},
			log:        &fakeLogger{},
			cfg:        cfg,
		}

		future := time.Now().Add(10 * time.Minute)
		claimed := &repositories.ClaimedDepositTransaction{
			ClaimedFromStatus: depositstatus.Creating,
			ProcessingToken:   uuid.New(),
			Transaction: &entities.DepositTransaction{
				OrderCode: 12345,
				ExpiresAt: &future,
			},
		}

		w.reconcile(ctx, claimed)

		if updatedStatus != depositstatus.Pending {
			t.Fatalf("expected transaction to be recovered to Pending, got %v", updatedStatus)
		}
	})

	t.Run("ProviderCancelled - finishes cancelled status", func(t *testing.T) {
		gateway := &fakePaymentGatewayService{
			getPaymentLinkInfoFunc: func(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
				return &payment.PaymentLinkInfo{
					OrderCode: orderCode,
					Status:    payment.ProviderCancelled,
				}, nil
			},
		}

		var updatedStatus depositstatus.DepositStatus
		repo := &fakeDepositTransactionRepository{
			updateWithTokenFunc: func(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
				if s, ok := updates["status"]; ok {
					updatedStatus = s.(depositstatus.DepositStatus)
				}
				return 1, nil
			},
		}

		w := &PaymentReconciliationWorker{
			repo:       repo,
			gateway:    gateway,
			completion: &fakePaymentWebhookUseCase{},
			uow:        &fakeUnitOfWork{},
			log:        &fakeLogger{},
			cfg:        cfg,
		}

		claimed := &repositories.ClaimedDepositTransaction{
			ProcessingToken: uuid.New(),
			Transaction: &entities.DepositTransaction{
				OrderCode: 12345,
			},
		}

		w.reconcile(ctx, claimed)

		if updatedStatus != depositstatus.Cancelled {
			t.Fatalf("expected transaction to be set to Cancelled, got %v", updatedStatus)
		}
	})

	t.Run("reconcile error - defers retry", func(t *testing.T) {
		gateway := &fakePaymentGatewayService{
			getPaymentLinkInfoFunc: func(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
				return nil, errors.New("network error")
			},
		}

		token := uuid.New()
		var lastErrorCode string
		repo := &fakeDepositTransactionRepository{
			getByOrderCodeFunc: func(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
				return &entities.DepositTransaction{
					AuditableEntity: entities.AuditableEntity{
						BaseEntity: entities.BaseEntity{
							CreatedAt: time.Now().UTC(),
						},
					},
					OrderCode:       orderCode,
					ProcessingToken: &token,
					Status:          depositstatus.Pending,
				}, nil
			},
			updateWithTokenFunc: func(ctx context.Context, orderCode int64, tkn uuid.UUID, updates map[string]any) (int64, error) {
				if c, ok := updates["last_provider_error_code"]; ok {
					if cStr, ok := c.(*string); ok {
						lastErrorCode = *cStr
					}
				}
				return 1, nil
			},
		}

		w := &PaymentReconciliationWorker{
			repo:       repo,
			gateway:    gateway,
			completion: &fakePaymentWebhookUseCase{},
			uow:        &fakeUnitOfWork{},
			log:        &fakeLogger{},
			cfg:        cfg,
		}

		claimed := &repositories.ClaimedDepositTransaction{
			ProcessingToken: token,
			Transaction: &entities.DepositTransaction{
				OrderCode: 12345,
			},
		}

		w.reconcile(ctx, claimed)

		if lastErrorCode != "PROVIDER_LOOKUP_ERROR" {
			t.Fatalf("expected error code to be PROVIDER_LOOKUP_ERROR, got %s", lastErrorCode)
		}
	})
}

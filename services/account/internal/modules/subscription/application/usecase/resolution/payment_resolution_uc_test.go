package resolution_test

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/resolution"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDepositRepo struct {
	mock.Mock
	repositories.IDepositTransactionRepository
}

func (m *mockDepositRepo) GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	args := m.Called(ctx, orderCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DepositTransaction), args.Error(1)
}

func (m *mockDepositRepo) Update(ctx context.Context, tx *entities.DepositTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

type mockWebhookUC struct {
	mock.Mock
	uc_interfaces.IPaymentWebhookUseCase
}

func (m *mockWebhookUC) CompleteVerifiedPayment(ctx context.Context, evidence *payment.PaymentLinkInfo) error {
	args := m.Called(ctx, evidence)
	return args.Error(0)
}

type directUoW struct{}

func (u *directUoW) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// allStatuses is every status the admin API can report (029 data-model §2). The
// matrix tests below run the WHOLE list rather than a few samples: the guard this
// replaces was written as "not success", and the statuses it silently accepted
// were the ones nobody happened to test.
var allStatuses = []depositstatus.DepositStatus{
	depositstatus.Pending,
	depositstatus.Success,
	depositstatus.FailedLegacy,
	depositstatus.Creating,
	depositstatus.ReconciliationRequired,
	depositstatus.Reconciling,
	depositstatus.CreationFailed,
	depositstatus.Cancelled,
	depositstatus.Expired,
	depositstatus.InvestigationRequired,
	depositstatus.Refunded,
}

var statusNames = map[depositstatus.DepositStatus]string{
	depositstatus.Pending:                "pending",
	depositstatus.Success:                "success",
	depositstatus.FailedLegacy:           "failed_legacy",
	depositstatus.Creating:               "creating",
	depositstatus.ReconciliationRequired: "reconciliation_required",
	depositstatus.Reconciling:            "reconciling",
	depositstatus.CreationFailed:         "creation_failed",
	depositstatus.Cancelled:              "cancelled",
	depositstatus.Expired:                "expired",
	depositstatus.InvestigationRequired:  "investigation_required",
	depositstatus.Refunded:               "refunded",
}

// TestPaymentResolutionUseCase_ResolveAsPaid(t *testing.T) — kept from the
// original suite: evidence is mandatory, because an administrator asserting "the
// money arrived" without a bank reference and a link id is not evidence.
func TestPaymentResolutionUseCase_ResolveAsPaid(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(mockDepositRepo)
	mockWebhook := new(mockWebhookUC)
	uow := &directUoW{}

	uc := resolution.NewPaymentResolutionUseCase(mockRepo, mockWebhook, uow)

	t.Run("fails when evidence is missing or incomplete", func(t *testing.T) {
		err := uc.ResolveAsPaid(ctx, uc_interfaces.ManualPaymentResolutionInput{
			OrderCode: 12345,
			Evidence:  nil,
		})
		assert.Error(t, err)

		err = uc.ResolveAsPaid(ctx, uc_interfaces.ManualPaymentResolutionInput{
			OrderCode: 12345,
			Evidence: &payment.PaymentLinkInfo{
				Reference:     "",
				PaymentLinkID: "link_1",
			},
		})
		assert.Error(t, err)
	})

	t.Run("succeeds when evidence is complete", func(t *testing.T) {
		evidence := &payment.PaymentLinkInfo{
			Reference:     "FT12345",
			PaymentLinkID: "link_123",
		}
		mockWebhook.On("CompleteVerifiedPayment", ctx, evidence).Return(nil)

		err := uc.ResolveAsPaid(ctx, uc_interfaces.ManualPaymentResolutionInput{
			OrderCode: 12345,
			Evidence:  evidence,
		})
		assert.NoError(t, err)
		mockWebhook.AssertExpectations(t)
	})
}

// TestPaymentResolutionUseCase_ResolveAsPaidMarksTheEvidencePaid is the
// regression the endpoint shipped with: the completion path is the webhook's and
// refuses any link whose status is not PAID, while this caller has typed evidence
// and no provider status at all. Without the assertion below the zero value ""
// travelled through and every resolve-paid answered 500.
func TestPaymentResolutionUseCase_ResolveAsPaidMarksTheEvidencePaid(t *testing.T) {
	ctx := context.Background()
	mockWebhook := new(mockWebhookUC)
	uc := resolution.NewPaymentResolutionUseCase(new(mockDepositRepo), mockWebhook, &directUoW{})

	var captured *payment.PaymentLinkInfo
	mockWebhook.On("CompleteVerifiedPayment", mock.Anything, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*payment.PaymentLinkInfo)
		})

	err := uc.ResolveAsPaid(ctx, uc_interfaces.ManualPaymentResolutionInput{
		OrderCode:      12345,
		ExpectedStatus: depositstatus.ReconciliationRequired,
		Reason:         "Khách đã chuyển khoản",
		Evidence: &payment.PaymentLinkInfo{
			Reference:     "FT12345",
			PaymentLinkID: "link_123",
		},
	})
	assert.NoError(t, err)
	if assert.NotNil(t, captured) {
		assert.Equal(t, payment.ProviderPaid, captured.Status,
			"resolve-paid must hand the completion path a PAID link; anything else is rejected with a 500")
	}
}

// TestPaymentResolutionUseCase_StatusMatrix is the authority for which statuses
// each manual action may start from, expressed as data. The frontend mirrors this
// table, but curl does not go through a disabled button, so both sides must agree
// and this side wins.
func TestPaymentResolutionUseCase_StatusMatrix(t *testing.T) {
	allowed := func(statuses ...depositstatus.DepositStatus) map[depositstatus.DepositStatus]bool {
		set := make(map[depositstatus.DepositStatus]bool, len(statuses))
		for _, status := range statuses {
			set[status] = true
		}
		return set
	}

	paid := allowed(
		depositstatus.Pending, depositstatus.Creating, depositstatus.Reconciling,
		depositstatus.ReconciliationRequired, depositstatus.InvestigationRequired,
		// Money can arrive after the link was cancelled, expired, or failed under
		// the legacy state machine. Manual reconciliation is the only rescue path.
		depositstatus.Cancelled, depositstatus.Expired, depositstatus.FailedLegacy,
	)
	cancelledAndRetry := allowed(
		depositstatus.Pending, depositstatus.Creating, depositstatus.Reconciling,
		depositstatus.ReconciliationRequired, depositstatus.InvestigationRequired,
	)
	creationFailed := allowed(
		depositstatus.Creating, depositstatus.ReconciliationRequired, depositstatus.InvestigationRequired,
	)

	t.Run("resolve-paid", func(t *testing.T) {
		for _, status := range allStatuses {
			status := status
			t.Run(statusNames[status], func(t *testing.T) {
				webhook := new(mockWebhookUC)
				uc := resolution.NewPaymentResolutionUseCase(new(mockDepositRepo), webhook, &directUoW{})
				if paid[status] {
					webhook.On("CompleteVerifiedPayment", mock.Anything, mock.Anything).Return(nil)
				}

				err := uc.ResolveAsPaid(context.Background(), uc_interfaces.ManualPaymentResolutionInput{
					OrderCode:      12345,
					ExpectedStatus: status,
					Reason:         "matrix",
					Evidence:       &payment.PaymentLinkInfo{Reference: "FT1", PaymentLinkID: "link_1"},
				})

				assertOutcome(t, err, paid[status])
				if !paid[status] {
					webhook.AssertNotCalled(t, "CompleteVerifiedPayment", mock.Anything, mock.Anything)
				}
			})
		}
	})

	transitions := []struct {
		name  string
		allow map[depositstatus.DepositStatus]bool
		run   func(uc uc_interfaces.IPaymentResolutionUseCase, status depositstatus.DepositStatus) error
	}{
		{
			name:  "resolve-cancelled",
			allow: cancelledAndRetry,
			run: func(uc uc_interfaces.IPaymentResolutionUseCase, status depositstatus.DepositStatus) error {
				return uc.ResolveAsCancelled(context.Background(), uc_interfaces.ManualPaymentResolutionInput{
					OrderCode: 12345, ExpectedStatus: status, Reason: "matrix",
				})
			},
		},
		{
			name:  "retry-investigation",
			allow: cancelledAndRetry,
			run: func(uc uc_interfaces.IPaymentResolutionUseCase, status depositstatus.DepositStatus) error {
				return uc.RetryInvestigation(context.Background(), uc_interfaces.ManualPaymentResolutionInput{
					OrderCode: 12345, ExpectedStatus: status, Reason: "matrix",
				})
			},
		},
		{
			name:  "resolve-creation-failed",
			allow: creationFailed,
			run: func(uc uc_interfaces.IPaymentResolutionUseCase, status depositstatus.DepositStatus) error {
				return uc.ResolveAsCreationFailed(context.Background(), uc_interfaces.ManualPaymentResolutionInput{
					OrderCode: 12345, ExpectedStatus: status, Reason: "matrix",
				})
			},
		},
	}

	for _, transition := range transitions {
		transition := transition
		t.Run(transition.name, func(t *testing.T) {
			for _, status := range allStatuses {
				status := status
				t.Run(statusNames[status], func(t *testing.T) {
					repo := new(mockDepositRepo)
					row := &entities.DepositTransaction{UserID: uuid.New(), OrderCode: 12345, Status: status}
					if transition.allow[status] {
						repo.On("GetByOrderCodeWithLock", mock.Anything, int64(12345)).Return(row, nil)
						repo.On("Update", mock.Anything, row).Return(nil)
					}
					uc := resolution.NewPaymentResolutionUseCase(repo, new(mockWebhookUC), &directUoW{})

					err := transition.run(uc, status)

					assertOutcome(t, err, transition.allow[status])
					if !transition.allow[status] {
						repo.AssertNotCalled(t, "GetByOrderCodeWithLock", mock.Anything, mock.Anything)
					}
				})
			}
		})
	}
}

// assertOutcome is one assertion for both directions of the matrix: allowed
// statuses must succeed, forbidden ones must fail with the Vietnamese 400 the API
// contract specifies (not a 500 and not a silent success).
func assertOutcome(t *testing.T, err error, allowed bool) {
	t.Helper()
	if allowed {
		assert.NoError(t, err)
		return
	}
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Không thể áp dụng quyết định xử lý thủ công")
	}
}

// TestPaymentResolutionUseCase_StatusMismatch — kept from the original suite: the
// row under lock is checked against the status the handler read, so a webhook or
// another administrator that moved the row in between makes the request fail
// instead of overwriting the newer state.
func TestPaymentResolutionUseCase_StatusMismatch(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(mockDepositRepo)
	mockWebhook := new(mockWebhookUC)
	uow := &directUoW{}

	uc := resolution.NewPaymentResolutionUseCase(mockRepo, mockWebhook, uow)

	t.Run("fails when current status does not match expected status", func(t *testing.T) {
		txRow := &entities.DepositTransaction{
			UserID:    uuid.New(),
			OrderCode: 12345,
			Status:    depositstatus.ReconciliationRequired,
		}
		mockRepo.On("GetByOrderCodeWithLock", ctx, int64(12345)).Return(txRow, nil)

		err := uc.RetryInvestigation(ctx, uc_interfaces.ManualPaymentResolutionInput{
			OrderCode:      12345,
			ExpectedStatus: depositstatus.InvestigationRequired,
			Reason:         "Retry test",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Không thể áp dụng quyết định xử lý thủ công")
		mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})
}

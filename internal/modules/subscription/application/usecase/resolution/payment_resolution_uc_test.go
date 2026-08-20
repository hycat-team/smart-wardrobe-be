package resolution_test

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/resolution"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"

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
			Status:    depositstatus.Cancelled,
		}
		mockRepo.On("GetByOrderCodeWithLock", ctx, int64(12345)).Return(txRow, nil)

		err := uc.RetryInvestigation(ctx, uc_interfaces.ManualPaymentResolutionInput{
			OrderCode:      12345,
			ExpectedStatus: depositstatus.InvestigationRequired,
			Reason:         "Retry test",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Không thể áp dụng quyết định xử lý thủ công")
	})
}

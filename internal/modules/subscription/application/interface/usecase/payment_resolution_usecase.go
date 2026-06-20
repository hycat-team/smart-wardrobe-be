package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
)

type ManualPaymentResolutionInput struct {
	OrderCode      int64
	ExpectedStatus depositstatus.DepositStatus
	ActorAdminID   string
	Reason         string
	Evidence       *payment.PaymentLinkInfo
}

type IPaymentResolutionUseCase interface {
	RetryInvestigation(ctx context.Context, input ManualPaymentResolutionInput) error
	ResolveAsPaid(ctx context.Context, input ManualPaymentResolutionInput) error
	ResolveAsCancelled(ctx context.Context, input ManualPaymentResolutionInput) error
	ResolveAsCreationFailed(ctx context.Context, input ManualPaymentResolutionInput) error
}

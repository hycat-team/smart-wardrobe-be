package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
)

type IPaymentWebhookUseCase interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error
	CompleteVerifiedPayment(ctx context.Context, info *payment.PaymentLinkInfo) error
	AutoRefund(ctx context.Context, orderCode int64, reason string) error
}

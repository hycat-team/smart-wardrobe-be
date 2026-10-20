package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
)

type IPaymentWebhookUseCase interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error
	CompleteVerifiedPayment(ctx context.Context, info *payment.PaymentLinkInfo) error
	// MarkCancelled records a provider-confirmed cancellation. It is idempotent
	// and never downgrades a settled transaction.
	MarkCancelled(ctx context.Context, info *payment.PaymentLinkInfo) error
	AutoRefund(ctx context.Context, orderCode int64, reason string) error
}

package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
)

type IPaymentWebhookUseCase interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error
	CompleteVerifiedPayment(ctx context.Context, info *payment.PaymentLinkInfo) error
}

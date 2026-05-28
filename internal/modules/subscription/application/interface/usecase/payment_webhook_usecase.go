package usecase

import (
	"context"
)

type IPaymentWebhookUseCase interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error
}
